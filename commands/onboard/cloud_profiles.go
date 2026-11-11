package onboard

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/aliyunidaas/alibaba-cloud-idaas/commands/common"
	"github.com/aliyunidaas/alibaba-cloud-idaas/utils"
	"gopkg.in/ini.v1"
)

func credentialCommand(filename, name string, simple bool) (string, error) {
	self, err := os.Executable()
	if err != nil {
		return "", err
	}
	path, err := filepath.Abs(filename)
	if err != nil {
		return "", err
	}
	// Some Alibaba CLI/credentials-go versions split process_command with Fields.
	// Quoting cannot make those consumers accept whitespace; never persist a broken command.
	if simple && strings.ContainsAny(self+path+name, " \t") {
		return "", fmt.Errorf("aliyun external credential consumers may not support whitespace in executable/config paths; use a path without spaces or a consumer with quoted-argument support")
	}
	return common.CredentialProcess(self, path, name)
}

func writeAliyunCliProfiles(profiles []generatedProfile, filename, pop string) error {
	home, err := os.UserHomeDir()
	if err != nil {
		return err
	}
	path := filepath.Join(home, ".aliyun", "config.json")
	count := 0
	err = common.UpdateConfigFile(path, func(data []byte) ([]byte, error) {
		root := map[string]interface{}{}
		if len(data) > 0 {
			var err error
			root, err = common.DecodeConfigObject(data)
			if err != nil {
				return nil, fmt.Errorf("invalid aliyun config; existing file was not changed: %w", err)
			}
		}
		var entries []interface{}
		if value, ok := root["profiles"]; ok {
			var valid bool
			entries, valid = value.([]interface{})
			if !valid {
				return nil, fmt.Errorf("aliyun profiles must be an array")
			}
		}
		region := parseRegionFromPop(pop)
		if region == "" {
			region = "cn-hangzhou"
		}
		for _, p := range profiles {
			if !strings.EqualFold(p.VendorType, vendorAlibaba) {
				continue
			}
			command, err := credentialCommand(filename, p.Name, true)
			if err != nil {
				return nil, err
			}
			entry := map[string]interface{}{"name": p.Name, "mode": "External", "region_id": region, "output_format": "json", "language": "en", "process_command": command}
			found := false
			for i, value := range entries {
				old, ok := value.(map[string]interface{})
				if !ok {
					return nil, fmt.Errorf("invalid aliyun profile entry")
				}
				if old["name"] != p.Name {
					continue
				}
				if old["mode"] != "External" {
					return nil, fmt.Errorf("aliyun profile %s already uses another credential provider; choose another --prefix", p.Name)
				}
				oldCommand, _ := old["process_command"].(string)
				if !common.CredentialProcessMatches(oldCommand, filename, p.Name) {
					return nil, fmt.Errorf("aliyun profile %s is not an IDaaS profile; choose another --prefix", p.Name)
				}
				for k, v := range old {
					entry[k] = v
				}
				entry["process_command"] = command
				entries[i] = entry
				found = true
				break
			}
			if !found {
				entries = append(entries, entry)
			}
			count++
		}
		if count == 0 {
			return nil, fmt.Errorf("no Alibaba Cloud roles found for target aliyun-cli")
		}
		root["profiles"] = entries
		// Preserve existing default. First-time Alibaba CLI keeps its established behavior.
		if cur, _ := root["current"].(string); cur == "" {
			for _, p := range profiles {
				if strings.EqualFold(p.VendorType, vendorAlibaba) {
					root["current"] = p.Name
					break
				}
			}
		}
		return json.MarshalIndent(root, "", "  ")
	})
	if err == nil {
		utils.Stdout.Printf("      wrote %d aliyun-cli profile(s) to %s\n", count, path)
	}
	return err
}

var awsRegionPattern = regexp.MustCompile(`^[a-z]{2}(-[a-z]+)+-[0-9]+$`)

func awsPartition(arn string) (string, error) {
	parts := strings.SplitN(arn, ":", 6)
	if len(parts) != 6 || parts[0] != "arn" || parts[2] != "iam" || !strings.HasPrefix(parts[5], "role/") || len(parts[5]) <= 5 {
		return "", fmt.Errorf("invalid AWS IAM role ARN")
	}
	switch parts[1] {
	case "aws", "aws-cn", "aws-us-gov":
		return parts[1], nil
	}
	return "", fmt.Errorf("unsupported AWS partition %s", parts[1])
}

func regionForPartition(partition, region string) bool {
	if !awsRegionPattern.MatchString(region) {
		return false
	}
	switch partition {
	case "aws-cn":
		return strings.HasPrefix(region, "cn-")
	case "aws-us-gov":
		return strings.HasPrefix(region, "us-gov-")
	case "aws":
		return !strings.HasPrefix(region, "cn-") && !strings.HasPrefix(region, "us-gov-") && !strings.HasPrefix(region, "us-iso") && !strings.HasPrefix(region, "eu-isoe-")
	}
	return false
}

