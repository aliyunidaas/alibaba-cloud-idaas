package cloud_account

import (
	"fmt"
	"strings"
	"time"

	"github.com/aliyunidaas/alibaba-cloud-idaas/config"
	"github.com/aliyunidaas/alibaba-cloud-idaas/constants"
	"github.com/aliyunidaas/alibaba-cloud-idaas/idaaslog"
	"github.com/aliyunidaas/alibaba-cloud-idaas/idp"
	"github.com/aliyunidaas/alibaba-cloud-idaas/oidc"
	"github.com/aliyunidaas/alibaba-cloud-idaas/utils"
	"github.com/pkg/errors"
)

type FetchCloudAccountTokenWithOidcConfigOptions struct {
	ForceNew           bool
	ForceNewCloudToken bool
}

type FetchCloudAccountTokenWithOidcOptions struct {
	Endpoint         string
	RoleExternalId   string
	FetchAccessToken func() (string, error)
	ForceNew         bool
}

func FetchCloudAccountTokenWithOidcConfig(profile string, cloudAccountTokenConfig *config.CloudAccountTokenConfig,
	configOptions *FetchCloudAccountTokenWithOidcConfigOptions) (
	*CloudAccountToken, error) {
	if cloudAccountTokenConfig.AccessTokenProvider == nil {
		return nil, errors.New("AccessTokenProvider is required")
	}
	cloudAccountEndpoint, cloudAccountEndpointErr := cloudAccountTokenConfig.GetCloudAccountEndpoint()
	if cloudAccountEndpointErr != nil {
		return nil, cloudAccountEndpointErr
	}
	if cloudAccountEndpoint == "" {
		return nil, errors.New("CloudAccountEndpoint is required")
	}
	options := &FetchCloudAccountTokenWithOidcOptions{
		Endpoint:       cloudAccountEndpoint,
		RoleExternalId: cloudAccountTokenConfig.CloudAccountRoleExternalId,
		FetchAccessToken: func() (string, error) {
			fetchOidcTokenOptions := &idp.FetchOidcTokenOptions{
				ForceNew: configOptions.ForceNew,
				CacheKey: cloudAccountTokenConfig.AccessTokenProvider.GetCacheKey(),
			}
			// MUST be Access Token for Cloud Account Token obtain
			cloudAccountTokenConfig.AccessTokenProvider.TokenType = oidc.TokenAccessToken
			return idp.FetchOidcToken(profile, cloudAccountTokenConfig.AccessTokenProvider, fetchOidcTokenOptions)
		},
		ForceNew: configOptions.ForceNew || configOptions.ForceNewCloudToken,
	}
	return FetchCloudAccountTokenWithOidc(profile, cloudAccountTokenConfig, options)
}

func FetchCloudAccountTokenWithOidc(profile string, cloudAccountTokenConfig *config.CloudAccountTokenConfig, options *FetchCloudAccountTokenWithOidcOptions) (*CloudAccountToken, error) {
	digest := cloudAccountTokenConfig.Digest()
	readCacheFileOptions := &utils.ReadCacheOptions{
		Context: map[string]interface{}{
			"profile": profile,
			"digest":  digest,
			"config":  cloudAccountTokenConfig,
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
	cloudAccountTokenStr, err := utils.ReadCacheFileWithEncryptionCallback(
		constants.CategoryCloudToken, cacheKey, readCacheFileOptions)
	if err != nil {
		idaaslog.Error.PrintfLn("Error fetch cloud_token token with OIDC: %v", err)
		return nil, err
	}
	return UnmarshalCloudAccountToken(cloudAccountTokenStr)
}

func fetchContent(options *FetchCloudAccountTokenWithOidcOptions) (int, string, error) {
	accessToken, err := options.FetchAccessToken()
	if err != nil {
		idaaslog.Error.PrintfLn("Error fetching access token: %v", err)
		return 600, "", err
	}
	cloudAccountTokenJson, err := fetchCloudAccountToken(options.Endpoint, options.RoleExternalId, accessToken)
	if err != nil {
		idaaslog.Error.PrintfLn("Error fetching Cloud Account token: %v", err)
		return 600, "", err
	}
	return 200, cloudAccountTokenJson, nil
}

func isContentExpiringOrExpired(s *utils.StringWithTime) bool {
	cloudAccountToken, err := UnmarshalCloudAccountToken(s.Content)
	if err != nil {
		return true
	}
	valid := cloudAccountToken.IsValidAtLeastThreshold(20 * time.Minute)
	idaaslog.Debug.PrintfLn("Check Cloud Account token is expiring or expired: %s", !valid)
	return !valid
}

func isContentExpired(s *utils.StringWithTime) bool {
	cloudAccountToken, err := UnmarshalCloudAccountToken(s.Content)
	if err != nil {
		return true
	}
	valid := cloudAccountToken.IsValidAtLeastThreshold(3 * time.Minute)
	idaaslog.Debug.PrintfLn("Check Cloud Account token is expired: %s", !valid)
	return !valid
}

func fetchCloudAccountToken(cloudAccountEndpoint, cloudAccountRoleExternalId, accessToken string) (string, error) {
	client := utils.BuildHttpClient()
	endpoint := utils.NewUrlBuilder(cloudAccountEndpoint)
	endpoint.AddQuery("cloudAccountRoleExternalId", cloudAccountRoleExternalId)
	headers := map[string]string{
		"Authorization": "Bearer " + accessToken,
	}
	cloudAccountTokenJson, err := utils.FetchAsString(client, utils.HttpMethodGet, endpoint.BuildUrl(), headers)
	if err != nil {
		if hint := obtainErrorHint(err.Error()); hint != "" {
			return "", errors.Wrapf(err,
				"Fetch cloud account token failed, endpoint: %s, external ID: %s\n  hint: %s", cloudAccountEndpoint, cloudAccountRoleExternalId, hint)
		}
		return "", errors.Wrapf(err,
			"Fetch cloud account token failed, endpoint: %s, external ID: %s", cloudAccountEndpoint, cloudAccountRoleExternalId)
	}
	idaaslog.Unsafe.PrintfLn("Fetch cloud account token: %s", cloudAccountTokenJson)
	return cloudAccountTokenJson, nil
}

// obtainErrorHint maps known obtainAccessCredential error bodies to actionable guidance.
func obtainErrorHint(errStr string) string {
	switch {
	case strings.Contains(errStr, "invalid_audience"), strings.Contains(errStr, "invalid_client_credential"):
		return "换发时服务端 PAM→云 STS 内部联邦的 client_assertion(private_key_jwt) audience 无效——通常是该云账号在当前环境的 OIDC 联邦/授权服务器 audience 配置问题（非客户端可修复），请联系后端/管理员核对云账号 onboarding 与联邦 audience。"
	case strings.Contains(errStr, "operation_denied_by_license"):
		return "该操作被实例 License 限制，请确认实例 License 版本与应用授权，或联系管理员开通对应能力。"
	case strings.Contains(errStr, "has not been authorized by resource server"):
		return "broker 客户端未被委派到 PAM 资源服务器，请配置 M2M 委派授权（scope: cloud_account_role:obtain_access_credential）。"
	case strings.Contains(errStr, "not_found"):
		return "云角色不存在或未授权给当前用户，请确认角色已 onboard 且 PS 授权规则已授予该用户。"
	}
	return ""
}
