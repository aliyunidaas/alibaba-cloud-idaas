package idp

import (
	"github.com/aliyunidaas/alibaba-cloud-idaas/config"
	"github.com/aliyunidaas/alibaba-cloud-idaas/idaaslog"
	"github.com/aliyunidaas/alibaba-cloud-idaas/oidc"
	"github.com/pkg/errors"
)

func FetchIdTokenDeviceCode(oidcTokenProviderDeviceCodeConfig *config.OidcTokenProviderDeviceCodeConfig,
	fetchOptions *FetchOidcTokenOptions) (*oidc.TokenResponse, error) {
	issuer := oidcTokenProviderDeviceCodeConfig.Issuer
	options := &oidc.FetchDeviceCodeFlowOptions{
		ClientId:     oidcTokenProviderDeviceCodeConfig.ClientId,
		ClientSecret: oidcTokenProviderDeviceCodeConfig.ClientSecret,
		Scope:        oidcTokenProviderDeviceCodeConfig.Scope,
		ShowQrCode:   oidcTokenProviderDeviceCodeConfig.ShowQrCode,
		SmallQrCode:  oidcTokenProviderDeviceCodeConfig.SmallQrCode,
		AutoOpenUrl:  oidcTokenProviderDeviceCodeConfig.AutoOpenUrl,
		ForceNew:     fetchOptions.ForceNew,
		CacheKey:     fetchOptions.CacheKey,
	}

	if !fetchOptions.ForceNew && fetchOptions.CacheKey != "" {
		tokenResponse := oidc.TryFetchTokenViaRefreshToken(issuer, fetchOptions.CacheKey, options)
		if tokenResponse != nil {
			idaaslog.Unsafe.PrintfLn("Try fetch token via refresh token response success %+v", tokenResponse)
			return tokenResponse, nil
		}
	}

	tokenResponse, err := oidc.FetchTokenViaDeviceCodeFlow(issuer, options)
	if err != nil {
		return nil, errors.Wrapf(err, "failed fetch id token via device code, issuer: %s", issuer)
	}
	return tokenResponse, nil
}
