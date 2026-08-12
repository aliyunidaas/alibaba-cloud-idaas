package idp

import (
	"github.com/aliyunidaas/alibaba-cloud-idaas/config"
	"github.com/aliyunidaas/alibaba-cloud-idaas/oidc"
)

func FetchAccessTokenClientCredentialsClientIdSecret(credentialConfig *config.OidcTokenProviderClientCredentialsConfig, options *oidc.OidcCommonOptions) (*oidc.TokenResponse, error) {
	tokenEndpoint := credentialConfig.TokenEndpoint
	fetchTokenOptions := &oidc.FetchTokenOptions{
		ClientId:     credentialConfig.ClientId,
		ClientSecret: credentialConfig.ClientSecret,
		GrantType:    oidc.GrantTypeClientCredentials,
		Scope:        credentialConfig.Scope,
	}
	if options != nil {
		if options.GrantType != "" {
			fetchTokenOptions.GrantType = options.GrantType
		}
		fetchTokenOptions.SubjectTokenType = options.SubjectTokenType
		fetchTokenOptions.SubjectToken = options.SubjectToken
		fetchTokenOptions.RequestedTokenType = options.RequestedTokenType
	}

	tokenResponse, errorResponse, err := oidc.FetchToken(tokenEndpoint, fetchTokenOptions)
	return parseFetchAccessToken(tokenResponse, errorResponse, err)
}
