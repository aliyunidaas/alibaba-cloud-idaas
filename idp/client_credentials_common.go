package idp

import (
	"github.com/aliyunidaas/alibaba-cloud-idaas/config"
	"github.com/aliyunidaas/alibaba-cloud-idaas/oidc"
	"github.com/pkg/errors"
)

func buildFetchTokenCommonOptions(credentialConfig *config.OidcTokenProviderClientCredentialsConfig, options *oidc.OidcCommonOptions) *oidc.FetchTokenCommonOptions {
	tokenCommonsOptions := &oidc.FetchTokenCommonOptions{
		TokenEndpoint:                      credentialConfig.TokenEndpoint,
		ClientId:                           credentialConfig.ClientId,
		GrantType:                          oidc.GrantTypeClientCredentials,
		Scope:                              credentialConfig.Scope,
		ApplicationFederatedCredentialName: credentialConfig.ApplicationFederatedCredentialName,
	}
	updateTokenCommonOptions(tokenCommonsOptions, options)
	return tokenCommonsOptions
}

func updateTokenCommonOptions(tokenCommonsOptions *oidc.FetchTokenCommonOptions, options *oidc.OidcCommonOptions) {
	if options != nil {
		if options.GrantType != "" {
			tokenCommonsOptions.GrantType = options.GrantType
		}
		if options.Scope != "" {
			tokenCommonsOptions.Scope = options.Scope
		}
		tokenCommonsOptions.SubjectTokenType = options.SubjectTokenType
		tokenCommonsOptions.SubjectToken = options.SubjectToken
		tokenCommonsOptions.RequestedTokenType = options.RequestedTokenType
	}
}

func parseFetchAccessToken(tokenResponse *oidc.TokenResponse, errorResponse *oidc.ErrorResponse, err error) (*oidc.TokenResponse, error) {
	if err != nil {
		return nil, err
	}
	if errorResponse != nil {
		return nil, errors.Errorf("fetch token failed, error: %s, description: %s, requestId: %s",
			errorResponse.Error, errorResponse.ErrorDescription, errorResponse.RequestId)
	}
	return tokenResponse, nil
}
