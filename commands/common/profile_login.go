package common

import (
	"fmt"
	"path/filepath"

	"github.com/aliyunidaas/alibaba-cloud-idaas/config"
)

// ProfileLoginContext is the effective input to DoLogin, without reading tokens.
type ProfileLoginContext struct {
	Issuer         string
	ClientID       string
	ClientIDSource string
	Scope          string
}

func ResolveConfigFilename(filename string) (string, error) {
	if filename == "" {
		var err error
		filename, err = config.GetDefaultCloudCredentialConfigFile()
		if err != nil {
			return "", err
		}
	}
	return filepath.Abs(filename)
}

// ResolveProfileLoginContext preserves login --profile's selection and defaults.
// It deliberately does not resolve another profile or alter any cache key.
func ResolveProfileLoginContext(sts *config.CloudStsConfig, profile, scope, client string) (*ProfileLoginContext, error) {
	var provider *config.OidcTokenProviderConfig
	if sts != nil {
		if sts.CloudAccount != nil && sts.CloudAccount.AccessTokenProvider != nil {
			provider = sts.CloudAccount.AccessTokenProvider
		} else if sts.OidcToken != nil {
			provider = sts.OidcToken
		}
	}
	if provider == nil {
		return nil, fmt.Errorf("profile has no OIDC token provider (device_code)")
	}
	dc := provider.OidcTokenProviderDeviceCode
	if dc == nil {
		return nil, fmt.Errorf("profile has no device_code provider")
	}
	result := &ProfileLoginContext{Issuer: dc.Issuer, ClientID: client, Scope: scope, ClientIDSource: "explicit"}
	if result.ClientID == "" {
		result.ClientID = dc.ClientId
		result.ClientIDSource = "profile:" + profile
	}
	if result.ClientID == "" {
		result.ClientID = DefaultClientId
		result.ClientIDSource = "built-in default"
	}
	if result.Scope == "" {
		result.Scope = dc.Scope
	}
	if result.Scope == "" {
		result.Scope = DefaultPamScope
	}
	return result, nil
}
