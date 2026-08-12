package common

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/aliyunidaas/alibaba-cloud-idaas/config"
	"github.com/aliyunidaas/alibaba-cloud-idaas/idp"
	"github.com/aliyunidaas/alibaba-cloud-idaas/oidc"
	"github.com/aliyunidaas/alibaba-cloud-idaas/utils"
	"github.com/pkg/errors"
)

const (
	// DefaultPamScope is the default scope for login — PAM all capabilities.
	DefaultPamScope = "urn:cloud:idaas:pam|.all"
	// DefaultClientId is the fallback client ID when discovery doesn't return cli_client_id.
	DefaultClientId = "iap_developer"
	// LoginProfileTag is the profile tag used for login token caching.
	LoginProfileTag = "cloud-idaas-cli"
	// DiscoveryPath is the well-known path for instance discovery.
	DiscoveryPath = "/.well-known/cloud-idaas-configuration"
)

// InstanceDiscovery corresponds to the /.well-known/cloud-idaas-configuration response.
type InstanceDiscovery struct {
	InstanceId                       string `json:"instance_id"`
	DefaultAuthorizationServerIssuer string `json:"default_authorization_server_issuer"`
	DeveloperApiEndpoint             struct {
		Internet string `json:"internet"`
		Vpc      string `json:"vpc"`
	} `json:"developer_api_endpoint"`
	CliClientId string `json:"cli_client_id"`
}

// FetchInstanceDiscovery calls /.well-known/cloud-idaas-configuration and returns the discovery data.
func FetchInstanceDiscovery(httpClient *http.Client, instance string) (*InstanceDiscovery, error) {
	discoveryUrl := "https://" + instance + DiscoveryPath
	body, err := utils.FetchAsString(httpClient, utils.HttpMethodGet, discoveryUrl, nil)
	if err != nil {
		return nil, err
	}
	var discovery InstanceDiscovery
	if err := json.Unmarshal([]byte(body), &discovery); err != nil {
		return nil, errors.Wrapf(err, "unmarshal discovery response: %s", body)
	}
	return &discovery, nil
}

// DoLogin performs a device-code login and caches the access token.
// Used by both the `login` command and `onboard` (when no valid token exists).
func DoLogin(issuer, clientId, scope string, forceNew bool) (string, error) {
	if scope == "" {
		scope = DefaultPamScope
	}
	if clientId == "" {
		clientId = DefaultClientId
	}
	provider := &config.OidcTokenProviderConfig{
		TokenType: oidc.TokenAccessToken,
		OidcTokenProviderDeviceCode: &config.OidcTokenProviderDeviceCodeConfig{
			Issuer:      issuer,
			ClientId:    clientId,
			Scope:       scope,
			AutoOpenUrl: true,
			ShowQrCode:  true,
			SmallQrCode: true,
		},
	}
	return idp.FetchOidcToken(LoginProfileTag, provider, &idp.FetchOidcTokenOptions{
		ForceNew: forceNew,
		CacheKey: provider.GetCacheKey(),
	})
}

// ResolveClientId resolves the client ID from discovery or falls back to default.
func ResolveClientId(discovery *InstanceDiscovery) string {
	if discovery != nil && discovery.CliClientId != "" {
		return discovery.CliClientId
	}
	return DefaultClientId
}

// ResolvePopEndpoint resolves the developer API endpoint from discovery, preferring VPC if requested.
func ResolvePopEndpoint(discovery *InstanceDiscovery, preferVpc bool) string {
	pop := discovery.DeveloperApiEndpoint.Internet
	if preferVpc && discovery.DeveloperApiEndpoint.Vpc != "" {
		pop = discovery.DeveloperApiEndpoint.Vpc
	}
	return utils.NormalizeDeveloperApiEndpoint(pop)
}

// ValidateDiscovery checks that the discovery response has all required fields.
func ValidateDiscovery(discovery *InstanceDiscovery) error {
	if discovery.InstanceId == "" || discovery.DefaultAuthorizationServerIssuer == "" {
		return fmt.Errorf("incomplete discovery response: %+v", discovery)
	}
	return nil
}

// InferredProfile holds instance info extracted from an existing profile.
type InferredProfile struct {
	Instance string // instance domain, e.g. "acme.aliyunidaas.com"
	ClientId string // broker client id, e.g. "app_xxx"
	Issuer   string // full issuer URL
}

// InferFromProfiles reads existing profiles and extracts instance info
// from the first cloud_account_token profile's AccessTokenProvider.DeviceCode config.
// Returns nil if no suitable profile is found.
func InferFromProfiles(configFilename string) *InferredProfile {
	cloudConfig, err := config.ReadCloudCredentialConfig(configFilename)
	if err != nil || cloudConfig == nil {
		return nil
	}
	for _, sts := range cloudConfig.Profile {
		if sts.CloudAccount != nil && sts.CloudAccount.AccessTokenProvider != nil {
			dc := sts.CloudAccount.AccessTokenProvider.OidcTokenProviderDeviceCode
			if dc != nil && dc.Issuer != "" {
				return &InferredProfile{
					Instance: extractDomainFromIssuer(dc.Issuer),
					ClientId: dc.ClientId,
					Issuer:   dc.Issuer,
				}
			}
		}
	}
	return nil
}

// extractDomainFromIssuer extracts the domain from an issuer URL like
// https://acme.aliyunidaas.com/api/v2/iauths_system/oauth2 → acme.aliyunidaas.com
func extractDomainFromIssuer(issuer string) string {
	s := issuer
	for _, prefix := range []string{"https://", "http://"} {
		if len(s) > len(prefix) && s[:len(prefix)] == prefix {
			s = s[len(prefix):]
		}
	}
	for i := 0; i < len(s); i++ {
		if s[i] == '/' {
			return s[:i]
		}
	}
	return s
}
