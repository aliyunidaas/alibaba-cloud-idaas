package utils

import (
	"encoding/base64"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"runtime"
	"strings"
	"time"

	"github.com/aliyunidaas/alibaba-cloud-idaas/constants"
	"github.com/aliyunidaas/alibaba-cloud-idaas/idaaslog"
	"github.com/pkg/errors"
)

const (
	HttpMethodPut  = "PUT"
	HttpMethodGet  = "GET"
	HttpMethodPost = "POST"
)

var UserAgent = getUserAgent()

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

func Fetch(client *http.Client, method, endpoint string, headers map[string]string) ([]byte, error) {
	req, err := http.NewRequest(method, endpoint, nil)
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
	return &http.Client{
		Timeout: 10 * time.Second,
	}
}

func getUserAgent() string {
	userAgent := os.Getenv(constants.EnvUserAgent)
	idaasV2VersionPart := fmt.Sprintf("AlibabaCloudIDaaS/2.0 AlibabaCloudIDaaSCli/%s", constants.AlibabaCloudIdaasCliVersion)
	if userAgent != "" {
		return userAgent + " " + idaasV2VersionPart
	}
	return fmt.Sprintf("%s/%s %s",
		runtime.GOOS, runtime.GOARCH, idaasV2VersionPart)
}
