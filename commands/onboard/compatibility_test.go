package onboard

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/aliyunidaas/alibaba-cloud-idaas/commands/common"
	"github.com/aliyunidaas/alibaba-cloud-idaas/config"
)

func TestAWSPartitionsAndExistingDefaults(t *testing.T) {
	for _, tc := range []struct{ partition, region string }{{"aws", "us-east-1"}, {"aws-cn", "cn-north-1"}, {"aws-us-gov", "us-gov-west-1"}} {
		t.Run(tc.partition, func(t *testing.T) {
			home := t.TempDir()
			t.Setenv("HOME", home)
			t.Setenv("AWS_CONFIG_FILE", "")
			t.Setenv("AWS_SHARED_CREDENTIALS_FILE", "")
			file := filepath.Join(home, ".aws", "config")
			os.MkdirAll(filepath.Dir(file), 0700)
			original := "[default]\nregion = eu-west-1\noutput = json\n[profile original]\nsso_session = corporate\n[sso-session corporate]\nsso_region = eu-west-1\n"
			os.WriteFile(file, []byte(original), 0600)
			p := generatedProfile{Name: "test", VendorType: vendorAws, RoleARN: "arn:" + tc.partition + ":iam::123456789012:role/demo"}
			if err := writeAwsCliProfiles([]generatedProfile{p}, filepath.Join(home, "broker.json")); err != nil {
				t.Fatal(err)
			}
			data, _ := os.ReadFile(file)
			s := string(data)
			if !strings.HasPrefix(s, original) || strings.Count(s, "[default]") != 1 || !strings.Contains(s, "region = "+tc.region) {
				t.Fatal(s)
			}
			// Re-running preserves an explicitly customized compatible region and unknown settings.
			alternative := map[string]string{"aws": "ap-southeast-1", "aws-cn": "cn-northwest-1", "aws-us-gov": "us-gov-east-1"}[tc.partition]
			s = strings.Replace(s, "region = "+tc.region, "region = "+alternative, 1)
			s += "custom_setting = keep\n"
			os.WriteFile(file, []byte(s), 0600)
			if err := writeAwsCliProfiles([]generatedProfile{p}, filepath.Join(home, "broker.json")); err != nil {
				t.Fatal(err)
			}
			data, _ = os.ReadFile(file)
			if !strings.Contains(string(data), "region = "+alternative) || !strings.Contains(string(data), "custom_setting = keep") {
				t.Fatal(string(data))
			}
		})
	}
}

func TestAWSWrongPartitionRegionFailsWithoutChangingFile(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("AWS_CONFIG_FILE", "")
	t.Setenv("AWS_SHARED_CREDENTIALS_FILE", "")
	file := filepath.Join(home, ".aws", "config")
	os.MkdirAll(filepath.Dir(file), 0700)
	cfg := filepath.Join(home, "broker.json")
	cmd, err := credentialCommand(cfg, "china", false)
	if err != nil {
		t.Fatal(err)
	}
	original := "[profile china]\ncredential_process = " + cmd + "\nregion = us-east-1\n"
	os.WriteFile(file, []byte(original), 0600)
	p := generatedProfile{Name: "china", VendorType: vendorAws, RoleARN: "arn:aws-cn:iam::123456789012:role/demo"}
	if err := writeAwsCliProfiles([]generatedProfile{p}, cfg); err == nil {
		t.Fatal("wrong region was accepted")
	}
	data, _ := os.ReadFile(file)
	if string(data) != original {
		t.Fatal("overwrote incompatible region silently")
	}
	p.AWSRegion = "cn-north-1"
	if err := writeAwsCliProfiles([]generatedProfile{p}, cfg); err != nil {
		t.Fatal(err)
	}
}

