package common

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestConfigUpdatePreservesSymlinkAndDetectsExternalEdit(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "config")
	os.WriteFile(path, []byte("old"), 0600)
	link := filepath.Join(root, "link")
	if e := os.Symlink(path, link); e != nil {
		t.Skip(e)
	}
	if e := UpdateConfigFile(link, func(b []byte) ([]byte, error) { return append(b, []byte("-new")...), nil }); e != nil {
		t.Fatal(e)
	}
	info, _ := os.Lstat(link)
	if info.Mode()&os.ModeSymlink == 0 {
		t.Fatal("replaced user's symlink")
	}
	b, _ := os.ReadFile(path)
	if string(b) != "old-new" {
		t.Fatal(string(b))
	}
	e := UpdateConfigFile(link, func([]byte) ([]byte, error) {
		os.WriteFile(path, []byte("other-editor"), 0600)
		return []byte("stale"), nil
	})
	if e == nil {
		t.Fatal("must reject intervening edit")
	}
	b, _ = os.ReadFile(path)
	if string(b) != "other-editor" {
		t.Fatal("overwrote another editor")
	}
	e = UpdateConfigFile(path, func([]byte) ([]byte, error) { return nil, errors.New("invalid format") })
	if e == nil {
		t.Fatal("missing validation error")
	}
	b, _ = os.ReadFile(path)
	if string(b) != "other-editor" {
		t.Fatal("validation changed config")
	}
}

func TestConfigUpdateLockPreventsCompetingCLIWriter(t *testing.T) {
	p := filepath.Join(t.TempDir(), "config")
	os.WriteFile(p, []byte("old"), 0600)
	e := UpdateConfigFile(p, func(b []byte) ([]byte, error) {
		nested := UpdateConfigFile(p, func([]byte) ([]byte, error) { return []byte("lost"), nil })
		if nested == nil {
			t.Fatal("competing writer was accepted")
		}
		return []byte("new"), nil
	})
	if e != nil {
		t.Fatal(e)
	}
	if _, e = os.Stat(p + ".idaas.lock"); !os.IsNotExist(e) {
		t.Fatal("lock not removed")
	}
}

func TestCredentialProcessPreservesArgumentBoundariesAndIdentity(t *testing.T) {
	root := t.TempDir()
	p := filepath.Join(root, "配置 space.json")
	command, e := CredentialProcess(filepath.Join(root, "bin space"), p, "role")
	if e != nil {
		t.Fatal(e)
	}
	if !strings.Contains(command, "\""+p+"\"") {
		t.Fatal(command)
	}
	if !CredentialProcessMatches(command, p, "role") {
		t.Fatal("quoted command did not match")
	}
	if CredentialProcessMatches(command, p, "role-other") || CredentialProcessMatches(command, filepath.Join(root, "other"), "role") {
		t.Fatal("matched a different identity")
	}
	if _, e = CredentialProcess("bad\npath", p, "role"); e == nil {
		t.Fatal("control character accepted")
	}
}

func TestDecodeConfigObjectPreservesLargeNumbersAndRejectsNonObjects(t *testing.T) {
	root, err := DecodeConfigObject([]byte(`{"extension":9007199254740993}`))
	if err != nil {
		t.Fatal(err)
	}
	encoded, err := json.Marshal(root)
	if err != nil || string(encoded) != `{"extension":9007199254740993}` {
		t.Fatal("numeric extension lost precision")
	}
	for _, data := range []string{"null", "[]", "{}{}", "broken"} {
		if _, err := DecodeConfigObject([]byte(data)); err == nil {
			t.Fatalf("accepted non-object input %s", data)
		}
	}
}
