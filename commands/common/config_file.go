package common

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/aliyunidaas/alibaba-cloud-idaas/config"
	"os"
	"path/filepath"
	"strings"
)

// DecodeConfigObject preserves large numeric extension fields during a merge.
func DecodeConfigObject(data []byte) (map[string]interface{}, error) {
	if !json.Valid(data) {
		return nil, fmt.Errorf("invalid JSON configuration")
	}
	var root map[string]interface{}
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.UseNumber()
	if err := decoder.Decode(&root); err != nil {
		return nil, err
	}
	if root == nil {
		return nil, fmt.Errorf("configuration must be a JSON object")
	}
	return root, nil
}

// UpdateConfigFile serializes this CLI's writers and detects intervening edits.
// Existing symlinks are followed explicitly; their targets are atomically replaced.
// It is deliberately not used for OAuth or credential caches.
func UpdateConfigFile(filename string, update func([]byte) ([]byte, error)) error {
	path, err := filepath.Abs(filename)
	if err != nil {
		return err
	}
	if err = os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return err
	}
	// Resolve parent links as well as a possible file link, including dangling links.
	parent, err := filepath.EvalSymlinks(filepath.Dir(path))
	if err != nil {
		return err
	}
	path = filepath.Join(parent, filepath.Base(path))
	if info, e := os.Lstat(path); e == nil && info.Mode()&os.ModeSymlink != 0 {
		path, err = filepath.EvalSymlinks(path)
		if err != nil {
			return fmt.Errorf("resolve config symlink: %w", err)
		}
	} else if e != nil && !os.IsNotExist(e) {
		return e
	}
	lock, err := os.OpenFile(path+".idaas.lock", os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
	if err != nil {
		return fmt.Errorf("config is locked or cannot be locked (%s): %w", path, err)
	}
	lock.Close()
	defer os.Remove(path + ".idaas.lock")
	if info, e := os.Stat(path); e == nil && !info.Mode().IsRegular() {
		return fmt.Errorf("config is not a regular file: %s", path)
	} else if e != nil && !os.IsNotExist(e) {
		return e
	}
	before, err := os.ReadFile(path)
	exists := err == nil
	if err != nil && !os.IsNotExist(err) {
		return err
	}
	next, err := update(before)
	if err != nil {
		return err
	}
	if exists && bytes.Equal(before, next) {
		return nil
	}
	temp, err := os.CreateTemp(filepath.Dir(path), ".idaas-config-*")
	if err != nil {
		return err
	}
	name := temp.Name()
	defer os.Remove(name)
	if err = temp.Chmod(0600); err == nil {
		_, err = temp.Write(next)
	}
	if err == nil {
		err = temp.Sync()
	}
	closeErr := temp.Close()
	if err != nil {
		return err
	}
	if closeErr != nil {
		return closeErr
	}
	// A non-cooperating editor is not covered by the lock. Refuse a known stale snapshot.
	current, e := os.ReadFile(path)
	if (exists && (e != nil || !bytes.Equal(current, before))) || (!exists && !os.IsNotExist(e)) {
		return fmt.Errorf("config changed while updating %s; retry after other writers finish", path)
	}
	if info, e := os.Lstat(path); e == nil && info.Mode()&os.ModeSymlink != 0 {
		return fmt.Errorf("config path became a symlink: %s", path)
	}
	return os.Rename(name, path)
}

func CredentialProcess(executable, filename, profile string) (string, error) {
	exe, err := filepath.Abs(executable)
	if err != nil {
		return "", err
	}
	file, err := filepath.Abs(filename)
	if err != nil {
		return "", err
	}
	quote := func(s string) (string, error) {
		if strings.ContainsAny(s, "\"\r\n\x00") {
			return "", fmt.Errorf("credential process argument contains unsupported quotes or control characters")
		}
		if !strings.ContainsAny(s, " \t") {
			return s, nil
		}
		return "\"" + s + "\"", nil
	}
	e, err := quote(exe)
	if err != nil {
		return "", err
	}
	f, err := quote(file)
	if err != nil {
		return "", err
	}
	p, err := quote(profile)
	if err != nil {
		return "", err
	}
	return e + " fetch-token --profile " + p + " --config " + f, nil
}

func AWSConfigFile() (string, error) {
	if path := os.Getenv("AWS_CONFIG_FILE"); path != "" {
		return filepath.Abs(path)
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".aws", "config"), nil
}

// CredentialProcessMatches checks arguments, not profile-name substrings. Backslashes
// remain literal so Windows paths are not interpreted as POSIX shell escapes.
func CredentialProcessMatches(command, filename, profile string) bool {
	var args []string
	var token strings.Builder
	quoted := false
	for _, r := range command {
		if r == '"' {
			quoted = !quoted
			continue
		}
		if (r == ' ' || r == '\t') && !quoted {
			if token.Len() > 0 {
				args = append(args, token.String())
				token.Reset()
			}
			continue
		}
		token.WriteRune(r)
	}
	if quoted {
		return false
	}
	if token.Len() > 0 {
		args = append(args, token.String())
	}
	if len(args) < 4 || args[1] != "fetch-token" {
		return false
	}
	foundProfile := ""
	foundConfig := ""
	for i := 2; i < len(args); i++ {
		if (args[i] == "--profile" || args[i] == "-p" || args[i] == "--config" || args[i] == "-c") && i+1 < len(args) {
			flag := args[i]
			i++
			if flag == "--profile" || flag == "-p" {
				foundProfile = args[i]
			} else {
				foundConfig = args[i]
			}
		}
	}
	if foundProfile != profile {
		return false
	}
	if foundConfig == "" {
		var err error
		foundConfig, err = config.GetDefaultCloudCredentialConfigFile()
		if err != nil {
			return false
		}
	}
	a, err := filepath.Abs(foundConfig)
	if err != nil {
		return false
	}
	b, err := filepath.Abs(filename)
	return err == nil && a == b
}