func TestMalformedAndForeignCloudProfilesAreNotOverwritten(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("AWS_CONFIG_FILE", "")
	t.Setenv("AWS_SHARED_CREDENTIALS_FILE", "")
	file := filepath.Join(home, ".aliyun", "config.json")
	os.MkdirAll(filepath.Dir(file), 0700)
	for _, original := range []string{"{invalid", "null", `{"profiles":[{"name":"demo","mode":"AK","access_key_id":"synthetic"}]}`} {
		os.WriteFile(file, []byte(original), 0600)
		e := writeAliyunCliProfiles([]generatedProfile{{Name: "demo", VendorType: vendorAlibaba}}, filepath.Join(home, "broker.json"), "https://eiam-developerapi.cn-hangzhou.aliyuncs.com")
		if e == nil {
			t.Fatal("expected validation/collision error")
		}
		data, _ := os.ReadFile(file)
		if string(data) != original {
			t.Fatal("changed existing config on failure")
		}
	}
	aws := filepath.Join(home, ".aws", "config")
	os.MkdirAll(filepath.Dir(aws), 0700)
	original := "[profile demo]\nsso_session = original\n"
	os.WriteFile(aws, []byte(original), 0600)
	if e := writeAwsCliProfiles([]generatedProfile{{Name: "demo", VendorType: vendorAws, RoleARN: "arn:aws:iam::123456789012:role/demo"}}, filepath.Join(home, "broker.json")); e == nil {
		t.Fatal("SSO identity overwritten")
	}
	data, _ := os.ReadFile(aws)
	if string(data) != original {
		t.Fatal("SSO config changed")
	}
}

func TestGenerationPreservesExistingIdentityNamesAndUnknownFields(t *testing.T) {
	file := filepath.Join(t.TempDir(), "config.json")
	issuer := "https://a.example/oauth"
	roles := []assumableRole{{CloudAccountRoleName: "Same Role", CloudAccountRoleExternalId: "acs:ram::111:role/same", CloudAccountVendorType: vendorAlibaba}}
	first, _, e := generateAndSaveProfiles(file, "", "a", "https://api.a", issuer, common.DefaultClientId, roles)
	if e != nil {
		t.Fatal(e)
	}
	var root map[string]interface{}
	data, _ := os.ReadFile(file)
	json.Unmarshal(data, &root)
	root["unknown_root"] = "preserve"
	profiles := root["profile"].(map[string]interface{})
	profiles[first[0].Name].(map[string]interface{})["unknown_profile"] = "preserve"
	original, _ := json.Marshal(root)
	os.WriteFile(file, original, 0600)
	second, _, e := generateAndSaveProfiles(file, "", "b", "https://api.b", "https://b.example/oauth", common.DefaultClientId, roles)
	if e != nil {
		t.Fatal(e)
	}
	if second[0].Name == first[0].Name {
		t.Fatal("cross-instance role reused an existing name")
	}
	again, _, e := generateAndSaveProfiles(file, "", "b", "https://api.b", "https://b.example/oauth", common.DefaultClientId, roles)
	if e != nil || again[0].Name != second[0].Name {
		t.Fatalf("unstable repeated name %+v %v", again, e)
	}
	cfg, e := config.ReadCloudCredentialConfig(file)
	if e != nil {
		t.Fatal(e)
	}
	if cfg.Profile[first[0].Name].CloudAccount.InstanceId != "a" || cfg.CurrentProfile != first[0].Name {
		t.Fatal("changed existing default or identity")
	}
	// Public/VPC endpoint changes are explicit configuration choices, not identity changes.
	_, _, e = generateAndSaveProfiles(file, "", "b", "https://vpc.api.b", "https://b.example/oauth", common.DefaultClientId, roles)
	if e != nil {
		t.Fatal(e)
	}
	cfg, e = config.ReadCloudCredentialConfig(file)
	if e != nil || cfg.Profile[second[0].Name].CloudAccount.DeveloperApiEndpoint != "https://vpc.api.b" || cfg.Profile[first[0].Name].CloudAccount.DeveloperApiEndpoint != "https://api.a" {
		t.Fatal("endpoint choice was ignored or changed another identity")
	}
	data, _ = os.ReadFile(file)
	json.Unmarshal(data, &root)
	if root["unknown_root"] != "preserve" || root["profile"].(map[string]interface{})[first[0].Name].(map[string]interface{})["unknown_profile"] != "preserve" {
		t.Fatal("unknown fields lost")
	}
}

