package onboard

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/aliyunidaas/alibaba-cloud-idaas/commands/common"
	"github.com/aliyunidaas/alibaba-cloud-idaas/config"
	"github.com/aliyunidaas/alibaba-cloud-idaas/utils"
	"github.com/pkg/errors"
	"github.com/urfave/cli/v2"
)

const (
	listPathFormat = "%s/v2/%s/cloudAccountRoles/_/actions/listAssumableCloudAccountRoles"
	vendorAlibaba  = "alibaba_cloud"
	vendorAws      = "aws"
)

type assumableRoleList struct {
	Entities []assumableRole `json:"entities"`
}

type assumableRole struct {
	CloudAccountId             string `json:"cloudAccountId"`
	CloudAccountRoleId         string `json:"cloudAccountRoleId"`
	CloudAccountRoleName       string `json:"cloudAccountRoleName"`
	CloudAccountRoleExternalId string `json:"cloudAccountRoleExternalId"`
	CloudAccountVendorType     string `json:"cloudAccountVendorType"`
	Status                     string `json:"status"`
}

type generatedProfile struct {
	Name       string
	VendorType string
}

var (
	stringFlagInstance = &cli.StringFlag{
		Name:    "instance",
		Aliases: []string{"i"},
		Usage:   "IDaaS instance domain (optional if already logged in or profiles exist)",
	}
	stringFlagTarget = &cli.StringFlag{
		Name:  "target",
		Usage: "Target CLI tools (comma-separated: aliyun-cli,aws-cli,tencentcloud-cli,mcp,none). Default: all applicable",
	}
	stringFlagPrefix = &cli.StringFlag{
		Name:  "prefix",
		Usage: "Generated profile name prefix",
		Value: "aliyun",
	}
	stringFlagConfig = &cli.StringFlag{
		Name:    "config",
		Aliases: []string{"c"},
		Usage:   "IDaaS config file (default ~/.aliyun/alibaba-cloud-idaas.json)",
	}
	boolFlagVpc = &cli.BoolFlag{
		Name:  "vpc",
		Usage: "Prefer VPC developer-api endpoint when available",
	}
	stringFlagClientId = &cli.StringFlag{
		Name:  "client-id",
		Usage: "Broker client application id (passed to login, default: from profile or iap_cloud_idaas_cli)",
	}
	boolFlagForceNew = &cli.BoolFlag{
		Name:    "force-new",
		Aliases: []string{"N"},
		Usage:   "Force device-code login (passed to login)",
	}
)

func BuildCommand() *cli.Command {
	return &cli.Command{
		Name:  "onboard",
		Usage: "Discover instance, list assumable cloud roles, and generate CLI tool configs",
		Flags: []cli.Flag{
			stringFlagInstance,
			stringFlagTarget,
			stringFlagPrefix,
			stringFlagConfig,
			boolFlagVpc,
			stringFlagClientId,
			boolFlagForceNew,
		},
		Action: func(c *cli.Context) error {
			return runOnboard(&onboardOptions{
				instance:   c.String("instance"),
				target:     c.String("target"),
				prefix:     c.String("prefix"),
				configFile: c.String("config"),
				preferVpc:  c.Bool("vpc"),
				clientId:   c.String("client-id"),
				forceNew:   c.Bool("force-new"),
			})
		},
	}
}

type onboardOptions struct {
	instance   string
	target     string
	prefix     string
	configFile string
	preferVpc  bool
	clientId   string
	forceNew   bool
}

