package utils

import (
	"bytes"
	"crypto/tls"
	"crypto/x509"
	"encoding/base64"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"os/user"
	"runtime"
	"strings"
	"time"

	"github.com/aliyunidaas/alibaba-cloud-idaas/constants"
	"github.com/aliyunidaas/alibaba-cloud-idaas/idaaslog"
	"github.com/aliyunidaas/alibaba-cloud-idaas/utils/features"
	"github.com/pkg/errors"
)

var (
	UnsafeSkipCertificateVerification = idaaslog.IsOn(os.Getenv(constants.EnvUnsafeSkipCertificateVerification))
	RootCertificates                  = os.Getenv(constants.EnvRootCertificates)
)

const (
	HttpMethodPut  = "PUT"
	HttpMethodGet  = "GET"
	HttpMethodPost = "POST"
)

var UserAgent = GetUserAgent()

func PostHttp(postUrl string, parameters map[string]string) (int, string, error) {
	client := BuildHttpClient()
	postBody := ""
	for key, value := range parameters {
		if len(postBody) > 0 {
			postBody += "&"
		}
		postBody += url.QueryEscape(key) + "=" + url.QueryEscape(value)
	}
	req, err := http.NewRequest(HttpMethodPost, postUrl, strings.NewReader(postBody))
	if err != nil {
		return 0, "", errors.Wrapf(err, "new request: %s", postUrl)
	}
	req.Header.Set("User-Agent", UserAgent)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	resp, err := client.Do(req)
	if err != nil {
		return 0, "", errors.Wrapf(err, "do post request: %s", postUrl)
	}
	defer resp.Body.Close()
	logHttpResponse(resp)
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return 0, "", errors.Wrapf(err, "read response body: %s", postUrl)
	}
	return resp.StatusCode, string(body), nil
}

func GetHttp(getUrl string) (int, string, error) {
	client := BuildHttpClient()
	req, err := http.NewRequest(HttpMethodGet, getUrl, nil)
	if err != nil {
		return 0, "", errors.Wrapf(err, "new request: %s", getUrl)
	}
	req.Header.Set("User-Agent", UserAgent)
	resp, err := client.Do(req)
	if err != nil {
		return 0, "", errors.Wrapf(err, "do get request: %s", getUrl)
	}
	defer resp.Body.Close()
	logHttpResponse(resp)
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return 0, "", errors.Wrapf(err, "read response body: %s", getUrl)
	}
	return resp.StatusCode, string(body), nil
}

func FetchAsString(client *http.Client, method, endpoint string, headers map[string]string) (string, error) {
	body, err := Fetch(client, method, endpoint, headers)
	if err != nil {
		return "", err
	}
	return string(body), nil
}

func FetchWithBodyAsString(client *http.Client, method, endpoint string, headers map[string]string, requestBytes []byte) (string, error) {
	body, err := FetchWithBody(client, method, endpoint, headers, requestBytes)
	if err != nil {
		return "", err
	}
	return string(body), nil
}

func Fetch(client *http.Client, method, endpoint string, headers map[string]string) ([]byte, error) {
	return FetchWithBody(client, method, endpoint, headers, nil)
}

func FetchWithBody(client *http.Client, method, endpoint string, headers map[string]string, requestBytes []byte) ([]byte, error) {
	var requestBody io.Reader
	if requestBytes != nil {
		idaaslog.Unsafe.PrintfLn("Fetch with body: %s %s %v %s", method, endpoint, headers, string(requestBytes))
		requestBody = bytes.NewBuffer(requestBytes)
	} else {
		requestBody = nil
	}
	req, err := http.NewRequest(method, endpoint, requestBody)
	if err != nil {
		return nil, errors.Wrap(err, "new request: "+endpoint)
	}
	req.Header.Set("User-Agent", UserAgent)
	for k, v := range headers {
		req.Header.Set(k, v)
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, errors.Wrapf(err, "do %s request: %s", method, endpoint)
	}
	defer resp.Body.Close()
	logHttpResponse(resp)
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, errors.Wrapf(err, "read response body: %s", endpoint)
	}
	idaaslog.Unsafe.PrintfLn("%s %s, response: base64-encoded: %s", method, endpoint, base64.StdEncoding.EncodeToString(body))
	if resp.StatusCode != 200 {
		return nil, errors.Errorf("status code %d not 200: %s", resp.StatusCode, string(body))
	}
	return body, nil
}

