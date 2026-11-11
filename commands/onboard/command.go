package onboard

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/aliyunidaas/alibaba-cloud-idaas/cloud"
	"github.com/aliyunidaas/alibaba-cloud-idaas/cloud/cloud_account"
	"github.com/aliyunidaas/alibaba-cloud-idaas/commands/common"
	"github.com/aliyunidaas/alibaba-cloud-idaas/config"
	"github.com/aliyunidaas/alibaba-cloud-idaas/utils"
	"github.com/pkg/errors"
	"github.com/urfave/cli/v2"
)

const (
	vendorAlibaba = "alibaba_cloud"
	vendorAws     = "aws"
	// TODO(contract): vendor value "tencent_cloud" is an assumption pending
	// backend alignment on cloudAccountVendorType for TencentCloud roles.
	vendorTencent = "tencent_cloud"
)

// vendorPrefix returns the default profile name prefix for a cloud vendor.
func vendorPrefix(vendorType string) string {
	switch {
	case strings.EqualFold(vendorType, vendorAlibaba):
		return "aliyun"
	case strings.EqualFold(vendorType, vendorAws):
		return "aws"
	case strings.EqualFold(vendorType, vendorTencent):
		return "tencentcloud"
	default:
		return "cloud"
	}
}

type assumableRole = common.AssumableRole

type generatedProfile struct {
	Name       string
	VendorType string
	RoleARN    string
	AWSRegion  string
}

