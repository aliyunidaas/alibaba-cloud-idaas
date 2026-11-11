package show

import (
	"encoding/json"
	"fmt"
	"github.com/aliyunidaas/alibaba-cloud-idaas/commands/show_signer_public_key"
	"gopkg.in/ini.v1"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/aliyunidaas/alibaba-cloud-idaas/commands/common"
	"github.com/aliyunidaas/alibaba-cloud-idaas/config"
	"github.com/aliyunidaas/alibaba-cloud-idaas/constants"
	"github.com/aliyunidaas/alibaba-cloud-idaas/utils"
	"github.com/urfave/cli/v2"
)

func BuildCommand() *cli.Command {
	return &cli.Command{
		Name:  "show",
		Usage: "Query and display information (profiles, roles, credentials, cache, token, status, instance)",
		Subcommands: []*cli.Command{
			buildShowProfiles(),
			buildShowRoles(),
			buildShowCache(),
			buildShowToken(),
			buildShowStatus(),
			buildShowInstance(),
			buildShowSignerKey(),
		},
	}
}

// --- show profiles ---

func buildShowProfiles() *cli.Command {
	return &cli.Command{
		Name:    "profiles",
		Aliases: []string{"profile", "p"},
		Usage:   "List configured profiles",
		Flags: []cli.Flag{
			&cli.StringFlag{Name: "config", Aliases: []string{"c"}, Usage: "IDaaS config file"},
			&cli.StringFlag{Name: "profile-filter", Aliases: []string{"f"}, Usage: "Profile name filter"},
			&cli.BoolFlag{Name: "no-color", Usage: "Output without color"},
		},
		Action: func(c *cli.Context) error {
			return showProfiles(c.String("config"), c.String("profile-filter"), !c.Bool("no-color"))
		},
	}
}

func showProfiles(configFile, filter string, color bool) error {
	if configFile == "" {
		var err error
		configFile, err = config.GetDefaultCloudCredentialConfigFile()
		if err != nil {
			return err
		}
	}
	cloudConfig, err := config.ReadCloudCredentialConfig(configFile)
	if err != nil {
		return err
	}
	if cloudConfig == nil {
		utils.Stdout.Println("No config found.")
		return nil
	}
	utils.Stdout.Printf("Config: %s\n", configFile)
	utils.Stdout.Printf("Current: %s\n\n", cloudConfig.CurrentProfile)
	names := []string{}
	for name := range cloudConfig.Profile {
		names = append(names, name)
	}
	sort.Strings(names)
	for _, name := range names {
		sts := cloudConfig.Profile[name]
		if sts == nil {
			continue
		}
		if filter != "" && !contains(name, filter) {
			continue
		}
		provider := "unknown"
		if sts.CloudAccount != nil {
			provider = "cloud_account_token"
		} else if sts.AlibabaCloud != nil {
			provider = "alibaba_cloud_sts"
		} else if sts.Aws != nil {
			provider = "aws_sts"
		} else if sts.OidcToken != nil {
			provider = "oidc_token"
		} else if sts.Credential != nil {
			provider = "credential"
		} else if sts.Agent != nil {
			provider = "agent"
		}
		marker := "  "
		if name == cloudConfig.CurrentProfile {
			marker = "* "
		}
		utils.Stdout.Printf("%s%s\t[%s]\n", marker, name, provider)
	}
	return nil
}

// --- show roles ---

func buildShowRoles() *cli.Command {
	return &cli.Command{
		Name:    "roles",
		Aliases: []string{"role", "r"},
		Usage:   "List assumable cloud roles for current user",
		Flags: []cli.Flag{
			&cli.StringFlag{Name: "instance", Aliases: []string{"i"}, Usage: "IDaaS instance domain (optional if profiles exist)"},
			&cli.StringFlag{Name: "scope", Aliases: []string{"s"}, Usage: "Scope (default: urn:cloud:idaas:pam|.all)", Value: common.DefaultPamScope},
			&cli.StringFlag{Name: "client-id", Usage: "Broker client id (default: from profile or iap_cloud_idaas_cli)"},
			&cli.StringFlag{Name: "config", Aliases: []string{"c"}, Usage: "IDaaS config file"},
			&cli.BoolFlag{Name: "vpc", Usage: "Prefer VPC endpoint"},
			&cli.BoolFlag{Name: "json", Usage: "Machine-readable JSON output"},
		},
		Action: func(c *cli.Context) error {
			return showRoles(c.String("instance"), c.String("scope"), c.String("client-id"), c.String("config"), c.Bool("vpc"), c.Bool("json"))
		},
	}
}

