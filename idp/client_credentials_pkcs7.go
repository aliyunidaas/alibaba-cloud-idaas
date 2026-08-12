package idp

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"time"

	"github.com/aliyunidaas/alibaba-cloud-idaas/config"
	"github.com/aliyunidaas/alibaba-cloud-idaas/idaaslog"
	"github.com/aliyunidaas/alibaba-cloud-idaas/oidc"
	"github.com/aliyunidaas/alibaba-cloud-idaas/utils"
	"github.com/pkg/errors"
)

const (
	Pkcs7ProviderAlibabaCloud = "alibaba_cloud"
	Pkcs7ProviderAws          = "aws"
	Pkcs7ProviderAzure        = "azure"
)

func FetchAccessTokenClientCredentialsPkcs7(credentialConfig *config.OidcTokenProviderClientCredentialsConfig, options *oidc.OidcCommonOptions) (*oidc.TokenResponse, error) {
	tokenEndpoint := credentialConfig.TokenEndpoint
	pkcs7, err := fetchPkcs7(credentialConfig)
	if err != nil {
		return nil, err
	}
	fetchTokenPkcs7BearerOptions := &oidc.FetchTokenPkcs7BearerOptions{
		FetchTokenCommonOptions: buildFetchTokenCommonOptions(credentialConfig, options),
		Pkcs7:                   base64.StdEncoding.EncodeToString(pkcs7),
	}
	tokenResponse, errorResponse, err := oidc.FetchTokenPkcs7Bearer(tokenEndpoint, fetchTokenPkcs7BearerOptions)
	return parseFetchAccessToken(tokenResponse, errorResponse, err)
}

func fetchPkcs7(credentialConfig *config.OidcTokenProviderClientCredentialsConfig) ([]byte, error) {
	pkcs7Config := credentialConfig.ClientAssertionPkcs7Config
	provider := pkcs7Config.Provider

	var pkcs7 []byte
	var pkcs7Err error
	if provider == Pkcs7ProviderAlibabaCloud {
		pkcs7, pkcs7Err = fetchPkcs7ForAlibabaCloud(pkcs7Config.AlibabaCloudIdaasInstanceId, pkcs7Config.AlibabaCloudMode)
	} else if provider == Pkcs7ProviderAws {
		pkcs7, pkcs7Err = fetchPkcs7ForAwsImdsv2Rsa2048()
	} else if provider == Pkcs7ProviderAzure {
		pkcs7, pkcs7Err = fetchPkcs7ForAzure()
	} else {
		return nil, errors.New("unknown provider " + provider)
	}

	if pkcs7Err != nil {
		return nil, errors.Wrapf(pkcs7Err, "fetch pkcs7 from %s failed", provider)
	}

	idaaslog.Unsafe.PrintfLn("Fetch %s PKCS#7 %s", provider, base64.StdEncoding.EncodeToString(pkcs7))
	return pkcs7, nil
}

// reference: https://www.alibabacloud.com/help/en/ecs/user-guide/use-instance-identities
func fetchPkcs7ForAlibabaCloud(instanceId, mode string) ([]byte, error) {
	isHardenMode := getAlibabaCloudHardenMode(mode)
	client := utils.BuildHttpClient()
	token := ""
	if isHardenMode {
		var err error
		token, err = fetchAlibabaCloudSecureToken(client)
		if err != nil {
			return nil, err
		}
	}

	audience, err := buildAlibabaCloudPkcs7Audience(instanceId)
	if err != nil {
		return nil, err
	}
	return fetchAlibabaCloudPkcs7(client, audience, token)
}

// reference: https://docs.aws.amazon.com/AWSEC2/latest/UserGuide/verify-iid.html
func fetchPkcs7ForAwsImdsv2Rsa2048() ([]byte, error) {
	client := utils.BuildHttpClient()
	token, err := fetchAwsSecureToken(client)
	if err != nil {
		return nil, err
	}
	return fetchAwsPkcs7Imdsv2Rsa2048(client, token)
}

type AzurePkcs7Response struct {
	Encoding  string `json:"encoding"`
	Signature string `json:"signature"`
}

