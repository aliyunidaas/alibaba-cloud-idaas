package onboard

import (
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/aliyunidaas/alibaba-cloud-idaas/commands/common"
	"github.com/aliyunidaas/alibaba-cloud-idaas/config"
	"github.com/aliyunidaas/alibaba-cloud-idaas/constants"
	"github.com/aliyunidaas/alibaba-cloud-idaas/utils"
	"github.com/urfave/cli/v2"
)

// Exercise the command entry point, HTTPS discovery, both role pages and both
// cloud writers. Only synthetic, isolated credentials are used; any unexpected
// authentication request fails at the local server instead of opening a browser.
func TestOnboardFlowAndPartialFailure(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("AWS_CONFIG_FILE", "")
	t.Setenv("AWS_SHARED_CREDENTIALS_FILE", "")
	t.Setenv("IDAAS_PROFILE", "")
	t.Setenv("AWS_SHARED_CREDENTIALS_FILE", filepath.Join(home, "missing-credentials"))
	var endpoint string
	roleCalls, unexpected := 0, 0
	srv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case common.DiscoveryPath:
			json.NewEncoder(w).Encode(map[string]interface{}{"instance_id": "isolated-test", "default_authorization_server_issuer": endpoint + "/oauth", "developer_api_endpoint": map[string]string{"internet": endpoint}})
		case "/v2/isolated-test/cloudAccountRoles/_/actions/listAssumableCloudAccountRoles":
			roleCalls++
			if r.URL.Query().Get("includePermissionTemplateRoles") != "true" || r.Header.Get("Authorization") == "" {
				t.Error("role discovery mode or authorization missing")
			}
			if r.URL.Query().Get("nextToken") == "" {
				fmt.Fprint(w, `{"entities":[{"cloudAccountRoleId":"ali","cloudAccountRoleName":"ordinary","cloudAccountRoleExternalId":"acs:ram::123456789012:role/ordinary","cloudAccountVendorType":"alibaba_cloud"}],"nextToken":"next+/=","totalCount":2}`)
			} else {
				if r.URL.Query().Get("nextToken") != "next+/=" {
					t.Error("cursor was not preserved")
				}
				fmt.Fprint(w, `{"entities":[{"cloudAccountRoleId":"aws","cloudAccountRoleName":"template","cloudAccountRoleExternalId":"arn:aws-cn:iam::123456789012:role/template","cloudAccountVendorType":"aws","createdByPermissionTemplate":true}],"totalCount":2}`)
			}
		default:
			unexpected++
			http.Error(w, "unexpected local request", http.StatusBadRequest)
		}
	}))
	defer srv.Close()
	endpoint = srv.URL
	certFile := filepath.Join(home, "ca.pem")
	if err := os.WriteFile(certFile, pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: srv.Certificate().Raw}), 0600); err != nil {
		t.Fatal(err)
	}
	previousRoots := utils.RootCertificates
	utils.RootCertificates = certFile
	t.Cleanup(func() { utils.RootCertificates = previousRoots })
	provider := &config.OidcTokenProviderConfig{TokenType: "access_token", OidcTokenProviderDeviceCode: &config.OidcTokenProviderDeviceCodeConfig{Issuer: endpoint + "/oauth", ClientId: common.DefaultClientId, Scope: common.DefaultPamScope, AutoOpenUrl: true, ShowQrCode: true, SmallQrCode: true}}
	claims := fmt.Sprintf(`{"exp":%d,"sub":"synthetic"}`, time.Now().Add(time.Hour).Unix())
	jwt := "e30." + base64.RawURLEncoding.EncodeToString([]byte(claims)) + ".synthetic"
	cache := &utils.StringWithTime{CacheTime: time.Now().UnixMilli(), Content: jwt}
	content, err := cache.Marshal()
	if err != nil {
		t.Fatal(err)
	}
	if err = utils.WriteCacheFileWithEncryption(constants.CategoryOidcToken, provider.GetCacheKey(), content); err != nil {
		t.Fatal(err)
	}
	broker := filepath.Join(home, "broker.json")
	aliFile := filepath.Join(home, ".aliyun", "config.json")
	if err = os.MkdirAll(filepath.Dir(aliFile), 0700); err != nil {
		t.Fatal(err)
	}
	original := `{"current":"original","profiles":[{"name":"original","mode":"AK"}]}`
	if err = os.WriteFile(aliFile, []byte(original), 0600); err != nil {
		t.Fatal(err)
	}
	runs := 0
	var reportFile string
	run := func() error {
		runs++
		reportFile = filepath.Join(home, fmt.Sprintf("result-%d.json", runs))
		app := &cli.App{Commands: []*cli.Command{BuildCommand()}}
		return app.Run([]string{"idaas", "onboard", "--instance", strings.TrimPrefix(endpoint, "https://"), "--client-id", common.DefaultClientId, "--config", broker, "--target", "aliyun-cli,aws-cli", "--result-file", reportFile})
	}
	if err = run(); err != nil {
		t.Fatal(err)
	}
	checkReport := func(status, aliStatus string) {
		t.Helper()
		data, err := os.ReadFile(reportFile)
		if err != nil {
			t.Fatal(err)
		}
		var report onboardResult
		if err := json.Unmarshal(data, &report); err != nil {
			t.Fatal(err)
		}
		if report.SchemaVersion != 1 || report.Status != status || !report.BrokerConfigSaved || report.ProfileCount != 2 || len(report.Targets) != 2 || report.Targets[0].Status != aliStatus || report.Targets[1].Status != "success" {
			t.Fatalf("inaccurate structured result: %+v", report)
		}
		if strings.Contains(string(data), jwt) || strings.Contains(string(data), "{malformed") {
			t.Fatal("structured report contains token or raw error material")
		}
	}
	checkReport("success", "success")
	cfg, err := config.ReadCloudCredentialConfig(broker)
	if err != nil || len(cfg.Profile) != 2 || cfg.Version != config.Version1 {
		t.Fatalf("invalid broker result: %v", err)
	}
	ali, _ := os.ReadFile(aliFile)
	var root map[string]interface{}
	if err = json.Unmarshal(ali, &root); err != nil || root["current"] != "original" {
		t.Fatal("changed the original Alibaba default")
	}
	awsFile := filepath.Join(home, ".aws", "config")
	aws, err := os.ReadFile(awsFile)
	if err != nil || !strings.Contains(string(aws), "region = cn-north-1") || strings.Contains(string(aws), "[default]") || !strings.Contains(string(aws), "--config "+broker) {
		t.Fatal("invalid AWS named profile")
	}
	if err = os.WriteFile(aliFile, []byte("{malformed"), 0600); err != nil {
		t.Fatal(err)
	}
	err = run()
	if err == nil || !strings.Contains(err.Error(), "partially completed") || !strings.Contains(err.Error(), "aliyun-cli") {
		t.Fatalf("partial failure must be observable: %v", err)
	}
	checkReport("partial_success", "failed")
	ali, _ = os.ReadFile(aliFile)
	awsAfter, _ := os.ReadFile(awsFile)
	if string(ali) != "{malformed" || string(awsAfter) != string(aws) || roleCalls != 4 || unexpected != 0 {
		t.Fatal("failure changed existing files, skipped pagination, or triggered unexpected authentication")
	}
	// A non-cloud current profile must not block the unique cloud context.
	// Exercise the actual command with both --instance and --client-id omitted.
	cfg, err = config.ReadCloudCredentialConfig(broker)
	if err != nil {
		t.Fatal(err)
	}
	cfg.Profile["static"] = &config.CloudStsConfig{Credential: &config.CredentialConfig{}}
	cfg.CurrentProfile = "static"
	if err := saveCloudCredentialConfig(broker, cfg); err != nil {
		t.Fatal(err)
	}
	app := &cli.App{Commands: []*cli.Command{BuildCommand()}}
	if err := app.Run([]string{"idaas", "onboard", "--config", broker, "--target", "none", "--result-file", filepath.Join(home, "inferred.json")}); err != nil {
		t.Fatal(err)
	}
	cfg, err = config.ReadCloudCredentialConfig(broker)
	if err != nil || cfg.CurrentProfile != "static" || len(cfg.Profile) != 3 || roleCalls != 6 || unexpected != 0 {
		t.Fatal("inference changed defaults, duplicated identities, or initiated a device flow")
	}
}