func showRoles(instance, scope, clientId, configFile string, preferVpc, jsonOutput bool) error {
	context, err := common.ResolveRegistrationContext(configFile, instance, clientId)
	if err != nil {
		return err
	}
	instance = context.Instance
	clientId = context.ClientID
	httpClient := utils.BuildHttpClient()

	discovery, err := common.FetchInstanceDiscovery(httpClient, instance)
	if err != nil {
		return fmt.Errorf("discovery failed: %w", err)
	}
	if err := common.ValidateDiscovery(discovery); err != nil {
		return err
	}
	pop := common.ResolvePopEndpoint(discovery, preferVpc)
	issuer := discovery.DefaultAuthorizationServerIssuer
	if clientId == "" {
		clientId = common.DefaultClientId
	}

	// Login (use cached token if available)
	accessToken, err := common.DoLogin(issuer, clientId, scope, false)
	if err != nil {
		return fmt.Errorf("login failed: %w", err)
	}

	// List roles
	list, err := common.DiscoverAssumableRoles(httpClient, pop, discovery.InstanceId, accessToken)
	if err != nil {
		return fmt.Errorf("list roles failed: %w", err)
	}
	if jsonOutput {
		body, err := json.Marshal(list)
		if err != nil {
			return fmt.Errorf("marshal roles: %w", err)
		}
		utils.Stdout.Println(string(body))
		return nil
	}

	if len(list.Entities) == 0 {
		utils.Stdout.Println("No assumable cloud roles found. Check direct/group role grants, permission template grants, enabled template status and matching usage type.")
		return nil
	}
	utils.Stdout.Printf("Found %d role(s):\n\n", len(list.Entities))
	utils.Stdout.Printf("  %-30s %-55s %-15s %-10s %s\n", "NAME", "EXTERNAL ID", "VENDOR", "STATUS", "TEMPLATE-CREATED")
	for _, r := range list.Entities {
		utils.Stdout.Printf("  %-30s %-55s %-15s %-10s %s\n",
			r.CloudAccountRoleName, r.CloudAccountRoleExternalId, r.CloudAccountVendorType, r.Status, common.TemplateCreationLabel(r.CreatedByPermissionTemplate))
	}
	return nil
}

// --- show cache ---

func buildShowCache() *cli.Command {
	return &cli.Command{
		Name:    "cache",
		Aliases: []string{"c"},
		Usage:   "Show cache entries",
		Flags: []cli.Flag{
			&cli.StringFlag{Name: "category", Aliases: []string{"C"}, Usage: "Category filter"},
			&cli.StringFlag{Name: "name", Aliases: []string{"n"}, Usage: "Name filter"},
		},
		Action: func(c *cli.Context) error {
			return listCacheMetadata(c.String("category"), c.String("name"))
		},
	}
}

// --- show token ---

func buildShowToken() *cli.Command {
	return &cli.Command{
		Name:    "token",
		Aliases: []string{"t"},
		Usage:   "Credential display requires the explicit legacy show-token command",
		Flags: []cli.Flag{
			&cli.StringFlag{Name: "profile", Aliases: []string{"p"}, Usage: "Profile"},
			&cli.StringFlag{Name: "config", Aliases: []string{"c"}, Usage: "Config file"},
			&cli.StringFlag{Name: "oidc-field", Usage: "OIDC field (id_token or access_token)"},
			&cli.BoolFlag{Name: "no-color", Usage: "No color"},
			&cli.BoolFlag{Name: "force-new", Aliases: []string{"N"}, Usage: "Force refresh"},
		},
		Action: func(c *cli.Context) error {
			return fmt.Errorf("show token does not expose credentials implicitly; use the explicit legacy show-token command only if you intend to display secret credentials")
		},
	}
}

// --- show status ---

func buildShowStatus() *cli.Command {
	return &cli.Command{
		Name:    "status",
		Aliases: []string{"s"},
		Usage:   "Show current profile, login state, and serve daemon status",
		Flags: []cli.Flag{
			&cli.StringFlag{Name: "profile", Aliases: []string{"p"}, Usage: "Profile"},
			&cli.StringFlag{Name: "config", Aliases: []string{"c"}, Usage: "Config file"},
			&cli.BoolFlag{Name: "json", Usage: "JSON output"},
		},
		Action: func(c *cli.Context) error {
			return showStatus(c.String("profile"), c.String("config"), c.Bool("json"))
		},
	}
}

