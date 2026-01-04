package alibaba_cloud

import (
	"fmt"
	"time"

	openapi "github.com/alibabacloud-go/darabonba-openapi/v2/client"
	sts20150401 "github.com/alibabacloud-go/sts-20150401/v2/client"
	util "github.com/alibabacloud-go/tea-utils/v2/service"
	"github.com/alibabacloud-go/tea/tea"
	"github.com/aliyunidaas/alibaba-cloud-idaas/cloud/cloud_common"
	"github.com/aliyunidaas/alibaba-cloud-idaas/config"
	"github.com/aliyunidaas/alibaba-cloud-idaas/constants"
	"github.com/aliyunidaas/alibaba-cloud-idaas/idaaslog"
	"github.com/aliyunidaas/alibaba-cloud-idaas/idp"
	"github.com/aliyunidaas/alibaba-cloud-idaas/utils"
	"github.com/pkg/errors"
)

type FetchStsWithOidcConfigOptions struct {
	ForceNew           bool
	ForceNewCloudToken bool
}

type FetchStsWithOidcOptions struct {
	Endpoint        string
	OidcProviderArn string
	RoleArn         string
	DurationSeconds int64
	RoleSessionName string
	FetchOidcToken  func() (string, error)
	ForceNew        bool
}

func FetchStsWithOidcConfig(profile string, alibabaCloudStsConfig *config.AlibabaCloudStsConfig,
	configOptions *FetchStsWithOidcConfigOptions) (
	*StsToken, error) {

	if alibabaCloudStsConfig.OidcTokenProvider == nil {
		return nil, errors.New("OidcTokenProvider is required")
	}
	stsEndpoint := alibabaCloudStsConfig.StsEndpoint
	if stsEndpoint == "" {
		if alibabaCloudStsConfig.Region == "" {
			return nil, errors.New("StsEndpoint or Region at least one is required")
		}
		stsEndpoint = fmt.Sprintf("sts.%s.aliyuncs.com", alibabaCloudStsConfig.Region)
		idaaslog.Debug.PrintfLn("Get sts endpoint: %s", stsEndpoint)
	}
	options := &FetchStsWithOidcOptions{
		Endpoint:        stsEndpoint,
		OidcProviderArn: alibabaCloudStsConfig.OidcProviderArn,
		RoleArn:         alibabaCloudStsConfig.RoleArn,
		RoleSessionName: alibabaCloudStsConfig.RoleSessionName,
		DurationSeconds: alibabaCloudStsConfig.DurationSeconds,
		FetchOidcToken: func() (string, error) {
			fetchOidcTokenOptions := &idp.FetchOidcTokenOptions{
				ForceNew: configOptions.ForceNew,
				CacheKey: alibabaCloudStsConfig.OidcTokenProvider.GetCacheKey(),
			}
			return idp.FetchOidcToken(profile, alibabaCloudStsConfig.OidcTokenProvider, fetchOidcTokenOptions)
		},
		ForceNew: configOptions.ForceNew || configOptions.ForceNewCloudToken,
	}
	return FetchStsWithOidc(profile, alibabaCloudStsConfig, options)
}

func FetchStsWithOidc(profile string, alibabaCloudStsConfig *config.AlibabaCloudStsConfig, options *FetchStsWithOidcOptions) (*StsToken, error) {
	digest := alibabaCloudStsConfig.Digest()
	readCacheFileOptions := &utils.ReadCacheOptions{
		Context: map[string]interface{}{
			"profile": profile,
			"digest":  digest,
			"config":  alibabaCloudStsConfig,
		},
		FetchContent: func() (int, string, error) {
			return fetchContent(options)
		},
		ForceNew: options.ForceNew,
		IsContentExpiringOrExpired: func(s *utils.StringWithTime) bool {
			return isContentExpiringOrExpired(s)
		},
		IsContentExpired: func(s *utils.StringWithTime) bool {
			return isContentExpired(s)
		},
	}

	cacheKey := fmt.Sprintf("%s_%s", profile, digest[0:32])
	idaaslog.Debug.PrintfLn("Cache key: %s %s", constants.CategoryCloudToken, cacheKey)
	stsTokenStr, err := utils.ReadCacheFileWithEncryptionCallback(
		constants.CategoryCloudToken, cacheKey, readCacheFileOptions)
	if err != nil {
		idaaslog.Error.PrintfLn("Error fetch cloud_token token with OIDC: %v", err)
		return nil, err
	}
	return UnmarshalStsToken(stsTokenStr)
}

