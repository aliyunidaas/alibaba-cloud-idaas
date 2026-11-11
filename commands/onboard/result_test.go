package onboard

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestResultFileNeverOverwritesConfigurationOrOtherWriter(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("AWS_CONFIG_FILE", "")
	t.Setenv("AWS_SHARED_CREDENTIALS_FILE", "")
	broker := filepath.Join(home, "broker.json")
	if _, err := prepareResultFile(broker, broker); err == nil {
		t.Fatal("result file may become the new Broker configuration")
	}
	file := filepath.Join(home, "report.json")
	path, err := prepareResultFile(file, broker)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte("other writer"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := writeResultFile(path, &onboardResult{SchemaVersion: 1}); err == nil {
		t.Fatal("overwrote a concurrently created file")
	}
	data, err := os.ReadFile(path)
	if err != nil || string(data) != "other writer" {
		t.Fatal("destroyed another writer's file")
	}
	if _, err := prepareResultFile(path, broker); err == nil {
		t.Fatal("accepted an existing result file")
	}
	if _, err := prepareResultFile(filepath.Join(home, ".alibaba_cloud_idaas_seed1"), broker); err == nil {
		t.Fatal("allowed a non-JSON destination")
	}
}

func TestEarlyFailureProducesStructuredStageWithoutAuthentication(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("AWS_CONFIG_FILE", "")
	t.Setenv("AWS_SHARED_CREDENTIALS_FILE", "")
	file := filepath.Join(home, "failure.json")
	err := runOnboard(&onboardOptions{instance: "invalid.test", clientId: "explicit", target: "mcp", configFile: filepath.Join(home, "broker.json"), resultFile: file})
	if err == nil {
		t.Fatal("invalid target was accepted")
	}
	data, err := os.ReadFile(file)
	if err != nil {
		t.Fatal(err)
	}
	var result onboardResult
	if err := json.Unmarshal(data, &result); err != nil {
		t.Fatal(err)
	}
	if result.Status != "failed" || result.Stage != "validate_options" || result.FailureCode != "validate_options_failed" || result.BrokerConfigSaved {
		t.Fatalf("early failure lost its stage: %+v", result)
	}
	if _, err := os.Stat(filepath.Join(home, ".alibaba_cloud_idaas_seed1")); !os.IsNotExist(err) {
		t.Fatal("validation failure accessed token storage")
	}
}

func TestResultPathCannotAliasBrokerThroughParentSymlink(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("AWS_CONFIG_FILE", "")
	t.Setenv("AWS_SHARED_CREDENTIALS_FILE", "")
	real := filepath.Join(home, "actual")
	if err := os.Mkdir(real, 0700); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(home, "link")
	if err := os.Symlink(real, link); err != nil {
		t.Skip(err)
	}
	if _, err := prepareResultFile(filepath.Join(link, "broker.json"), filepath.Join(real, "broker.json")); err == nil {
		t.Fatal("symlinked parent bypassed config protection")
	}
}