func showStatus(flagProfile, configFile string, jsonOutput bool) error {
	profile := common.ResolveProfile(flagProfile)
	configFile, err := common.ResolveConfigFilename(configFile)
	if err != nil {
		return err
	}
	cloudConfig, err := config.ReadCloudCredentialConfig(configFile)
	if err != nil {
		return err
	}
	if cloudConfig == nil {
		utils.Stdout.Println("No config found. Run 'alibaba-cloud-idaas onboard --instance <domain>' first.")
		return nil
	}
	if profile == "" {
		profile = cloudConfig.CurrentProfile
	}
	if profile == "" {
		utils.Stdout.Println("No current profile set.")
		return nil
	}
	stsConfig, ok := cloudConfig.Profile[profile]
	if !ok || stsConfig == nil {
		return fmt.Errorf("profile %q not found or invalid", profile)
	}
	providerType := "unknown"
	instanceId := ""
	if stsConfig.CloudAccount != nil {
		providerType = "cloud_account_token (PAM)"
		instanceId = stsConfig.CloudAccount.GetInstanceId()
	} else if stsConfig.AlibabaCloud != nil {
		providerType = "alibaba_cloud_sts (keyless)"
	} else if stsConfig.Aws != nil {
		providerType = "aws_sts (keyless)"
	} else if stsConfig.OidcToken != nil {
		providerType = "oidc_token"
	} else if stsConfig.Credential != nil {
		providerType = "credential (static)"
	} else if stsConfig.Agent != nil {
		providerType = "agent"
	}
	serveRunning := checkServe()
	if jsonOutput {
		value := map[string]interface{}{"profile": profile, "provider": providerType, "instance_id": instanceId, "serve_running": serveRunning}
		for k, v := range statusContext(cloudConfig, stsConfig, profile, configFile) {
			value[k] = v
		}
		encoded, err := json.Marshal(value)
		if err != nil {
			return err
		}
		utils.Stdout.Println(string(encoded))
	} else {
		utils.Stdout.Printf("Profile:   %s\n", profile)
		utils.Stdout.Printf("Provider:  %s\n", providerType)
		if instanceId != "" {
			utils.Stdout.Printf("Instance:  %s\n", instanceId)
		}
		utils.Stdout.Printf("Serve:     %s\n", boolStr(serveRunning, "running", "not running"))
		utils.Stdout.Printf("\nProfiles:  %d total\n", len(cloudConfig.Profile))
		extra := statusContext(cloudConfig, stsConfig, profile, configFile)
		for _, key := range []string{"config_file", "config_state", "client_id", "client_id_source", "client_context", "issuer", "broker_current_profile", "aliyun_current_profile", "defaults_differ", "aws_default_state", "aws_default_idaas_profile", "aws_environment_profile", "cache_readability", "login_state", "cloud_api_verification"} {
			utils.Stdout.Printf("%s: %v\n", key, extra[key])
		}
	}
	return nil
}

// --- show instance ---

func buildShowInstance() *cli.Command {
	return &cli.Command{
		Name:    "instance",
		Aliases: []string{"i"},
		Usage:   "Show instance discovery information",
		Flags: []cli.Flag{
			&cli.StringFlag{Name: "instance", Aliases: []string{"d"}, Usage: "IDaaS instance domain", Required: true},
			&cli.BoolFlag{Name: "vpc", Usage: "Prefer VPC endpoint"},
			&cli.StringFlag{Name: "config", Aliases: []string{"c"}, Usage: "Broker config for client-id resolution"},
			&cli.StringFlag{Name: "client-id", Usage: "Optional override; default: matching profile or iap_cloud_idaas_cli"},
		},
		Action: func(c *cli.Context) error {
			filename, err := common.ResolveConfigFilename(c.String("config"))
			if err != nil {
				return err
			}
			context, err := common.ResolveRegistrationContext(filename, c.String("instance"), c.String("client-id"))
			if err != nil {
				return err
			}
			utils.Stdout.Printf("Config file:     %s\nClient ID:       %s\nClient source:   %s\n", filename, context.ClientID, context.ClientIDSource)
			return showInstance(context.Instance, c.Bool("vpc"))
		},
	}
}

func showInstance(instance string, preferVpc bool) error {
	instance = normalizeDomain(instance)
	httpClient := utils.BuildHttpClient()
	discovery, err := common.FetchInstanceDiscovery(httpClient, instance)
	if err != nil {
		return fmt.Errorf("discovery failed: %w", err)
	}
	utils.Stdout.Printf("Instance ID:      %s\n", discovery.InstanceId)
	utils.Stdout.Printf("Issuer:           %s\n", discovery.DefaultAuthorizationServerIssuer)
	utils.Stdout.Printf("POP (internet):   %s\n", discovery.DeveloperApiEndpoint.Internet)
	if discovery.DeveloperApiEndpoint.Vpc != "" {
		utils.Stdout.Printf("POP (vpc):        %s\n", discovery.DeveloperApiEndpoint.Vpc)
	}
	return nil
}

