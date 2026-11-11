package onboard

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/aliyunidaas/alibaba-cloud-idaas/config"
)

func TestGenerateProfilesPreservesDefaultsAndTemplateRoles(t *testing.T) {
	path := filepath.Join(t.TempDir(), "profiles.json")
	initial := &config.CloudCredentialConfig{Version: config.Version1, CurrentProfile: "existing", Profile: map[string]*config.CloudStsConfig{"existing": {Comment: "preserve"}}}
	if err := saveCloudCredentialConfig(path, initial); err != nil {
		t.Fatal(err)
	}
	yes := true
	roles := []assumableRole{
		{CloudAccountRoleName: "Template Role", CloudAccountRoleExternalId: "arn:template", CloudAccountVendorType: vendorAlibaba, CreatedByPermissionTemplate: &yes},
		{CloudAccountRoleName: "Ordinary Role", CloudAccountRoleExternalId: "arn:ordinary", CloudAccountVendorType: vendorAlibaba},
	}
	profiles, _, err := generateAndSaveProfiles(path, "", "instance", "https://api.example.com", "https://issuer.example.com", "client", roles)
	if err != nil {
		t.Fatal(err)
	}
	saved, err := config.ReadCloudCredentialConfig(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(profiles) != 2 || len(saved.Profile) != 3 || saved.CurrentProfile != "existing" {
		t.Fatalf("profile/default regression: %+v", saved)
	}
	if saved.Profile["aliyun-template-role"].CloudAccount.CloudAccountRoleExternalId != "arn:template" {
		t.Fatal("template role profile missing")
	}
}

func TestWriteAliyunCliProfilesUsesFirstAliyunProfileAsCurrent(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("AWS_CONFIG_FILE", "")
	t.Setenv("AWS_SHARED_CREDENTIALS_FILE", "")

	profiles := []generatedProfile{
		{Name: "aws-readonly", VendorType: vendorAws, RoleARN: "arn:aws:iam::123456789012:role/readonly"},
		{Name: "aliyun-oss-read", VendorType: vendorAlibaba},
		{Name: "aliyun-admin", VendorType: vendorAlibaba},
	}
	configFilename := filepath.Join(home, ".cloud_idaas", "idaas-cli.json")
	pop := "https://eiam-developerapi.cn-hangzhou.aliyuncs.com"

	if err := writeAliyunCliProfiles(profiles, configFilename, pop); err != nil {
		t.Fatalf("writeAliyunCliProfiles() error = %v", err)
	}

	aliyunConfigFile := filepath.Join(home, ".aliyun", "config.json")
	data, err := os.ReadFile(aliyunConfigFile)
	if err != nil {
		t.Fatalf("read aliyun config: %v", err)
	}
	var config struct {
		Current  string `json:"current"`
		Profiles []struct {
			Name string `json:"name"`
		} `json:"profiles"`
	}
	if err := json.Unmarshal(data, &config); err != nil {
		t.Fatalf("unmarshal aliyun config: %v", err)
	}

	if config.Current != "aliyun-oss-read" {
		t.Fatalf("current = %q, want first Alibaba Cloud profile %q", config.Current, "aliyun-oss-read")
	}
	for _, profile := range config.Profiles {
		if profile.Name == "aws-readonly" {
			t.Fatalf("AWS profile %q must not be written to aliyun-cli config", profile.Name)
		}
	}
}

func TestWriteAwsCliProfilesDoesNotCreateDefault(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("AWS_CONFIG_FILE", "")
	t.Setenv("AWS_SHARED_CREDENTIALS_FILE", "")

	profiles := []generatedProfile{
		{Name: "aliyun-oss-read", VendorType: vendorAlibaba},
		{Name: "aws-readonly", VendorType: vendorAws, RoleARN: "arn:aws:iam::123456789012:role/readonly"},
		{Name: "aws-admin", VendorType: vendorAws, RoleARN: "arn:aws:iam::123456789012:role/admin"},
	}
	configFilename := filepath.Join(home, ".cloud_idaas", "idaas-cli.json")

	if err := writeAwsCliProfiles(profiles, configFilename); err != nil {
		t.Fatalf("writeAwsCliProfiles() error = %v", err)
	}

	awsConfigFile := filepath.Join(home, ".aws", "config")
	data, err := os.ReadFile(awsConfigFile)
	if err != nil {
		t.Fatalf("read aws config: %v", err)
	}
	content := string(data)
	firstNamedProfile := strings.Index(content, "[profile ")
	if firstNamedProfile < 0 {
		t.Fatalf("AWS config has no named profile sections:\n%s", content)
	}
	if strings.Contains(content, "[default]") {
		t.Fatal("onboard must not silently create AWS default")
	}
	if !strings.Contains(content, "[profile aws-readonly]") || !strings.Contains(content, "[profile aws-admin]") {
		t.Fatalf("AWS named profiles are incomplete:\n%s", content)
	}
	if strings.Contains(content, "[profile aliyun-oss-read]") {
		t.Fatalf("Alibaba Cloud profile must not be written to AWS config:\n%s", content)
	}
}

func TestWriteAwsCliProfilesPreservesExistingDefault(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("AWS_CONFIG_FILE", "")
	t.Setenv("AWS_SHARED_CREDENTIALS_FILE", "")

	awsDir := filepath.Join(home, ".aws")
	if err := os.MkdirAll(awsDir, 0700); err != nil {
		t.Fatalf("create aws dir: %v", err)
	}
	awsConfigFile := filepath.Join(awsDir, "config")
	existing := "[default]\nregion = ap-southeast-1\noutput = json\n\n"
	if err := os.WriteFile(awsConfigFile, []byte(existing), 0600); err != nil {
		t.Fatalf("write existing aws config: %v", err)
	}

	profiles := []generatedProfile{{Name: "aws-readonly", VendorType: vendorAws, RoleARN: "arn:aws:iam::123456789012:role/readonly"}}
	configFilename := filepath.Join(home, ".cloud_idaas", "idaas-cli.json")
	if err := writeAwsCliProfiles(profiles, configFilename); err != nil {
		t.Fatalf("writeAwsCliProfiles() error = %v", err)
	}

	data, err := os.ReadFile(awsConfigFile)
	if err != nil {
		t.Fatalf("read aws config: %v", err)
	}
	content := string(data)
	if strings.Count(content, "[default]") != 1 {
		t.Fatalf("default section count = %d, want 1:\n%s", strings.Count(content, "[default]"), content)
	}
	if !strings.Contains(content, "region = ap-southeast-1") || !strings.Contains(content, "output = json") {
		t.Fatalf("existing default section was not preserved:\n%s", content)
	}
}
