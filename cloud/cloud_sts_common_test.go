package cloud

import (
	"testing"

	"github.com/aliyunidaas/alibaba-cloud-idaas/cloud/cloud_account"
)

func TestMarshalTencentCloudCliCredential(t *testing.T) {
	token := &cloud_account.CloudAccountTokenTencentCloudStsToken{
		SecretId:     "AKIDxxxxxxxx",
		SecretKey:    "Qgoxxxxxxxx",
		SessionToken: "tokxxxxxxxx",
		Expiration:   "2026-09-18T12:00:00Z",
	}
	got, err := MarshalTencentCloudCliCredential(token)
	if err != nil {
		t.Fatalf("MarshalTencentCloudCliCredential failed: %v", err)
	}
	want := `{"secretId":"AKIDxxxxxxxx","secretKey":"Qgoxxxxxxxx","token":"tokxxxxxxxx"}`
	if got != want {
		t.Errorf("credential json mismatch:\n got: %s\nwant: %s", got, want)
	}
}

func TestMarshalTencentCloudCliCredentialNil(t *testing.T) {
	if _, err := MarshalTencentCloudCliCredential(nil); err == nil {
		t.Error("expected error for nil token, got nil")
	}
}