func TestAWSSharedCredentialsCollisionDoesNotChangeConfig(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("AWS_CONFIG_FILE", "")
	t.Setenv("AWS_SHARED_CREDENTIALS_FILE", "")
	shared := filepath.Join(home, "custom-credentials")
	t.Setenv("AWS_SHARED_CREDENTIALS_FILE", shared)
	if err := os.WriteFile(shared, []byte("[demo]\naws_access_key_id = synthetic\naws_secret_access_key = synthetic\n"), 0600); err != nil {
		t.Fatal(err)
	}
	p := generatedProfile{Name: "demo", VendorType: vendorAws, RoleARN: "arn:aws:iam::123456789012:role/demo"}
	if err := writeAwsCliProfiles([]generatedProfile{p}, filepath.Join(home, "broker.json")); err == nil {
		t.Fatal("shared credentials shadowing was not detected")
	}
	if _, err := os.Stat(filepath.Join(home, ".aws", "config")); !os.IsNotExist(err) {
		t.Fatal("wrote cloud config despite credential collision")
	}
}

func TestAWSConfigEnvironmentAndTencentExistingFileProtection(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("AWS_SHARED_CREDENTIALS_FILE", filepath.Join(home, "missing-credentials"))
	custom := filepath.Join(home, "custom", "aws-config")
	t.Setenv("AWS_CONFIG_FILE", custom)
	p := generatedProfile{Name: "demo", VendorType: vendorAws, RoleARN: "arn:aws:iam::123456789012:role/demo"}
	if err := writeAwsCliProfiles([]generatedProfile{p}, filepath.Join(home, "broker.json")); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(custom); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(home, ".aws", "config")); !os.IsNotExist(err) {
		t.Fatal("ignored AWS_CONFIG_FILE")
	}
	tcFile := filepath.Join(home, ".tccli", "demo.credential")
	if err := os.MkdirAll(filepath.Dir(tcFile), 0700); err != nil {
		t.Fatal(err)
	}
	for _, original := range []string{"{invalid", "", `{"secretId":"synthetic"}`} {
		if err := os.WriteFile(tcFile, []byte(original), 0600); err != nil {
			t.Fatal(err)
		}
		// This must reject before contacting IDaaS to fetch any credential.
		if err := writeTencentcloudCliProfiles([]generatedProfile{{Name: "demo", VendorType: vendorTencent}}, "missing-broker.json"); err == nil || !strings.Contains(err.Error(), "already exists") {
			t.Fatalf("expected existing file protection, got %v", err)
		}
		data, err := os.ReadFile(tcFile)
		if err != nil || string(data) != original {
			t.Fatal("changed an existing Tencent identity")
		}
	}
}

func TestTargetsRejectTyposAndNormalizeWhitespace(t *testing.T) {
	roles := []assumableRole{{CloudAccountVendorType: vendorAws}}
	targets, e := validatedTargets(" aws-cli , aws-cli ", roles, true)
	if e != nil || len(targets) != 1 {
		t.Fatalf("%v %v", targets, e)
	}
	for _, s := range []string{"mcp", "aws-cl", "none,aws-cli", "aws-cli,", "aliyun-cli"} {
		if _, e := validatedTargets(s, roles, true); e == nil {
			t.Fatal(s)
		}
	}
}

func TestCloudWritersHandleWhitespaceAccordingToConsumer(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("AWS_CONFIG_FILE", "")
	t.Setenv("AWS_SHARED_CREDENTIALS_FILE", "")
	broker := filepath.Join(home, "配置 space.json")
	p := generatedProfile{Name: "demo", VendorType: vendorAws, RoleARN: "arn:aws:iam::123456789012:role/demo"}
	if err := writeAwsCliProfiles([]generatedProfile{p}, broker); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(filepath.Join(home, ".aws", "config"))
	if err != nil || !strings.Contains(string(data), `--config "`+broker+`"`) {
		t.Fatal("AWS argument boundary lost")
	}
	if err = writeAliyunCliProfiles([]generatedProfile{{Name: "demo", VendorType: vendorAlibaba}}, broker, ""); err == nil || !strings.Contains(err.Error(), "whitespace") {
		t.Fatalf("unsafe Alibaba consumer path must be rejected: %v", err)
	}
	if _, err = os.Stat(filepath.Join(home, ".aliyun", "config.json")); !os.IsNotExist(err) {
		t.Fatal("created unusable Alibaba config")
	}
}
