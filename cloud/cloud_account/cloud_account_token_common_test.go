package cloud_account

import (
	"encoding/json"
	"testing"
)

func TestUnmarshalTencentCloudToken(t *testing.T) {
	body := `{
  "cloudAccountId": "10001",
  "cloudAccountVendorType": "tencent_cloud",
  "cloudAccountRoleAccessCredential": {
    "accessCredentialExpiresAt": 1900000000,
    "tencentCloudStsToken": {
      "tmpSecretId": "AKIDxxxxxxxx",
      "tmpSecretKey": "Qgoxxxxxxxx",
      "token": "tokxxxxxxxx",
      "expiration": "2026-09-18T12:00:00Z"
    }
  }
}`
	token, err := UnmarshalCloudAccountToken(body)
	if err != nil {
		t.Fatalf("UnmarshalCloudAccountToken failed: %v", err)
	}
	if !token.IsTencentCloudToken() {
		t.Error("expected IsTencentCloudToken() == true")
	}
	if token.IsAlibabaCloudToken() {
		t.Error("expected IsAlibabaCloudToken() == false")
	}
	if token.IsAwsToken() {
		t.Error("expected IsAwsToken() == false")
	}
	sts := token.CloudAccountRoleAccessCredential.TencentCloudStsToken
	if sts == nil {
		t.Fatal("expected TencentCloudStsToken to be parsed")
	}
	if sts.SecretId != "AKIDxxxxxxxx" || sts.SecretKey != "Qgoxxxxxxxx" || sts.SessionToken != "tokxxxxxxxx" {
		t.Errorf("unexpected credential fields: %+v", sts)
	}
}

func TestIsTencentCloudTokenNegative(t *testing.T) {
	var token CloudAccountToken
	if token.IsTencentCloudToken() {
		t.Error("expected false for nil access credential")
	}
	token.CloudAccountRoleAccessCredential = &CloudAccountTokenCloudAccountRoleAccessCredential{
		AwsStsToken: &CloudAccountTokenAwsStsToken{AccessKeyId: "x"},
	}
	if token.IsTencentCloudToken() {
		t.Error("expected false when only AwsStsToken present")
	}
}

func TestTencentCloudFieldRoundTrip(t *testing.T) {
	token := &CloudAccountToken{
		CloudAccountVendorType: "tencent_cloud",
		CloudAccountRoleAccessCredential: &CloudAccountTokenCloudAccountRoleAccessCredential{
			TencentCloudStsToken: &CloudAccountTokenTencentCloudStsToken{
				SecretId:     "AKIDroundtrip",
				SecretKey:    "k",
				SessionToken: "s",
				Expiration:   "e",
			},
		},
	}
	marshaled, err := token.Marshal()
	if err != nil {
		t.Fatalf("Marshal failed: %v", err)
	}
	var raw map[string]json.RawMessage
	if err := json.Unmarshal([]byte(marshaled), &raw); err != nil {
		t.Fatalf("unmarshal raw failed: %v", err)
	}
	if _, ok := raw["cloudAccountRoleAccessCredential"]; !ok {
		t.Fatal("missing cloudAccountRoleAccessCredential in marshaled json")
	}
	parsed, err := UnmarshalCloudAccountToken(marshaled)
	if err != nil {
		t.Fatalf("UnmarshalCloudAccountToken failed: %v", err)
	}
	if !parsed.IsTencentCloudToken() {
		t.Error("expected IsTencentCloudToken() == true after round trip")
	}
}
