package cloud_account

import (
	"fmt"
	"net/url"
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
	cloudAccountEndpoint := cloudAccountTokenConfig.CloudAccountEndpoint
	if cloudAccountEndpoint == "" {
		return nil, errors.New("CloudAccountEndpoint is required")
	}
	options := &FetchCloudAccountTokenWithOidcOptions{
		Endpoint:       cloudAccountTokenConfig.CloudAccountEndpoint,
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
	endpoint := cloudAccountEndpoint
	if strings.Contains(cloudAccountEndpoint, "?") {
		endpoint += "&"
	} else {
		endpoint += "?"
	}
	endpoint += fmt.Sprintf("cloudAccountRoleExternalId=%s", url.QueryEscape(cloudAccountRoleExternalId))
	headers := map[string]string{
		"Authorization": "Bearer " + accessToken,
	}
	cloudAccountTokenJson, err := utils.FetchAsString(client, utils.HttpMethodGet, endpoint, headers)
	if err != nil {
		return "", errors.Wrapf(err,
			"Fetch cloud account token failed, endpoint: %s, external ID: %s", cloudAccountEndpoint, cloudAccountRoleExternalId)
	}
	idaaslog.Unsafe.PrintfLn("Fetch cloud account token: %s", cloudAccountTokenJson)
	return cloudAccountTokenJson, nil
}