func runOnboard(opts *onboardOptions) error {
	instance := normalizeDomain(opts.instance)
	prefix := opts.prefix
	if prefix == "" {
		prefix = "aliyun"
	}

	// If --instance not provided, try to infer from existing profiles
	if instance == "" {
		configFilename, err := resolveConfigFilename(opts.configFile)
		if err == nil {
			inferred := common.InferFromProfiles(configFilename)
			if inferred != nil && inferred.Instance != "" {
				instance = inferred.Instance
				utils.Stdout.Printf("[1/4] Using instance from existing profile: %s\n", instance)
			}
		}
	}
	if instance == "" {
		return errors.New("instance domain is required. Please run 'alibaba-cloud-idaas login --instance <域名> --client-id <app_id>' first, or specify --instance")
	}

	httpClient := utils.BuildHttpClient()

	// 1. Discover instance
	utils.Stdout.Printf("[1/4] Discovering instance %s ...\n", instance)
	discovery, err := common.FetchInstanceDiscovery(httpClient, instance)
	if err != nil {
		return errors.Wrap(err, "instance discovery failed. Check the domain is correct and the IDaaS instance is accessible")
	}
	if err := common.ValidateDiscovery(discovery); err != nil {
		return err
	}
	pop := common.ResolvePopEndpoint(discovery, opts.preferVpc)
	issuer := discovery.DefaultAuthorizationServerIssuer
	clientId := opts.clientId
	// If client-id not explicitly provided, try from existing profiles first, then discovery
	if clientId == "" {
		configFilename, _ := resolveConfigFilename(opts.configFile)
		if inferred := common.InferFromProfiles(configFilename); inferred != nil && inferred.ClientId != "" {
			clientId = inferred.ClientId
		}
	}
	if clientId == "" {
		clientId = common.DefaultClientId
	}
	if clientId == common.DefaultClientId {
		utils.Stdout.Printf("      Tip: using default client-id '%s'. If login fails with 'not authorized', specify --client-id with your broker app id.\n", clientId)
	}
	utils.Stdout.Printf("      instance_id=%s  issuer=%s  pop=%s\n", discovery.InstanceId, issuer, pop)

	// 2. Login (delegate to common.DoLogin, transparent if cached)
	utils.Stdout.Printf("[2/4] Login (complete SSO/MFA in browser if needed) ...\n")
	_, err = common.DoLogin(issuer, clientId, common.DefaultPamScope, opts.forceNew)
	if err != nil {
		return errors.Wrap(err, "login failed. Run 'alibaba-cloud-idaas login --instance %s --client-id <app_id> --force-new' to retry")
	}

	// 3. List assumable roles (no vendor filter — multi-cloud)
	utils.Stdout.Printf("[3/4] Listing assumable cloud roles ...\n")
	roles, err := listAssumableRoles(httpClient, pop, discovery.InstanceId, issuer, clientId)
	if err != nil {
		return errors.Wrap(err, "list assumable cloud roles failed")
	}
	if len(roles) == 0 {
		return errors.New("no assumable cloud roles found for current user.\n  hint: 请联系管理员在 PS 授权规则中把目标云角色授予你或你所在的组，然后重试 onboard")
	}
	for _, r := range roles {
		utils.Stdout.Printf("      - %s\t%s\t[%s]\n", r.CloudAccountRoleName, r.CloudAccountRoleExternalId, r.CloudAccountVendorType)
	}

	// 4. Generate profiles + CLI configs
	utils.Stdout.Printf("[4/4] Generating profiles ...\n")
	configFilename, err := resolveConfigFilename(opts.configFile)
	if err != nil {
		return err
	}
	generated, firstProfile, err := generateAndSaveProfiles(configFilename, prefix, discovery.InstanceId, pop, issuer, clientId, roles)
	if err != nil {
		return err
	}
	utils.Stdout.Printf("      wrote %d profile(s) to %s\n", len(generated), configFilename)

	// Write CLI tool configs based on --target
	targets := parseTargets(opts.target, roles)
	if len(targets) > 0 && !containsString(targets, "none") {
		if containsString(targets, "aliyun-cli") {
			if err := writeAliyunCliProfiles(generated, configFilename, pop); err != nil {
				utils.Stderr.Fprintf("Warning: write aliyun-cli config failed: %v\n", err)
			}
		}
		if containsString(targets, "aws-cli") {
			if err := writeAwsCliProfiles(generated, configFilename); err != nil {
				utils.Stderr.Fprintf("Warning: write aws-cli config failed: %v\n", err)
			}
		}
	}

	utils.Stdout.Println("")
	utils.Stdout.Printf("Done. Generated %d profile(s). Try:\n", len(generated))
	utils.Stdout.Printf("  aliyun --profile %s sts GetCallerIdentity\n", firstProfile)
	return nil
}

func listAssumableRoles(client *http.Client, pop, instanceId, issuer, clientId string) ([]assumableRole, error) {
	// Get access token from cache (login should have cached it)
	accessToken, err := getAccessToken(issuer, clientId)
	if err != nil {
		return nil, errors.Wrap(err, "get access token from cache")
	}
	listUrl := fmt.Sprintf(listPathFormat, pop, instanceId)
	headers := map[string]string{"Authorization": "Bearer " + accessToken}
	body, err := utils.FetchAsString(client, utils.HttpMethodGet, listUrl, headers)
	if err != nil {
		return nil, err
	}
	var list assumableRoleList
	if err := json.Unmarshal([]byte(body), &list); err != nil {
		return nil, errors.Wrapf(err, "unmarshal list response: %s", body)
	}
	return list.Entities, nil
}

func getAccessToken(issuer, clientId string) (string, error) {
	return common.DoLogin(issuer, clientId, common.DefaultPamScope, false)
}

