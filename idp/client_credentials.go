package idp

import (
	"strings"

	"github.com/aliyunidaas/alibaba-cloud-idaas/config"
	"github.com/aliyunidaas/alibaba-cloud-idaas/oidc"
	"github.com/pkg/errors"
)

func FetchAccessTokenClientCredentials(credentialConfig *config.OidcTokenProviderClientCredentialsConfig, options *oidc.OidcCommonOptions) (*oidc.TokenResponse, error) {
	if credentialConfig == nil {
		return nil, errors.New("oidcTokenProviderClientCredentialsConfig is nil")
	}
	if credentialConfig.TokenEndpoint == "" {
		return nil, errors.New("oidcTokenProviderClientCredentialsConfig.TokenEndpoint is empty")
	}
	if credentialConfig.ClientId == "" {
		return nil, errors.New("oidcTokenProviderClientCredentialsConfig.ClientId is empty")
	}

	hasClientSecret := credentialConfig.ClientSecret != ""
	hasClientAssertionSigner := credentialConfig.GetClientAssertionSigner() != nil
	hasClientAssertionPkcs7 := credentialConfig.ClientAssertionPkcs7Config != nil
	hasClientAssertionPrivateCa := credentialConfig.ClientAssertionPrivateCaConfig != nil
	hasClientAssertionOidcToken := credentialConfig.ClientAssertionOidcTokenConfig != nil

	var clientAuthMethods []string
	if hasClientSecret {
		clientAuthMethods = append(clientAuthMethods, "secret")
	}
	if hasClientAssertionSigner {
		clientAuthMethods = append(clientAuthMethods, "signer")
	}
	if hasClientAssertionPkcs7 {
		clientAuthMethods = append(clientAuthMethods, "pkcs7")
	}
	if hasClientAssertionPrivateCa {
		clientAuthMethods = append(clientAuthMethods, "private_ca")
	}
	if hasClientAssertionOidcToken {
		clientAuthMethods = append(clientAuthMethods, "oidc_token")
	}

	if len(clientAuthMethods) > 1 {
		return nil, errors.Errorf("multiple client auth methods found: %s", strings.Join(clientAuthMethods, ", "))
	}

	if hasClientSecret {
		return FetchAccessTokenClientCredentialsClientIdSecret(credentialConfig, options)
	} else if hasClientAssertionSigner {
		return FetchAccessTokenClientCredentialsRfc7523(credentialConfig, options)
	} else if hasClientAssertionPkcs7 {
		return FetchAccessTokenClientCredentialsPkcs7(credentialConfig, options)
	} else if hasClientAssertionPrivateCa {
		return FetchAccessTokenClientCredentialsPrivateCa(credentialConfig, options)
	} else if hasClientAssertionOidcToken {
		return FetchAccessTokenClientCredentialsOidcToken(credentialConfig, options)
	} else {
		return nil, errors.New("client auth method must set one")
	}
}
