package common

import (
	"testing"

	"github.com/aliyunidaas/alibaba-cloud-idaas/config"
)

func TestProfileLoginContextKeepsExistingEffectiveInputs(t *testing.T) {
	for _, storedClient := range []string{"", "stored-client"} {
		for _, storedScope := range []string{"", "custom-scope"} {
			for _, explicit := range []bool{false, true} {
				dc := &config.OidcTokenProviderDeviceCodeConfig{Issuer: "https://issuer.test/oauth", ClientId: storedClient, Scope: storedScope}
				sts := &config.CloudStsConfig{OidcToken: &config.OidcTokenProviderConfig{OidcTokenProviderDeviceCode: dc}}
				client, scope := "", ""
				if explicit {
					client, scope = "override-client", "override-scope"
				}
				got, err := ResolveProfileLoginContext(sts, "selected", scope, client)
				if err != nil {
					t.Fatal(err)
				}
				// This is the old loginByProfile + DoLogin default sequence.
				wantClient, wantScope := client, scope
				if wantClient == "" {
					wantClient = dc.ClientId
				}
				if wantClient == "" {
					wantClient = DefaultClientId
				}
				if wantScope == "" {
					wantScope = dc.Scope
				}
				if wantScope == "" {
					wantScope = DefaultPamScope
				}
				if got.Issuer != dc.Issuer || got.ClientID != wantClient || got.Scope != wantScope {
					t.Fatalf("changed effective login input: %+v", got)
				}
			}
		}
	}
	if _, err := ResolveProfileLoginContext(&config.CloudStsConfig{Credential: &config.CredentialConfig{}}, "static", "", ""); err == nil {
		t.Fatal("unsupported login profile became a device flow")
	}
}