func generateAndSaveProfiles(configFilename, prefix, instanceId, pop, issuer, clientId string,
	roles []assumableRole) ([]generatedProfile, string, error) {
	cloudConfig, err := config.ReadCloudCredentialConfig(configFilename)
	if err != nil {
		return nil, "", errors.Wrap(err, "read config file failed")
	}
	if cloudConfig == nil {
		cloudConfig = &config.CloudCredentialConfig{Version: config.Version1, Profile: map[string]*config.CloudStsConfig{}}
	}
	if cloudConfig.Profile == nil {
		cloudConfig.Profile = map[string]*config.CloudStsConfig{}
	}

	oidcProvider := &config.OidcTokenProviderConfig{
		TokenType: "access_token",
		OidcTokenProviderDeviceCode: &config.OidcTokenProviderDeviceCodeConfig{
			Issuer:      issuer,
			ClientId:    clientId,
			Scope:       common.DefaultPamScope,
			AutoOpenUrl: true,
			ShowQrCode:  true,
			SmallQrCode: true,
		},
	}

	usedNames := map[string]bool{}
	var generated []generatedProfile
	var firstProfile string
	for _, r := range roles {
		name := uniqueProfileName(usedNames, prefix, r)
		cloudConfig.Profile[name] = &config.CloudStsConfig{
			CloudAccount: &config.CloudAccountTokenConfig{
				InstanceId:                 instanceId,
				DeveloperApiEndpoint:       pop,
				CloudAccountRoleExternalId: r.CloudAccountRoleExternalId,
				AccessTokenProvider:        oidcProvider,
			},
			Comment: fmt.Sprintf("%s (%s) generated by onboard", r.CloudAccountRoleName, r.CloudAccountId),
		}
		generated = append(generated, generatedProfile{Name: name, VendorType: r.CloudAccountVendorType})
		if firstProfile == "" {
			firstProfile = name
		}
	}
	if cloudConfig.CurrentProfile == "" {
		cloudConfig.CurrentProfile = firstProfile
	}
	if err := saveCloudCredentialConfig(configFilename, cloudConfig); err != nil {
		return nil, "", err
	}
	return generated, firstProfile, nil
}

func saveCloudCredentialConfig(configFilename string, cloudConfig *config.CloudCredentialConfig) error {
	if err := os.MkdirAll(filepath.Dir(configFilename), 0700); err != nil {
		return errors.Wrapf(err, "create config dir for %s", configFilename)
	}
	data, err := json.MarshalIndent(cloudConfig, "", "  ")
	if err != nil {
		return errors.Wrap(err, "marshal config failed")
	}
	return os.WriteFile(configFilename, data, 0600)
}

func writeAliyunCliProfiles(profiles []generatedProfile, configFilename, pop string) error {
	self, err := os.Executable()
	if err != nil || self == "" {
		self = "alibaba-cloud-idaas"
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return errors.Wrap(err, "get home dir")
	}
	aliyunConfigFile := filepath.Join(home, ".aliyun", "config.json")
	root := map[string]interface{}{}
	if data, readErr := os.ReadFile(aliyunConfigFile); readErr == nil && len(data) > 0 {
		_ = json.Unmarshal(data, &root)
	}
	kept := []interface{}{}
	if existing, ok := root["profiles"].([]interface{}); ok {
		for _, p := range existing {
			if pm, ok := p.(map[string]interface{}); ok {
				if name, _ := pm["name"].(string); !containsProfileName(profiles, name) {
					kept = append(kept, pm)
				}
			}
		}
	}
	region := parseRegionFromPop(pop)
	if region == "" {
		region = "cn-hangzhou"
	}
	for _, p := range profiles {
		if !strings.EqualFold(p.VendorType, vendorAlibaba) && p.VendorType != "" {
			continue
		}
		processCommand := fmt.Sprintf("%s fetch-token --profile %s", self, p.Name)
		if !isDefaultConfig(configFilename) {
			processCommand += " --config " + configFilename
		}
		kept = append(kept, map[string]interface{}{
			"name": p.Name, "mode": "External", "region_id": region,
			"output_format": "json", "language": "en", "process_command": processCommand,
		})
	}
	root["profiles"] = kept
	if cur, _ := root["current"].(string); cur == "" && len(profiles) > 0 {
		root["current"] = profiles[0].Name
	}
	if err := os.MkdirAll(filepath.Dir(aliyunConfigFile), 0700); err != nil {
		return errors.Wrapf(err, "create aliyun config dir")
	}
	data, err := json.MarshalIndent(root, "", "  ")
	if err != nil {
		return errors.Wrap(err, "marshal aliyun config failed")
	}
	if err := os.WriteFile(aliyunConfigFile, data, 0600); err != nil {
		return errors.Wrapf(err, "write aliyun config %s", aliyunConfigFile)
	}
	utils.Stdout.Printf("      wrote aliyun-cli profiles to %s\n", aliyunConfigFile)
	return nil
}

