package configure

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/aliyunidaas/alibaba-cloud-idaas/commands/common"
)

func TestSwitchOnlyChangesSelectedBrokerDefault(t *testing.T) {
	p := filepath.Join(t.TempDir(), "broker.json")
	original := `{"version":"1","current_profile":"one","custom":"keep","large_extension":9007199254740993,"profile":{"one":{"comment":"keep"},"two":{"comment":"also keep"}}}`
	os.WriteFile(p, []byte(original), 0600)
	if err := switchProfile(p, "missing"); err == nil {
		t.Fatal("accepted missing profile")
	}
	b, _ := os.ReadFile(p)
	if string(b) != original {
		t.Fatal("failed selection changed file")
	}
	if err := switchProfile(p, "two"); err != nil {
		t.Fatal(err)
	}
	b, _ = os.ReadFile(p)
	root, err := common.DecodeConfigObject(b)
	if err != nil || root["large_extension"].(json.Number).String() != "9007199254740993" {
		t.Fatal("numeric extension was changed")
	}
	var cfg map[string]interface{}
	json.Unmarshal(b, &cfg)
	if cfg["current_profile"] != "two" || cfg["custom"] != "keep" {
		t.Fatal(string(b))
	}
}