var (
	stringFlagInstance = &cli.StringFlag{
		Name:    "instance",
		Aliases: []string{"i"},
		Usage:   "IDaaS instance domain (optional if already logged in or profiles exist)",
	}
	stringFlagTarget = &cli.StringFlag{
		Name:  "target",
		Usage: "Target CLI tools (comma-separated: aliyun-cli,aws-cli,tencentcloud-cli,none). Default: all applicable",
	}
	stringFlagPrefix = &cli.StringFlag{
		Name:  "prefix",
		Usage: "Profile name prefix (default: vendor-based, e.g. aliyun/aws/tencentcloud)",
	}
	stringFlagConfig = &cli.StringFlag{
		Name:    "config",
		Aliases: []string{"c"},
		Usage:   "IDaaS config file (env override, otherwise existing legacy ~/.aliyun/alibaba-cloud-idaas.json or ~/.cloud_idaas/idaas-cli.json)",
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
			&cli.StringFlag{Name: "result-file", Usage: "Write a credential-free versioned JSON result to a new file; progress remains on the terminal"},
			&cli.StringFlag{Name: "aws-region", Usage: "AWS region override (must match the role ARN partition); defaults by partition"},
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
				awsRegion:  c.String("aws-region"),
				resultFile: c.String("result-file"),
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
	awsRegion  string
	resultFile string
}

func runOnboard(opts *onboardOptions) (runErr error) {
	configFilename, err := resolveConfigFilename(opts.configFile)
	if err != nil {
		return err
	}
	resultFile, err := prepareResultFile(opts.resultFile, configFilename)
	if err != nil {
		return err
	}
	result := &onboardResult{SchemaVersion: 1, Status: "failed", Stage: "resolve_context", ConfigFile: configFilename, Targets: []targetResult{}}
	defer func() {
		if resultFile == "" {
			return
		}
		if runErr == nil {
			result.Status, result.Stage = "success", "complete"
		} else {
			result.FailureCode = result.Stage + "_failed"
			if result.BrokerConfigSaved {
				result.Status = "partial_success"
			}
		}
		if err := writeResultFile(resultFile, result); err != nil {
			if runErr == nil {
				runErr = fmt.Errorf("configuration completed but result file could not be written: %w", err)
			} else {
				runErr = fmt.Errorf("%v; result file could not be written: %w", runErr, err)
			}
		}
	}()
	context, err := common.ResolveRegistrationContext(configFilename, opts.instance, opts.clientId)
	if err != nil {
		return err
	}
	instance := context.Instance
	result.Stage = "validate_options"
	prefix := opts.prefix
	if prefix != "" {
		for _, r := range prefix {
			if !((r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '-' || r == '_' || r == '.') {
				return fmt.Errorf("prefix must contain only letters, digits, dot, hyphen or underscore")
			}
		}
	}
	// Validate target spelling before device authorization or any configuration write.
	if _, err := validatedTargets(opts.target, nil, false); err != nil {
		return err
	}
	httpClient := utils.BuildHttpClient()
	result.Stage = "discover_instance"

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
	clientId := context.ClientID
	utils.Stdout.Printf("      client_id=%s source=%s\n", clientId, context.ClientIDSource)
	utils.Stdout.Printf("      instance_id=%s  issuer=%s  pop=%s\n", discovery.InstanceId, issuer, pop)

	// 2. Login (delegate to common.DoLogin, transparent if cached)
	result.Stage = "login"
	utils.Stdout.Printf("[2/4] Login (complete SSO/MFA in browser if needed) ...\n")
	_, err = common.DoLogin(issuer, clientId, common.DefaultPamScope, opts.forceNew)
	if err != nil {
		return errors.Wrapf(err, "login failed; retry login --instance %s --client-id %s --force-new after checking the error", instance, clientId)
	}

	// 3. List assumable roles (no vendor filter — multi-cloud)
	result.Stage = "list_roles"
	utils.Stdout.Printf("[3/4] Listing assumable cloud roles ...\n")
	roles, err := listAssumableRoles(httpClient, pop, discovery.InstanceId, issuer, clientId)
	if err != nil {
		return errors.Wrap(err, "list assumable cloud roles failed")
	}
	if len(roles) == 0 {
		return errors.New("no assumable cloud roles found for current user.\n  hint: check direct/group role grants and permission template grants, enabled template status and matching usage type")
	}
	for _, r := range roles {
		utils.Stdout.Printf("      - %s\t%s\t[%s]\ttemplate-created=%s\n", r.CloudAccountRoleName, r.CloudAccountRoleExternalId, r.CloudAccountVendorType, common.TemplateCreationLabel(r.CreatedByPermissionTemplate))
	}

	result.Stage = "validate_roles"
	targets, err := validatedTargets(opts.target, roles, true)
	if err != nil {
		return err
	}
	if opts.awsRegion != "" {
		for _, r := range roles {
			if strings.EqualFold(r.CloudAccountVendorType, vendorAws) {
				partition, e := awsPartition(r.CloudAccountRoleExternalId)
				if e != nil {
					return e
				}
				if !regionForPartition(partition, opts.awsRegion) {
					return fmt.Errorf("AWS region %s conflicts with partition %s", opts.awsRegion, partition)
				}
			}
		}
	}
	utils.Stdout.Printf("[4/4] Generating profiles ...\n")
	result.Stage = "generate_profiles"
	generated, _, err := generateAndSaveProfiles(configFilename, prefix, discovery.InstanceId, pop, issuer, clientId, roles)
	if err != nil {
		return err
	}
	for i := range generated {
		generated[i].AWSRegion = opts.awsRegion
	}
	utils.Stdout.Printf("      wrote %d profile(s) to %s\n", len(generated), configFilename)
	result.BrokerConfigSaved, result.ProfileCount = true, len(generated)
	result.Stage = "write_targets"
	failures := []string{}
	for _, target := range targets {
		var e error
		item := targetResult{Target: target, Status: "success", Files: targetFiles(target, generated)}
		switch target {
		case "none":
			result.Targets = append(result.Targets, item)
			continue
		case "aliyun-cli":
			e = writeAliyunCliProfiles(generated, configFilename, pop)
		case "aws-cli":
			e = writeAwsCliProfiles(generated, configFilename)
		case "tencentcloud-cli":
			e = writeTencentcloudCliProfiles(generated, configFilename)
		}
		if e != nil {
			item.Status, item.ErrorCode = "failed", "configuration_write_failed"
			result.Targets = append(result.Targets, item)
			failures = append(failures, target+": "+e.Error())
			utils.Stderr.Printf("FAILED %s: %v\n", target, e)
			continue
		}
		result.Targets = append(result.Targets, item)
		utils.Stdout.Printf("SUCCESS %s\n", target)
		for _, p := range generated {
			if target == "aliyun-cli" && strings.EqualFold(p.VendorType, vendorAlibaba) {
				utils.Stdout.Printf("  aliyun --profile %s sts GetCallerIdentity\n", p.Name)
				break
			}
			if target == "aws-cli" && strings.EqualFold(p.VendorType, vendorAws) {
				utils.Stdout.Printf("  aws --profile %s sts get-caller-identity\n", p.Name)
				break
			}
			if target == "tencentcloud-cli" && strings.EqualFold(p.VendorType, vendorTencent) {
				utils.Stdout.Printf("  tccli --profile %s sts GetCallerIdentity\n", p.Name)
				break
			}
		}
	}
	if len(failures) > 0 {
		return fmt.Errorf("onboard partially completed: Broker profiles are saved; failed targets: %s. Fix the reported targets and retry; existing defaults were preserved", strings.Join(failures, "; "))
	}
	utils.Stdout.Printf("Done. Generated %d profile(s). Existing defaults were preserved; AWS profiles are named only.\n", len(generated))
	return nil
}

func listAssumableRoles(client *http.Client, pop, instanceId, issuer, clientId string) ([]assumableRole, error) {
	// Get access token from cache (login should have cached it)
	accessToken, err := getAccessToken(issuer, clientId)
	if err != nil {
		return nil, errors.Wrap(err, "get access token from cache")
	}
	list, err := common.DiscoverAssumableRoles(client, pop, instanceId, accessToken)
	if err != nil {
		return nil, err
	}
	return list.Entities, nil
}

func getAccessToken(issuer, clientId string) (string, error) {
	return common.DoLogin(issuer, clientId, common.DefaultPamScope, false)
}

func generateAndSaveProfiles(configFilename, prefix, instanceId, pop, issuer, clientId string,
	roles []assumableRole) ([]generatedProfile, string, error) {
	var generated []generatedProfile
	var firstProfile string
	err := common.UpdateConfigFile(configFilename, func(original []byte) ([]byte, error) {
		cloudConfig, err := config.ReadCloudCredentialConfig(configFilename)
		if err != nil {
			return nil, errors.Wrap(err, "read config file failed")
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
		for name := range cloudConfig.Profile {
			usedNames[name] = true
		}
		sort.Slice(roles, func(i, j int) bool { return roles[i].CloudAccountRoleExternalId < roles[j].CloudAccountRoleExternalId })
		for _, r := range roles {
			// Use --prefix if specified, otherwise derive from vendor type
			effectivePrefix := prefix
			if effectivePrefix == "" {
				effectivePrefix = vendorPrefix(r.CloudAccountVendorType)
			}
			name := ""
			existing := []string{}
			for candidate, sts := range cloudConfig.Profile {
				if !strings.HasPrefix(candidate, effectivePrefix+"-") || sts == nil || sts.CloudAccount == nil {
					continue
				}
				account := sts.CloudAccount
				if account.InstanceId == instanceId && account.CloudAccountRoleExternalId == r.CloudAccountRoleExternalId && account.AccessTokenProvider != nil && account.AccessTokenProvider.OidcTokenProviderDeviceCode != nil {
					dc := account.AccessTokenProvider.OidcTokenProviderDeviceCode
					if dc.Issuer == issuer && dc.ClientId == clientId {
						existing = append(existing, candidate)
					}
				}
			}
			sort.Strings(existing)
			if len(existing) > 0 {
				name = existing[0]
			} else {
				name = uniqueProfileName(usedNames, effectivePrefix, r)
			}
			if len(existing) == 0 {
				cloudConfig.Profile[name] = &config.CloudStsConfig{
					CloudAccount: &config.CloudAccountTokenConfig{
						InstanceId:                 instanceId,
						DeveloperApiEndpoint:       pop,
						CloudAccountRoleExternalId: r.CloudAccountRoleExternalId,
						AccessTokenProvider:        oidcProvider,
					},
					Comment: fmt.Sprintf("%s (%s) generated by onboard", r.CloudAccountRoleName, r.CloudAccountId),
				}
			}
			generated = append(generated, generatedProfile{Name: name, VendorType: r.CloudAccountVendorType, RoleARN: r.CloudAccountRoleExternalId})
			if firstProfile == "" {
				firstProfile = name
			}
		}
		if cloudConfig.CurrentProfile == "" {
			cloudConfig.CurrentProfile = firstProfile
		}
		encoded, err := json.Marshal(cloudConfig)
		if err != nil {
			return nil, err
		}
		var oldRoot, newRoot map[string]interface{}
		if len(original) > 0 {
			oldRoot, err = common.DecodeConfigObject(original)
			if err != nil {
				return nil, err
			}
		}
		if err = json.Unmarshal(encoded, &newRoot); err != nil {
			return nil, err
		}
		if oldRoot == nil {
			oldRoot = map[string]interface{}{}
		}
		// Keep all unrelated and unknown profile/root fields exactly as JSON values.
		oldProfiles, _ := oldRoot["profile"].(map[string]interface{})
		if oldProfiles == nil {
			oldProfiles = map[string]interface{}{}
		}
		newProfiles := newRoot["profile"].(map[string]interface{})
		for _, p := range generated {
			if old, exists := oldProfiles[p.Name]; !exists {
				oldProfiles[p.Name] = newProfiles[p.Name]
			} else {
				// Reusing an identity must still honor this invocation's public/VPC
				// endpoint choice, without discarding scopes or unknown settings.
				profile, ok := old.(map[string]interface{})
				if !ok {
					return nil, fmt.Errorf("invalid existing profile %s", p.Name)
				}
				account, ok := profile["cloud_account_token"].(map[string]interface{})
				if !ok {
					return nil, fmt.Errorf("invalid existing cloud account profile %s", p.Name)
				}
				account["developer_api_endpoint"] = pop
			}
		}
		oldRoot["profile"] = oldProfiles
		oldRoot["version"] = newRoot["version"]
		oldRoot["current_profile"] = newRoot["current_profile"]
		return json.MarshalIndent(oldRoot, "", "  ")
	})
	return generated, firstProfile, err
}

func saveCloudCredentialConfig(configFilename string, cloudConfig *config.CloudCredentialConfig) error {
	if err := os.MkdirAll(filepath.Dir(configFilename), 0700); err != nil {
		return errors.Wrapf(err, "create config dir for %s", configFilename)
	}
	data, err := json.MarshalIndent(cloudConfig, "", "  ")
	if err != nil {
		return errors.Wrap(err, "marshal config failed")
	}
	return common.UpdateConfigFile(configFilename, func(_ []byte) ([]byte, error) { return data, nil })
}

// writeTencentcloudCliProfiles writes ~/.tccli/<profile>.credential for each TencentCloud role.
// tccli has no external credential process (unlike aliyun-cli/aws-cli), so
// the temporary credential is written statically and must be refreshed via
// fetch-token --output when it expires.
func writeTencentcloudCliProfiles(profiles []generatedProfile, configFilename string) error {
	home, err := os.UserHomeDir()
	if err != nil {
		return errors.Wrap(err, "get home dir")
	}
	tccliDir := filepath.Join(home, ".tccli")
	if err := os.MkdirAll(tccliDir, 0700); err != nil {
		return errors.Wrapf(err, "create tccli dir %s", tccliDir)
	}
	for _, p := range profiles {
		if !strings.EqualFold(p.VendorType, vendorTencent) {
			continue
		}
		credentialFile := filepath.Join(tccliDir, p.Name+".credential")
		if err := checkNewTencentCredentialPath(credentialFile); err != nil {
			return err
		}
		// Fetch the credential via the same path as fetch-token (reuses the
		// cloud_token cache and the already-cached OIDC access token).
		sts, _, err := cloud.FetchCloudStsFromDefaultConfig(configFilename, p.Name, &cloud.FetchCloudStsOptions{})
		if err != nil {
			return errors.Wrapf(err, "fetch TencentCloud credential for profile %s", p.Name)
		}
		cloudAccountToken, ok := sts.(*cloud_account.CloudAccountToken)
		if !ok || !cloudAccountToken.IsTencentCloudToken() {
			return fmt.Errorf("profile %s did not return a TencentCloud credential", p.Name)
		}
		credentialJson, err := cloud.MarshalTencentCloudCliCredential(
			cloudAccountToken.CloudAccountRoleAccessCredential.TencentCloudStsToken)
		if err != nil {
			return errors.Wrapf(err, "marshal tccli credential for profile %s", p.Name)
		}
		// tccli stores static credentials and has no Broker identity reference.
		// An existing file cannot be safely attributed to this onboarding run.
		if err := common.UpdateConfigFile(credentialFile, func(_ []byte) ([]byte, error) {
			if err := checkNewTencentCredentialPath(credentialFile); err != nil {
				return nil, err
			}
			return []byte(credentialJson), nil
		}); err != nil {
			return errors.Wrapf(err, "write %s", credentialFile)
		}
		utils.Stdout.Printf("      wrote tccli profile %s to %s\n", p.Name, tccliDir)
	}
	return nil
}

func checkNewTencentCredentialPath(path string) error {
	if _, err := os.Lstat(path); err == nil {
		return fmt.Errorf("tccli credential file already exists (%s); choose another --prefix or explicitly back up and remove that file before retrying; no existing identity was overwritten", path)
	} else if !os.IsNotExist(err) {
		return err
	}
	return nil
}

func parseTargets(target string, roles []assumableRole) []string {
	if target == "" {
		// Auto: detect vendor types present
		targets := []string{}
		hasAlibaba := false
		hasAws := false
		hasTencent := false
		for _, r := range roles {
			if strings.EqualFold(r.CloudAccountVendorType, vendorAlibaba) {
				hasAlibaba = true
			}
			if strings.EqualFold(r.CloudAccountVendorType, vendorAws) {
				hasAws = true
			}
			if strings.EqualFold(r.CloudAccountVendorType, vendorTencent) {
				hasTencent = true
			}
		}
		if hasAlibaba {
			targets = append(targets, "aliyun-cli")
		}
		if hasAws {
			targets = append(targets, "aws-cli")
		}
		if hasTencent {
			targets = append(targets, "tencentcloud-cli")
		}
		return targets
	}
	return strings.Split(target, ",")
}

func resolveConfigFilename(configFile string) (string, error) {
	if configFile != "" {
		return filepath.Abs(configFile)
	}
	file, err := config.GetDefaultCloudCredentialConfigFile()
	if err != nil {
		return "", err
	}
	return filepath.Abs(file)
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