func writeAwsCliProfiles(profiles []generatedProfile, configFilename string) error {
	self, err := os.Executable()
	if err != nil || self == "" {
		self = "alibaba-cloud-idaas"
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return errors.Wrap(err, "get home dir")
	}
	awsConfigFile := filepath.Join(home, ".aws", "config")
	// Read existing config as lines
	var lines []string
	if data, readErr := os.ReadFile(awsConfigFile); readErr == nil {
		lines = strings.Split(string(data), "\n")
	}
	// Build a set of existing profile section names to avoid duplicates
	existingSections := map[string]bool{}
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "[profile ") {
			existingSections[strings.TrimSuffix(strings.TrimPrefix(line, "[profile "), "]")] = true
		}
	}
	// Append new profiles
	var newLines []string
	for _, p := range profiles {
		if !strings.EqualFold(p.VendorType, vendorAws) {
			continue
		}
		if existingSections[p.Name] {
			continue
		}
		processCommand := fmt.Sprintf("%s fetch-token --profile %s", self, p.Name)
		if !isDefaultConfig(configFilename) {
			processCommand += " --config " + configFilename
		}
		newLines = append(newLines,
			fmt.Sprintf("[profile %s]", p.Name),
			fmt.Sprintf("credential_process = %s", processCommand),
			"region = us-east-1",
			"",
		)
	}
	if len(newLines) == 0 {
		return nil
	}
	if err := os.MkdirAll(filepath.Dir(awsConfigFile), 0700); err != nil {
		return errors.Wrapf(err, "create aws config dir")
	}
	// Append to existing file
	content := strings.Join(lines, "\n")
	if len(content) > 0 && !strings.HasSuffix(content, "\n") {
		content += "\n"
	}
	content += strings.Join(newLines, "\n")
	if err := os.WriteFile(awsConfigFile, []byte(content), 0600); err != nil {
		return errors.Wrapf(err, "write aws config %s", awsConfigFile)
	}
	utils.Stdout.Printf("      wrote aws-cli profiles to %s\n", awsConfigFile)
	return nil
}

func parseTargets(target string, roles []assumableRole) []string {
	if target == "" {
		// Auto: detect vendor types present
		targets := []string{}
		hasAlibaba := false
		hasAws := false
		for _, r := range roles {
			if strings.EqualFold(r.CloudAccountVendorType, vendorAlibaba) {
				hasAlibaba = true
			}
			if strings.EqualFold(r.CloudAccountVendorType, vendorAws) {
				hasAws = true
			}
		}
		if hasAlibaba {
			targets = append(targets, "aliyun-cli")
		}
		if hasAws {
			targets = append(targets, "aws-cli")
		}
		return targets
	}
	return strings.Split(target, ",")
}

func resolveConfigFilename(configFile string) (string, error) {
	if configFile != "" {
		return configFile, nil
	}
	return config.GetDefaultCloudCredentialConfigFile()
}

func isDefaultConfig(configFilename string) bool {
	def, err := config.GetDefaultCloudCredentialConfigFile()
	if err != nil {
		return false
	}
	return configFilename == def
}

func uniqueProfileName(used map[string]bool, prefix string, r assumableRole) string {
	base := prefix + "-" + sanitizeName(r.CloudAccountRoleName)
	name := base
	i := 2
	for used[name] {
		name = fmt.Sprintf("%s-%d", base, i)
		i++
	}
	used[name] = true
	return name
}

func sanitizeName(s string) string {
	s = strings.ToLower(strings.TrimSpace(s))
	var sb strings.Builder
	prevDash := false
	for _, ch := range s {
		if (ch >= 'a' && ch <= 'z') || (ch >= '0' && ch <= '9') {
			sb.WriteRune(ch)
			prevDash = false
		} else if !prevDash {
			sb.WriteByte('-')
			prevDash = true
		}
	}
	out := strings.Trim(sb.String(), "-")
	if out == "" {
		return "role"
	}
	return out
}

func parseRegionFromPop(pop string) string {
	host := pop
	host = strings.TrimPrefix(host, "https://")
	host = strings.TrimPrefix(host, "http://")
	if idx := strings.Index(host, "/"); idx >= 0 {
		host = host[:idx]
	}
	const marker = "eiam-developerapi."
	i := strings.Index(host, marker)
	if i < 0 {
		return ""
	}
	rest := host[i+len(marker):]
	if j := strings.Index(rest, "."); j >= 0 {
		return rest[:j]
	}
	return ""
}

func normalizeDomain(domain string) string {
	domain = strings.TrimSpace(domain)
	domain = strings.TrimPrefix(domain, "https://")
	domain = strings.TrimPrefix(domain, "http://")
	domain = strings.TrimRight(domain, "/")
	return domain
}

func containsString(list []string, s string) bool {
	for _, v := range list {
		if v == s {
			return true
		}
	}
	return false
}

func containsProfileName(profiles []generatedProfile, name string) bool {
	for _, p := range profiles {
		if p.Name == name {
			return true
		}
	}
	return false
}