// --- show signer-key ---

func buildShowSignerKey() *cli.Command {
	command := show_signer_public_key.BuildCommand()
	command.Name = "signer-key"
	command.Aliases = []string{"sk"}
	return command
}

func statusContext(cfg *config.CloudCredentialConfig, sts *config.CloudStsConfig, profile, filename string) map[string]interface{} {
	value := map[string]interface{}{"config_file": filename, "config_state": "loaded", "broker_current_profile": cfg.CurrentProfile, "aliyun_current_profile": "", "defaults_differ": false, "cache_readability": "not_checked", "login_state": "not_checked", "cloud_api_verification": "not_checked", "client_id": "", "client_id_source": "unknown", "client_context": "not_applicable", "issuer": ""}
	if context, err := common.ResolveProfileLoginContext(sts, profile, "", ""); err == nil {
		value["client_id"] = context.ClientID
		value["client_id_source"] = context.ClientIDSource
		value["issuer"] = context.Issuer
		value["client_context"] = "profile_login"
	}
	home, err := os.UserHomeDir()
	if err == nil {
		data, e := os.ReadFile(filepath.Join(home, ".aliyun", "config.json"))
		if e == nil {
			var root struct {
				Current string `json:"current"`
			}
			if e = json.Unmarshal(data, &root); e == nil {
				value["aliyun_current_profile"] = root.Current
				value["defaults_differ"] = root.Current != "" && cfg.CurrentProfile != "" && root.Current != cfg.CurrentProfile
			} else {
				value["aliyun_config_state"] = "invalid"
			}
		}
	}
	value["aws_default_state"] = "not_found"
	if awsPath, e := common.AWSConfigFile(); e == nil {
		value["aws_config_file"] = awsPath
		data, e := os.ReadFile(awsPath)
		if e == nil {
			file, e := ini.LoadSources(ini.LoadOptions{AllowNestedValues: true}, data)
			if e != nil {
				value["aws_default_state"] = "invalid"
			} else if file.HasSection("default") {
				value["aws_default_state"] = "configured_identity_not_verified"
				command := file.Section("default").Key("credential_process").String()
				for name := range cfg.Profile {
					if common.CredentialProcessMatches(command, filename, name) {
						value["aws_default_idaas_profile"] = name
						break
					}
				}
			}
		}
	}
	if env := os.Getenv("AWS_PROFILE"); env != "" {
		value["aws_environment_profile"] = env
	}

	return value
}

func listCacheMetadata(category, name string) error {
	categories := []string{"oidc", "oidc_token", "token_response", "cloud_token"}
	if category == "" {
		if name != "" {
			return fmt.Errorf("specify --category when filtering cache metadata with --name")
		}
		utils.Stdout.Printf("Categories: %s\n", strings.Join(categories, ", "))
		return nil
	}
	allowed := false
	for _, c := range categories {
		if category == c {
			allowed = true
		}
	}
	if !allowed {
		return fmt.Errorf("unknown cache category %q", category)
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return err
	}
	entries, err := os.ReadDir(filepath.Join(home, constants.ConfigRootDir, constants.ConfigIdaasDir, category))
	if os.IsNotExist(err) {
		utils.Stdout.Println("No cache entries.")
		return nil
	}
	if err != nil {
		return err
	}
	for _, entry := range entries {
		if !entry.IsDir() && (name == "" || strings.Contains(entry.Name(), name)) {
			info, e := entry.Info()
			if e != nil {
				return e
			}
			utils.Stdout.Printf("%s\t%d bytes\n", entry.Name(), info.Size())
		}
	}
	return nil
}

// --- helpers ---

func normalizeDomain(domain string) string {
	domain = fmt.Sprintf("%s", domain)
	for _, prefix := range []string{"https://", "http://"} {
		if len(domain) > len(prefix) && domain[:len(prefix)] == prefix {
			domain = domain[len(prefix):]
		}
	}
	// trim trailing slash
	if len(domain) > 0 && domain[len(domain)-1] == '/' {
		domain = domain[:len(domain)-1]
	}
	return domain
}

func contains(s, substr string) bool {
	for i := 0; i+len(substr) <= len(s); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

func checkServe() bool {
	client := &http.Client{Timeout: 1 * time.Second}
	resp, err := client.Get("http://127.0.0.1:1127/version")
	if err != nil {
		return false
	}
	defer resp.Body.Close()
	return resp.StatusCode == 200
}

func boolStr(b bool, trueStr, falseStr string) string {
	if b {
		return trueStr
	}
	return falseStr
}
