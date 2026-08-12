package util

import (
	"fmt"
	"strings"

	"github.com/aliyunidaas/alibaba-cloud-idaas/config"
	"github.com/aliyunidaas/alibaba-cloud-idaas/oidc"

	"github.com/pkg/errors"
)

func GetClonedAgentConfig(configFilename, profile, scope string) (*config.AgentConfig, error) {
	profile, cloudStsConfig, err := config.FindProfile(configFilename, profile, false)
	if err != nil {
		return nil, fmt.Errorf("find profile `%s` error: %s", profile, err)
	}

	if cloudStsConfig.Agent == nil {
		return nil, fmt.Errorf("not agent profile `%s`", profile)
	}
	clonedAgentConfig, err := cloudStsConfig.Agent.Clone()
	if err != nil {
		return nil, err
	}

	err = updateWithScope(clonedAgentConfig.AccessTokenProvider, scope)
	if err != nil {
		return nil, err
	}

	// MUST be Access Token for Cloud Account Token obtain
	clonedAgentConfig.AccessTokenProvider.TokenType = oidc.TokenAccessToken

	return clonedAgentConfig, nil
}

func updateWithScope(oidcTokenProviderConfig *config.OidcTokenProviderConfig, scope string) error {
	if oidcTokenProviderConfig == nil {
		return nil
	}
	if scope == "" {
		// scope is empty, keep original config
		return nil
	}

	if oidcTokenProviderConfig.OidcTokenProviderDeviceCode != nil {
		oidcTokenProviderConfig.OidcTokenProviderDeviceCode.Scope = scope
	}
	if oidcTokenProviderConfig.OidcTokenProviderClientCredentials != nil {
		oidcTokenProviderConfig.OidcTokenProviderClientCredentials.Scope = scope
	}
	if oidcTokenProviderConfig.OpenApi != nil {
		audience, scopeValues, err := splitScope(scope)
		if err != nil {
			return errors.Wrapf(err, "parse scope %s failed", scope)
		}
		oidcTokenProviderConfig.OpenApi.Audience = audience
		oidcTokenProviderConfig.OpenApi.ScopeValues = scopeValues
	}

	return nil
}

func splitScope(scope string) (string, []string, error) {
	var audience string
	var scopeValues []string
	scopes := strings.Split(scope, " ")
	for _, singleScope := range scopes {
		singleScopeParts := strings.Split(singleScope, "|")
		if len(singleScopeParts) != 2 {
			return "", nil, errors.New("invalid scope format, must be audience|permission")
		}
		if audience == "" {
			audience = singleScopeParts[0]
		} else if audience != singleScopeParts[0] {
			return "", nil, errors.Errorf("multiple audience values are not allowed %s & %s", audience, singleScopeParts[0])
		}
		scopeValues = append(scopeValues, singleScopeParts[1])
	}
	return audience, scopeValues, nil
}
