package oidc

import (
	"fmt"
	"time"

	"github.com/aliyunidaas/alibaba-cloud-idaas/config"
	"github.com/aliyunidaas/alibaba-cloud-idaas/constants"
	"github.com/aliyunidaas/alibaba-cloud-idaas/idaaslog"
	"github.com/aliyunidaas/alibaba-cloud-idaas/idp"
	"github.com/aliyunidaas/alibaba-cloud-idaas/utils"
)

type FetchOidcTokenConfigOptions struct {
	ForceNew           bool
	ForceNewCloudToken bool
	FetchTokenType     FetchOidcTokenType
}

func FetchOidcToken(profile string, oidcTokenProviderConfig *config.OidcTokenProviderConfig, options *FetchOidcTokenConfigOptions) (
	*OidcToken, error) {
	digest := oidcTokenProviderConfig.Digest()
	readCacheFileOptions := &utils.ReadCacheOptions{
		Context: map[string]interface{}{
			"profile": profile,
			"digest":  digest,
			"config":  oidcTokenProviderConfig,
		},
		FetchContent: func() (int, string, error) {
			return fetchContent(oidcTokenProviderConfig, options)
		},
		IsContentExpiringOrExpired: func(s *utils.StringWithTime) bool {
			return isContentExpiringOrExpired(options.FetchTokenType, s)
		},
		IsContentExpired: func(s *utils.StringWithTime) bool {
			return isContentExpired(options.FetchTokenType, s)
		},
		ForceNew: options.ForceNew || options.ForceNewCloudToken,
	}

	cacheKey := fmt.Sprintf("%s_%s", profile, digest[0:32])
	idaaslog.Debug.PrintfLn("Cache key: %s %s", constants.CategoryCloudToken, cacheKey)
	oidcTokenStr, err := utils.ReadCacheFileWithEncryptionCallback(
		constants.CategoryCloudToken, cacheKey, readCacheFileOptions)
	if err != nil {
		idaaslog.Error.PrintfLn("Error fetch cloud_token token with OIDC: %v", err)
		return nil, err
	}
	return UnmarshalOidcToken(oidcTokenStr)
}

func fetchContent(oidcTokenProviderConfig *config.OidcTokenProviderConfig, options *FetchOidcTokenConfigOptions) (int, string, error) {
	startTime := time.Now().Unix()
	fetchOidcTokenOptions := &idp.FetchOidcTokenOptions{
		ForceNew: options.ForceNew,
		CacheKey: oidcTokenProviderConfig.GetCacheKey(),
	}
	tokenResponse, tokenResponseErr := idp.FetchTokenResponse(oidcTokenProviderConfig, fetchOidcTokenOptions)
	if tokenResponseErr == nil && tokenResponse != nil {
		oidcToken, oidcTokenErr := FromTokenResponse(startTime, tokenResponse)
		if oidcTokenErr != nil {
			return 600, "", oidcTokenErr
		}
		oidcTokenStr, oidcTokenStrErr := oidcToken.Marshal()
		if oidcTokenStrErr != nil {
			return 600, "", oidcTokenStrErr
		}
		return 200, oidcTokenStr, nil
	}
	return 600, "", tokenResponseErr
}

func isContentExpiringOrExpired(fetchTokenType FetchOidcTokenType, s *utils.StringWithTime) bool {
	oidcToken, err := UnmarshalOidcToken(s.Content)
	if err != nil {
		return true
	}
	valid := oidcToken.IsValidAtLeastThreshold(fetchTokenType, 3*time.Minute)
	idaaslog.Debug.PrintfLn("Check OIDC Token is expiring or expired: %s", !valid)
	return !valid
}

func isContentExpired(fetchTokenType FetchOidcTokenType, s *utils.StringWithTime) bool {
	oidcToken, err := UnmarshalOidcToken(s.Content)
	if err != nil {
		return true
	}
	valid := oidcToken.IsValidAtLeastThreshold(fetchTokenType, 1*time.Minute)
	idaaslog.Debug.PrintfLn("Check OIDC Token is expired: %s", !valid)
	return !valid
}
