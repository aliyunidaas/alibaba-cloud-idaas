package show

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/aliyunidaas/alibaba-cloud-idaas/commands/common"
	"github.com/aliyunidaas/alibaba-cloud-idaas/config"
	"github.com/urfave/cli/v2"
)

func TestStatusContextReportsIndependentDefaultsWithoutLogin(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("AWS_PROFILE", "explicit-aws")
	awsPath := filepath.Join(home, "custom-aws-config")
	t.Setenv("AWS_CONFIG_FILE", awsPath)
	brokerPath := filepath.Join(home, "broker.json")
	command, err := common.CredentialProcess("/synthetic/idaas", brokerPath, "broker-default")
	if err != nil {
		t.Fatal(err)
	}
	if err = os.WriteFile(awsPath, []byte("[default]\ncredential_process = "+command+"\n"), 0600); err != nil {
		t.Fatal(err)
	}
	if err = os.MkdirAll(filepath.Join(home, ".aliyun"), 0700); err != nil {
		t.Fatal(err)
	}
	if err = os.WriteFile(filepath.Join(home, ".aliyun", "config.json"), []byte(`{"current":"cloud-default"}`), 0600); err != nil {
		t.Fatal(err)
	}
	profile := &config.CloudStsConfig{CloudAccount: &config.CloudAccountTokenConfig{AccessTokenProvider: &config.OidcTokenProviderConfig{OidcTokenProviderDeviceCode: &config.OidcTokenProviderDeviceCodeConfig{Issuer: "https://invalid.test/oauth", ClientId: "custom-client"}}}}
	cfg := &config.CloudCredentialConfig{CurrentProfile: "broker-default", Profile: map[string]*config.CloudStsConfig{"broker-default": profile}}
	got := statusContext(cfg, profile, "broker-default", brokerPath)
	if got["client_id"] != "custom-client" || got["client_id_source"] != "profile:broker-default" || got["client_context"] != "profile_login" || got["issuer"] != "https://invalid.test/oauth" || got["config_state"] != "loaded" || got["cache_readability"] != "not_checked" || got["login_state"] != "not_checked" || got["cloud_api_verification"] != "not_checked" || got["defaults_differ"] != true || got["aws_default_idaas_profile"] != "broker-default" || got["aws_environment_profile"] != "explicit-aws" || got["aws_config_file"] != awsPath {
		t.Fatalf("incorrect metadata: %+v", got)
	}
	if _, err = os.Stat(filepath.Join(home, ".alibaba_cloud_idaas_seed1")); !os.IsNotExist(err) {
		t.Fatal("read-only status touched credential storage")
	}
}

func TestStatusUsesSelectedOidcProfileWithoutBorrowingOtherClients(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("AWS_CONFIG_FILE", filepath.Join(home, "missing-aws"))
	t.Setenv("AWS_PROFILE", "")
	selected := &config.CloudStsConfig{OidcToken: &config.OidcTokenProviderConfig{OidcTokenProviderDeviceCode: &config.OidcTokenProviderDeviceCodeConfig{Issuer: "https://selected.test/oauth"}}}
	other := &config.CloudStsConfig{CloudAccount: &config.CloudAccountTokenConfig{AccessTokenProvider: &config.OidcTokenProviderConfig{OidcTokenProviderDeviceCode: &config.OidcTokenProviderDeviceCodeConfig{Issuer: "https://other.test/oauth", ClientId: "other-client"}}}}
	cfg := &config.CloudCredentialConfig{CurrentProfile: "other", Profile: map[string]*config.CloudStsConfig{"selected": selected, "other": other}}
	got := statusContext(cfg, selected, "selected", filepath.Join(home, "broker.json"))
	if got["client_id"] != common.DefaultClientId || got["client_id_source"] != "built-in default" || got["issuer"] != "https://selected.test/oauth" {
		t.Fatalf("selected profile used another profile's context: %+v", got)
	}
}

func TestTokenStubNowFailsExplicitlyAndMissingProfileDoesNotPanic(t *testing.T) {
	app := &cli.App{Commands: []*cli.Command{BuildCommand()}}
	if err := app.Run([]string{"idaas", "show", "token"}); err == nil || !strings.Contains(err.Error(), "does not expose credentials") {
		t.Fatalf("expected actionable explicit error: %v", err)
	}
	file := filepath.Join(t.TempDir(), "broker.json")
	if err := os.WriteFile(file, []byte(`{"version":"1","current_profile":"missing","profile":{"null-profile":null}}`), 0600); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"missing", "null-profile"} {
		if err := showStatus(name, file, true); err == nil || !strings.Contains(err.Error(), "not found or invalid") {
			t.Fatal("invalid profile was accepted")
		}
	}
}

func TestCacheMetadataRequiresKnownCategoryForNameFilter(t *testing.T) {
	if err := listCacheMetadata("", "needle"); err == nil || !strings.Contains(err.Error(), "--category") {
		t.Fatal("name filter without category was silently ignored")
	}
	if err := listCacheMetadata("../outside", "needle"); err == nil {
		t.Fatal("metadata query accepted an unknown/traversing category")
	}
}
