package idp

import (
	"fmt"
	"net/url"
	"os"

	"github.com/aliyunidaas/alibaba-cloud-idaas/config"
	"github.com/aliyunidaas/alibaba-cloud-idaas/constants"
	"github.com/aliyunidaas/alibaba-cloud-idaas/oidc"
	"github.com/aliyunidaas/alibaba-cloud-idaas/utils"
	"github.com/pkg/errors"
)

const (
	OidcTokenProviderGcp    = "gcp"
	OidcTokenProviderCustom = "custom"
)

func FetchAccessTokenClientCredentialsOidcToken(credentialConfig *config.OidcTokenProviderClientCredentialsConfig, options *oidc.OidcCommonOptions) (*oidc.TokenResponse, error) {
	tokenEndpoint := credentialConfig.TokenEndpoint
	idToken, err := fetchOidcToken(credentialConfig)
	if err != nil {
		return nil, err
	}
	fetchTokenIdTokenBearerOptions := &oidc.FetchTokenIdTokenBearerOptions{
		FetchTokenCommonOptions: buildFetchTokenCommonOptions(credentialConfig, options),
		IdToken:                 idToken,
	}
	tokenResponse, errorResponse, err := oidc.FetchTokenIdTokenBearer(tokenEndpoint, fetchTokenIdTokenBearerOptions)
	return parseFetchAccessToken(tokenResponse, errorResponse, err)
}

func fetchOidcToken(credentialConfig *config.OidcTokenProviderClientCredentialsConfig) (string, error) {
	oidcTokenConfig := credentialConfig.ClientAssertionOidcTokenConfig
	provider := oidcTokenConfig.Provider
	if provider == OidcTokenProviderGcp {
		return fetchOidcTokenForGcp(oidcTokenConfig.GoogleVmIdentityUrl, oidcTokenConfig.GoogleVmIdentityAud)
	} else if provider == OidcTokenProviderCustom {
		if oidcTokenConfig.OidcToken != "" && oidcTokenConfig.OidcTokenFile != "" {
			return "", errors.Errorf("OidcToken and OidcTokenFile cannot both be set")
		}
		if oidcTokenConfig.OidcToken != "" {
			return oidcTokenConfig.OidcToken, nil
		} else if oidcTokenConfig.OidcTokenFile != "" {
			oidcTokenBytes, err := os.ReadFile(oidcTokenConfig.OidcTokenFile)
			if err != nil {
				return "", errors.Wrapf(err, "failed to read file %s", oidcTokenConfig.OidcTokenFile)
			}
			return string(oidcTokenBytes), nil
		} else {
			return "", errors.New("one of OidcToken or OidcTokenFile must be specified")
		}
	} else {
		return "", errors.New("unknown provider " + provider)
	}
}

// reference: https://cloud.google.com/compute/docs/instances/verifying-instance-identity
func fetchOidcTokenForGcp(endpoint, aud string) (string, error) {
	client := utils.BuildHttpClient()
	if aud == "" {
		aud = constants.DefaultAudienceAlibabaCloudIdaas
	}
	if endpoint == "" {
		endpoint = fmt.Sprintf("%s?audience=%s&format=%s&licenses=%s",
			"http://metadata/computeMetadata/v1/instance/service-accounts/default/identity",
			url.QueryEscape(aud), "full", "TRUE")
	}
	headers := map[string]string{
		"Metadata-Flavor": "Google",
	}
	return utils.FetchAsString(client, utils.HttpMethodGet, endpoint, headers)
}