// reference: https://learn.microsoft.com/en-us/azure/virtual-machines/instance-metadata-service?tabs=windows
func fetchPkcs7ForAzure() ([]byte, error) {
	client := utils.BuildHttpClient()
	azurePkcs7Endpoint := "http://169.254.169.254/metadata/attested/document?api-version=2020-09-01"
	headers := map[string]string{
		"Metadata": "true",
	}
	body, err := utils.Fetch(client, utils.HttpMethodGet, azurePkcs7Endpoint, headers)
	if err != nil {
		return nil, err
	}
	var azurePkcs7Response AzurePkcs7Response
	err = json.Unmarshal(body, &azurePkcs7Response)
	if err != nil {
		return nil, errors.Wrapf(err, "unmarshal response body: %s", string(body))
	}
	if azurePkcs7Response.Encoding != "pkcs7" {
		return nil, errors.Errorf("invalid pkcs7 response: %s", string(body))
	}
	pkcs7, err := base64.StdEncoding.DecodeString(azurePkcs7Response.Signature)
	if err != nil {
		return nil, errors.Wrapf(err, "invalid pkcs7: %s", string(body))
	}
	return pkcs7, nil
}

func getAlibabaCloudHardenMode(mode string) bool {
	if mode == "normal" {
		return false
	}
	if mode == "" || mode == "secure" {
		return true
	}
	idaaslog.Info.PrintfLn("Alibaba Cloud harden mode is not configured, default use harden mode")
	return true
}

func buildAlibabaCloudPkcs7Audience(instanceId string) (string, error) {
	audience := map[string]any{}
	audience["aud"] = instanceId
	audience["signingTime"] = time.Now().Unix()

	audienceBytes, err := json.Marshal(audience)
	if err != nil {
		// SHOULD NOT HAPPEN
		return "", errors.Wrap(err, "marshal audience bytes")
	}
	return string(audienceBytes), nil
}

func fetchAlibabaCloudPkcs7(client *http.Client, audience, token string) ([]byte, error) {
	alibabaCloudPkcs7Endpoint := fmt.Sprintf(
		"http://100.100.100.200/latest/dynamic/instance-identity/pkcs7?audience=%s", url.QueryEscape(audience))
	headers := map[string]string{}
	if token != "" {
		headers["X-aliyun-ecs-metadata-token"] = token
	}
	bytes, err := utils.Fetch(client, utils.HttpMethodGet, alibabaCloudPkcs7Endpoint, headers)
	if err != nil {
		return nil, err
	}
	pkcs7, err := base64.StdEncoding.DecodeString(trimAllSpaces(string(bytes)))
	if err != nil {
		idaaslog.Warn.PrintfLn("base64 decode pkcs7 response failed: %s, error: %s", string(bytes), err.Error())
		return bytes, nil
	}
	return pkcs7, nil
}

func fetchAlibabaCloudSecureToken(client *http.Client) (string, error) {
	alibabaCloudTokenEndpoint := "http://100.100.100.200/latest/api/token"
	headers := map[string]string{
		"X-aliyun-ecs-metadata-token-ttl-seconds": "3600",
	}
	return utils.FetchAsString(client, utils.HttpMethodPut, alibabaCloudTokenEndpoint, headers)
}

func fetchAwsPkcs7Imdsv2Rsa2048(client *http.Client, token string) ([]byte, error) {
	alibabaCloudPkcs7Endpoint := "http://169.254.169.254/latest/dynamic/instance-identity/rsa2048"
	headers := map[string]string{
		"X-aws-ec2-metadata-token": token,
	}
	bytes, err := utils.Fetch(client, utils.HttpMethodGet, alibabaCloudPkcs7Endpoint, headers)
	if err != nil {
		return nil, err
	}
	pkcs7, err := base64.StdEncoding.DecodeString(trimAllSpaces(string(bytes)))
	if err != nil {
		idaaslog.Warn.PrintfLn("base64 decode pkcs7 response failed: %s, error: %s", string(bytes), err.Error())
		return bytes, nil
	}
	return pkcs7, nil
}

func fetchAwsSecureToken(client *http.Client) (string, error) {
	awsTokenEndpoint := "http://169.254.169.254/latest/api/token"
	headers := map[string]string{
		"X-aws-ec2-metadata-token-ttl-seconds": "3600",
	}
	return utils.FetchAsString(client, utils.HttpMethodPut, awsTokenEndpoint, headers)
}

func trimAllSpaces(str string) string {
	filterString := ""
	for _, char := range str {
		if char == ' ' || char == '\t' || char == '\r' || char == '\n' {
			// SKIP
		} else {
			filterString += string(char)
		}
	}
	return filterString
}
