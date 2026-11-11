package onboard

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/aliyunidaas/alibaba-cloud-idaas/commands/common"
)

type targetResult struct {
	Target    string   `json:"target"`
	Status    string   `json:"status"`
	Files     []string `json:"configuration_files"`
	ErrorCode string   `json:"error_code,omitempty"`
}

type onboardResult struct {
	SchemaVersion     int            `json:"schema_version"`
	Status            string         `json:"status"`
	Stage             string         `json:"stage"`
	ConfigFile        string         `json:"config_file"`
	BrokerConfigSaved bool           `json:"broker_config_saved"`
	ProfileCount      int            `json:"profile_count"`
	Targets           []targetResult `json:"targets"`
	FailureCode       string         `json:"failure_code,omitempty"`
}

// canonicalNewPath also resolves symlinked ancestors of a not-yet-created file.
func canonicalNewPath(path string) (string, error) {
	path, err := filepath.Abs(path)
	if err != nil {
		return "", err
	}
	var tail []string
	for {
		resolved, e := filepath.EvalSymlinks(path)
		if e == nil {
			for i := len(tail) - 1; i >= 0; i-- {
				resolved = filepath.Join(resolved, tail[i])
			}
			return resolved, nil
		}
		if !os.IsNotExist(e) || filepath.Dir(path) == path {
			return "", e
		}
		tail = append(tail, filepath.Base(path))
		path = filepath.Dir(path)
	}
}

func prepareResultFile(filename, broker string) (string, error) {
	if filename == "" {
		return "", nil
	}
	if !strings.EqualFold(filepath.Ext(filename), ".json") {
		return "", fmt.Errorf("--result-file must name a new .json file")
	}
	if _, err := os.Lstat(filename); err == nil {
		return "", fmt.Errorf("result file already exists; choose a new --result-file path")
	} else if !os.IsNotExist(err) {
		return "", err
	}
	path, err := canonicalNewPath(filename)
	if err != nil {
		return "", err
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	aws, err := common.AWSConfigFile()
	if err != nil {
		return "", err
	}
	protected := []string{broker, aws, filepath.Join(home, ".aliyun", "config.json"), filepath.Join(home, ".aws", "credentials"), filepath.Join(home, ".cloud_idaas", "idaas-cli.json"), filepath.Join(home, ".aliyun", "alibaba-cloud-idaas.json")}
	if shared := os.Getenv("AWS_SHARED_CREDENTIALS_FILE"); shared != "" {
		protected = append(protected, shared)
	}
	for _, candidate := range protected {
		resolved, e := canonicalNewPath(candidate)
		if e != nil {
			return "", e
		}
		if resolved == path {
			return "", fmt.Errorf("--result-file must not replace a Broker or cloud CLI configuration")
		}
	}
	return path, nil
}

// writeResultFile publishes a complete report without replacing an existing file.
// Errors are codes only: raw backend errors may contain credential material.
func writeResultFile(filename string, result *onboardResult) error {
	data, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(filename), 0700); err != nil {
		return err
	}
	file, err := os.CreateTemp(filepath.Dir(filename), ".idaas-result-*")
	if err != nil {
		return err
	}
	defer os.Remove(file.Name())
	if _, err = file.Write(data); err == nil {
		err = file.Sync()
	}
	closeErr := file.Close()
	if err != nil {
		return err
	}
	if closeErr != nil {
		return closeErr
	}
	return os.Link(file.Name(), filename)
}

func targetFiles(target string, profiles []generatedProfile) []string {
	files := []string{}
	home, err := os.UserHomeDir()
	if err != nil {
		return files
	}
	switch target {
	case "aliyun-cli":
		files = append(files, filepath.Join(home, ".aliyun", "config.json"))
	case "aws-cli":
		if path, err := common.AWSConfigFile(); err == nil {
			files = append(files, path)
		}
	case "tencentcloud-cli":
		for _, profile := range profiles {
			if strings.EqualFold(profile.VendorType, vendorTencent) {
				files = append(files, filepath.Join(home, ".tccli", profile.Name+".credential"))
			}
		}
	}
	return files
}
