package common

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/aliyunidaas/alibaba-cloud-idaas/config"
)

func registrationProfile(domain, client string) *config.CloudStsConfig {
	return &config.CloudStsConfig{CloudAccount: &config.CloudAccountTokenConfig{InstanceId: domain, AccessTokenProvider: &config.OidcTokenProviderConfig{TokenType: "access_token", OidcTokenProviderDeviceCode: &config.OidcTokenProviderDeviceCodeConfig{Issuer: "https://" + domain + "/api/v2/oauth2", ClientId: client, Scope: DefaultPamScope}}}}
}

func TestRegistrationContextDoesNotCrossInstances(t *testing.T) {
	t.Setenv("IDAAS_PROFILE", "")
	p := filepath.Join(t.TempDir(), "config.json")
	cfg := &config.CloudCredentialConfig{Version: "1", CurrentProfile: "a", Profile: map[string]*config.CloudStsConfig{"a": registrationProfile("a.example", "client-a"), "b": registrationProfile("b.example", "client-b")}}
	write := func() {
		b, _ := json.Marshal(cfg)
		if err := os.WriteFile(p, b, 0600); err != nil {
			t.Fatal(err)
		}
	}
	write()
	for i := 0; i < 100; i++ {
		c, e := ResolveRegistrationContext(p, "https://b.example/", "")
		if e != nil || c.Instance != "b.example" || c.ClientID != "client-b" {
			t.Fatalf("cross instance: %+v %v", c, e)
		}
	}
	c, e := ResolveRegistrationContext(p, "new.example", "")
	if e != nil || c.ClientID != DefaultClientId || c.ClientIDSource != "built-in default" {
		t.Fatalf("new instance: %+v %v", c, e)
	}
	c, e = ResolveRegistrationContext(p, "", "override")
	if e != nil || c.Instance != "a.example" || c.ClientID != "override" {
		t.Fatalf("explicit client: %+v %v", c, e)
	}
	cfg.CurrentProfile = ""
	write()
	if _, e = ResolveRegistrationContext(p, "", ""); e == nil || !strings.Contains(e.Error(), "ambiguous") {
		t.Fatalf("expected ambiguity, got %v", e)
	}
	cfg.Profile["c"] = registrationProfile("b.example", "another-client")
	write()
	if _, e = ResolveRegistrationContext(p, "b.example", ""); e == nil {
		t.Fatal("ambiguous clients must not be selected randomly")
	}
	t.Setenv("IDAAS_PROFILE", "b")
	c, e = ResolveRegistrationContext(p, "", "")
	if e != nil || c.ClientID != "client-b" {
		t.Fatalf("selected profile: %+v %v", c, e)
	}
	c, e = ResolveRegistrationContext(p, "a.example", "")
	if e != nil || c.ClientID != "client-a" {
		t.Fatalf("explicit instance must scope environment selection: %+v %v", c, e)
	}
}

func TestRegistrationContextRejectsInvalidConfigAndDomain(t *testing.T) {
	p := filepath.Join(t.TempDir(), "config.json")
	os.WriteFile(p, []byte("broken"), 0600)
	if _, e := ResolveRegistrationContext(p, "a.example", ""); e == nil {
		t.Fatal("must report malformed config")
	}
	if _, e := ResolveRegistrationContext(p, "a.example", "explicit"); e != nil {
		t.Fatal(e)
	}
	for _, input := range []string{"https://u:p@example.com", "https://example.com/path", "https://example.com?q=x"} {
		if _, e := NormalizeInstance(input); e == nil {
			t.Fatal(input)
		}
	}
}

func TestImplicitNonCloudDefaultUsesOnlyUnambiguousCloudContext(t *testing.T) {
	t.Setenv("IDAAS_PROFILE", "")
	path := filepath.Join(t.TempDir(), "broker.json")
	cfg := &config.CloudCredentialConfig{Version: config.Version1, CurrentProfile: "static", Profile: map[string]*config.CloudStsConfig{
		"static": {Credential: &config.CredentialConfig{}},
		"role":   registrationProfile("a.example", "client-a"),
	}}
	write := func() []byte {
		t.Helper()
		data, err := json.Marshal(cfg)
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, data, 0600); err != nil {
			t.Fatal(err)
		}
		return data
	}
	original := write()
	got, err := ResolveRegistrationContext(path, "", "")
	if err != nil || got.Profile != "role" || got.Instance != "a.example" || got.ClientID != "client-a" || got.ClientIDSource != "profile:role" {
		t.Fatalf("non-cloud default blocked a unique cloud context: %+v %v", got, err)
	}
	after, err := os.ReadFile(path)
	if err != nil || string(after) != string(original) {
		t.Fatal("context lookup changed the original default/config")
	}
	cfg.Profile["same-context"] = registrationProfile("a.example", "client-a")
	write()
	if _, err := ResolveRegistrationContext(path, "", ""); err != nil {
		t.Fatal("multiple roles in the same context are not ambiguous")
	}
	for _, other := range []*config.CloudStsConfig{registrationProfile("b.example", "client-a"), registrationProfile("a.example", "client-b")} {
		cfg.Profile["other"] = other
		write()
		if _, err := ResolveRegistrationContext(path, "", ""); err == nil || !strings.Contains(err.Error(), "ambiguous") {
			t.Fatalf("different contexts must remain ambiguous: %v", err)
		}
	}
	delete(cfg.Profile, "other")
	write()
	for _, selected := range []string{"static", "missing"} {
		t.Setenv("IDAAS_PROFILE", selected)
		if _, err := ResolveRegistrationContext(path, "", ""); err == nil {
			t.Fatalf("explicit %s selection silently changed identity", selected)
		}
	}
	t.Setenv("IDAAS_PROFILE", "")
	for _, selected := range []string{"missing", "invalid-cloud"} {
		cfg.CurrentProfile = selected
		cfg.Profile["invalid-cloud"] = &config.CloudStsConfig{CloudAccount: &config.CloudAccountTokenConfig{}}
		write()
		if _, err := ResolveRegistrationContext(path, "", ""); err == nil {
			t.Fatalf("broken cloud/default %s silently changed identity", selected)
		}
	}
}