func BuildHttpClient() *http.Client {
	client := &http.Client{
		Timeout: 10 * time.Second,
	}
	if UnsafeSkipCertificateVerification {
		idaaslog.Warn.PrintfLn("Env %s is turned on, TLS certificate verification will be off", constants.EnvUnsafeSkipCertificateVerification)
		transport := &http.Transport{TLSClientConfig: &tls.Config{
			InsecureSkipVerify: true,
		}}
		client.Transport = transport
	} else if RootCertificates != "" {
		certPool, err := loadRootCertificates(RootCertificates)
		if err != nil {
			idaaslog.Error.PrintfLn("Error loading additional root certificates: %s", err)
		}
		if certPool != nil {
			idaaslog.Info.PrintfLn("Additional root certificates are loaded from ca file: %s", RootCertificates)
			transport := &http.Transport{TLSClientConfig: &tls.Config{
				RootCAs: certPool,
			}}
			client.Transport = transport
		}
	}
	return client
}

func loadRootCertificates(caFile string) (*x509.CertPool, error) {
	if caFile == "" {
		return nil, nil
	}
	caCertPool, err := x509.SystemCertPool()
	if err != nil {
		return nil, errors.Errorf("failed to load system cert pool: %s", err.Error())
	}

	caCert, err := os.ReadFile(caFile)
	if err != nil {
		return nil, errors.Errorf("cannot read from ca file: %s, error: %s", caFile, err.Error())
	}

	if ok := caCertPool.AppendCertsFromPEM(caCert); !ok {
		return nil, errors.Errorf("cannot add to cert pool from ca file: %s", caFile)
	}
	return caCertPool, nil
}

func GetUserAgent() string {
	userAgent := os.Getenv(constants.EnvUserAgent)
	idaasV2VersionPart := fmt.Sprintf("AlibabaCloudIDaaS/2.0 AlibabaCloudIDaaSCli/%s", constants.Version)
	if userAgent != "" {
		return userAgent + " " + idaasV2VersionPart
	}
	deviceId := buildDeviceId()
	deviceUid := buildDeviceUserId(deviceId)
	return fmt.Sprintf("%s/%s %s%s%s%s",
		runtime.GOOS, runtime.GOARCH, idaasV2VersionPart, buildFeatures(), deviceId, deviceUid)
}

func buildFeatures() string {
	featureStrings := features.GetEnabledFeatures()
	if len(featureStrings) > 0 {
		return " WithFeatures/(" + strings.Join(featureStrings, ";") + ")"
	}
	return ""
}

func buildDeviceId() string {
	mac, err := GetMacAddress()
	if err == nil {
		return " DeviceID/did_" + Sha256ToBase32(mac+"/device_id_static_salt")
	}
	return ""
}

func buildDeviceUserId(deviceId string) string {
	currentUser, errUser := user.Current()
	if deviceId != "" && errUser == nil && currentUser != nil {
		return " DeviceUID/duid_" + Sha256ToBase32(currentUser.Username+"@"+deviceId)
	}
	return ""
}

func logHttpResponse(resp *http.Response) {
	if resp != nil {
		idaasRequestId := resp.Header.Get("x-idaas-request-id")
		if idaasRequestId != "" {
			idaaslog.Debug.PrintfLn("Request ID: %s", idaasRequestId)
			return
		}
		acsRequestId := resp.Header.Get("x-acs-request-id")
		if acsRequestId != "" {
			idaaslog.Debug.PrintfLn("Request ID: %s", acsRequestId)
		}
	}
}