// replaceIniKeys preserves non-target sections, comments and profile customizations.
func replaceIniKeys(lines []string, values map[string]string) []string {
	seen := map[string]bool{}
	out := make([]string, 0, len(lines)+len(values))
	for _, line := range lines {
		parts := strings.SplitN(strings.TrimSpace(line), "=", 2)
		if len(parts) == 2 {
			key := strings.TrimSpace(parts[0])
			if value, ok := values[key]; ok {
				if !seen[key] {
					out = append(out, key+" = "+value)
					seen[key] = true
				}
				continue
			}
		}
		out = append(out, line)
	}
	for _, key := range []string{"credential_process", "region"} {
		if value, ok := values[key]; ok && !seen[key] {
			out = append(out, key+" = "+value)
		}
	}
	return out
}

func writeAwsCliProfiles(profiles []generatedProfile, filename string) error {
	home, err := os.UserHomeDir()
	if err != nil {
		return err
	}
	path, err := common.AWSConfigFile()
	if err != nil {
		return err
	}
	credentialsPath := os.Getenv("AWS_SHARED_CREDENTIALS_FILE")
	if credentialsPath == "" {
		credentialsPath = filepath.Join(home, ".aws", "credentials")
	}
	credentialsData, err := os.ReadFile(credentialsPath)
	if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("read AWS shared credentials: %w", err)
	}
	credentials, err := ini.LoadSources(ini.LoadOptions{}, credentialsData)
	if err != nil {
		return fmt.Errorf("invalid AWS shared credentials; existing files were not changed: %w", err)
	}
	for _, p := range profiles {
		if strings.EqualFold(p.VendorType, vendorAws) && credentials.HasSection(p.Name) && len(credentials.Section(p.Name).Keys()) > 0 {
			return fmt.Errorf("AWS shared credentials already defines profile %s; choose another --prefix to avoid provider precedence conflicts", p.Name)
		}
	}
	count := 0
	err = common.UpdateConfigFile(path, func(data []byte) ([]byte, error) {
		file, err := ini.LoadSources(ini.LoadOptions{AllowNestedValues: true}, data)
		if err != nil {
			return nil, fmt.Errorf("invalid AWS config; existing file was not changed: %w", err)
		}
		changes := map[string]map[string]string{}
		order := []string{}
		for _, p := range profiles {
			if !strings.EqualFold(p.VendorType, vendorAws) {
				continue
			}
			partition, err := awsPartition(p.RoleARN)
			if err != nil {
				return nil, fmt.Errorf("profile %s: %w", p.Name, err)
			}
			region := map[string]string{"aws": "us-east-1", "aws-cn": "cn-north-1", "aws-us-gov": "us-gov-west-1"}[partition]
			section := "profile " + p.Name
			if file.HasSection(section) {
				old := file.Section(section)
				for _, key := range []string{"aws_access_key_id", "aws_secret_access_key", "sso_session", "sso_start_url", "source_profile", "credential_source", "role_arn", "web_identity_token_file", "login_session"} {
					if old.HasKey(key) {
						return nil, fmt.Errorf("AWS profile %s uses another credential provider; choose another --prefix", p.Name)
					}
				}
				command := old.Key("credential_process").String()
				if !common.CredentialProcessMatches(command, filename, p.Name) {
					return nil, fmt.Errorf("AWS profile %s is not an IDaaS profile; choose another --prefix", p.Name)
				}
				if value := old.Key("region").String(); value != "" {
					region = value
				}
			}
			if p.AWSRegion != "" {
				region = p.AWSRegion
			}
			if !regionForPartition(partition, region) {
				return nil, fmt.Errorf("AWS profile %s: region %s conflicts with partition %s; use --aws-region with a region in that partition", p.Name, region, partition)
			}
			command, err := credentialCommand(filename, p.Name, false)
			if err != nil {
				return nil, err
			}
			changes[section] = map[string]string{"credential_process": command, "region": region}
			order = append(order, section)
			count++
		}
		if count == 0 {
			return nil, fmt.Errorf("no AWS roles found for target aws-cli")
		}
		var out, sectionLines []string
		sectionName := ""
		seen := map[string]bool{}
		flush := func() {
			if values, ok := changes[sectionName]; ok {
				sectionLines = replaceIniKeys(sectionLines, values)
				seen[sectionName] = true
			}
			out = append(out, sectionLines...)
			sectionLines = nil
		}
		if len(data) > 0 {
			for _, line := range strings.Split(strings.TrimSuffix(string(data), "\n"), "\n") {
				trimmed := strings.TrimSpace(line)
				if strings.HasPrefix(trimmed, "[") {
					flush()
					if end := strings.Index(trimmed, "]"); end > 0 {
						sectionName = trimmed[1:end]
					}
				}
				sectionLines = append(sectionLines, line)
			}
			flush()
		}
		for _, name := range order {
			if !seen[name] {
				out = append(out, "")
				out = append(out, replaceIniKeys([]string{"[" + name + "]"}, changes[name])...)
			}
		}
		// Do not create, delete or rewrite [default] implicitly.
		return []byte(strings.Join(out, "\n") + "\n"), nil
	})
	if err == nil {
		utils.Stdout.Printf("      wrote %d named AWS profile(s) to %s; default unchanged\n", count, path)
	}
	return err
}