func fetchContent(options *FetchStsWithOidcOptions) (int, string, error) {
	client, err := createStsClient(options.Endpoint)
	if err != nil {
		idaaslog.Error.PrintfLn("Error creating sts client: %v", err)
		return 600, "", err
	}
	oidcToken, err := options.FetchOidcToken()
	if err != nil {
		idaaslog.Error.PrintfLn("Error fetching oidc token: %v", err)
		return 600, "", err
	}
	stsResponse, err := assumeRoleWithOidc(client, oidcToken, options)
	if err != nil {
		idaaslog.Error.PrintfLn("Error assuming role: %v", err)
		return 600, "", err
	}
	if *stsResponse.StatusCode != 200 {
		idaaslog.Error.PrintfLn("failed assume role with OIDC, status: %v", stsResponse.StatusCode)
		return int(*stsResponse.StatusCode), "", errors.Errorf(
			"failed assume role with OIDC, status: %d", *stsResponse.StatusCode)
	}
	credentials := stsResponse.Body.Credentials
	stsToken := &StsToken{
		Mode:            "StsToken",
		AccessKeyId:     *credentials.AccessKeyId,
		AccessKeySecret: *credentials.AccessKeySecret,
		StsToken:        *credentials.SecurityToken,
		Expiration:      *credentials.Expiration,
	}
	stsTokenJson, err := stsToken.Marshal()
	if err != nil {
		idaaslog.Error.PrintfLn("Error marshaling sts token: %v", err)
		return 600, "", err
	}
	return 200, stsTokenJson, nil
}

func isContentExpiringOrExpired(s *utils.StringWithTime) bool {
	stsToken, err := UnmarshalStsToken(s.Content)
	if err != nil {
		return true
	}
	valid := stsToken.IsValidAtLeastThreshold(20 * time.Minute)
	idaaslog.Debug.PrintfLn("Check STS is expiring or expired: %s", !valid)
	return !valid
}

func isContentExpired(s *utils.StringWithTime) bool {
	stsToken, err := UnmarshalStsToken(s.Content)
	if err != nil {
		return true
	}
	valid := stsToken.IsValidAtLeastThreshold(3 * time.Minute)
	idaaslog.Debug.PrintfLn("Check STS is expired: %s", !valid)
	return !valid
}

func assumeRoleWithOidc(client *sts20150401.Client, oidcToken string, options *FetchStsWithOidcOptions) (
	*sts20150401.AssumeRoleWithOIDCResponse, error) {

	var roleSessionName string
	if options.RoleSessionName != "" {
		roleSessionName = options.RoleSessionName
	} else {
		roleSessionName = cloud_common.GenerateRoleSessionName(oidcToken)
		idaaslog.Info.PrintfLn(
			"Assume role session name not specified, use role session name %s", roleSessionName)
	}
	idaaslog.Debug.PrintfLn("Assume role, OIDCProviderArn: %s, RoleArn: %s, RoleSessionName: %s",
		options.OidcProviderArn, options.RoleArn, options.RoleSessionName)
	assumeRoleWithOidcRequest := &sts20150401.AssumeRoleWithOIDCRequest{
		OIDCProviderArn: tea.String(options.OidcProviderArn),
		RoleArn:         tea.String(options.RoleArn),
		OIDCToken:       tea.String(oidcToken),
		RoleSessionName: tea.String(roleSessionName),
	}
	if options.DurationSeconds > 0 {
		assumeRoleWithOidcRequest.DurationSeconds = tea.Int64(options.DurationSeconds)
	}
	runtime := &util.RuntimeOptions{}
	runtime.SetAutoretry(true)
	stsResponse, err := client.AssumeRoleWithOIDCWithOptions(assumeRoleWithOidcRequest, runtime)
	if err != nil {
		idaaslog.Error.PrintfLn("Error assume role with OIDC: %v", err)
	}
	return stsResponse, err
}

func createStsClient(endpoint string) (*sts20150401.Client, error) {
	openapiConfig := &openapi.Config{}
	// Endpoint referer: https://api.aliyun.com/product/Sts
	openapiConfig.Endpoint = tea.String(endpoint)
	client, err := sts20150401.NewClient(openapiConfig)
	if err != nil {
		idaaslog.Error.PrintfLn("Error create alibaba_cloud client: %v", err)
	}
	return client, err
}
