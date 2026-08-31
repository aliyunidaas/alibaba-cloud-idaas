package show

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/aliyunidaas/alibaba-cloud-idaas/commands/common"
	"github.com/aliyunidaas/alibaba-cloud-idaas/config"
	"github.com/aliyunidaas/alibaba-cloud-idaas/utils"
	"github.com/urfave/cli/v2"
)

const listRolesPathFormat = "%s/v2/%s/cloudAccountRoles/_/actions/listAssumableCloudAccountRoles"

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
	for name, sts := range cloudConfig.Profile {
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
	instance = normalizeDomain(instance)

	// Infer instance + client-id from existing profiles if not provided
	if instance == "" || clientId == "" {
		if configFile == "" {
			configFile, _ = config.GetDefaultCloudCredentialConfigFile()
		}
		if inferred := common.InferFromProfiles(configFile); inferred != nil {
			if instance == "" {
				instance = inferred.Instance
			}
			if clientId == "" {
				clientId = inferred.ClientId
			}
		}
	}
	if instance == "" {
		return fmt.Errorf("instance domain is required. Run 'alibaba-cloud-idaas login --instance <域名> --client-id <app_id>' first, or specify --instance")
	}
	instance = normalizeDomain(instance)
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
	listUrl := fmt.Sprintf(listRolesPathFormat, pop, discovery.InstanceId)
	headers := map[string]string{"Authorization": "Bearer " + accessToken}
	body, err := utils.FetchAsString(httpClient, utils.HttpMethodGet, listUrl, headers)
	if err != nil {
		return fmt.Errorf("list roles failed: %w", err)
	}
	var list assumableRoleList
	if err := json.Unmarshal([]byte(body), &list); err != nil {
		return fmt.Errorf("unmarshal failed: %w", err)
	}

	if jsonOutput {
		utils.Stdout.Println(body)
		return nil
	}

	if len(list.Entities) == 0 {
		utils.Stdout.Println("No assumable cloud roles found.")
		return nil
	}
	utils.Stdout.Printf("Found %d role(s):\n\n", len(list.Entities))
	utils.Stdout.Printf("  %-30s %-55s %-15s %s\n", "NAME", "EXTERNAL ID", "VENDOR", "STATUS")
	for _, r := range list.Entities {
		utils.Stdout.Printf("  %-30s %-55s %-15s %s\n",
			r.CloudAccountRoleName, r.CloudAccountRoleExternalId, r.CloudAccountVendorType, r.Status)
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
			// Delegate to existing show_cache command logic by re-running it
			return nil // placeholder — existing show-cache command remains as alias
		},
	}
}

// --- show token ---

func buildShowToken() *cli.Command {
	return &cli.Command{
		Name:    "token",
		Aliases: []string{"t"},
		Usage:   "Show current credential token (human-readable)",
		Flags: []cli.Flag{
			&cli.StringFlag{Name: "profile", Aliases: []string{"p"}, Usage: "Profile"},
			&cli.StringFlag{Name: "config", Aliases: []string{"c"}, Usage: "Config file"},
			&cli.StringFlag{Name: "oidc-field", Usage: "OIDC field (id_token or access_token)"},
			&cli.BoolFlag{Name: "no-color", Usage: "No color"},
			&cli.BoolFlag{Name: "force-new", Aliases: []string{"N"}, Usage: "Force refresh"},
		},
		Action: func(c *cli.Context) error {
			return nil // placeholder — existing show-token command remains as alias
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
	if !ok {
		utils.Stdout.Printf("Profile '%s' not found.\n", profile)
		return nil
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
		utils.Stdout.Printf(`{"profile":"%s","provider":"%s","instance_id":"%s","serve_running":%t}`+"\n",
			profile, providerType, instanceId, serveRunning)
	} else {
		utils.Stdout.Printf("Profile:   %s\n", profile)
		utils.Stdout.Printf("Provider:  %s\n", providerType)
		if instanceId != "" {
			utils.Stdout.Printf("Instance:  %s\n", instanceId)
		}
		utils.Stdout.Printf("Serve:     %s\n", boolStr(serveRunning, "running", "not running"))
		utils.Stdout.Printf("\nProfiles:  %d total\n", len(cloudConfig.Profile))
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
		},
		Action: func(c *cli.Context) error {
			return showInstance(c.String("instance"), c.Bool("vpc"))
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
	return &cli.Command{
		Name:    "signer-key",
		Aliases: []string{"sk"},
		Usage:   "Show external signer public key",
		Flags: []cli.Flag{
			&cli.StringFlag{Name: "profile", Aliases: []string{"p"}, Usage: "Profile"},
			&cli.StringFlag{Name: "config", Aliases: []string{"c"}, Usage: "Config file"},
		},
		Action: func(c *cli.Context) error {
			return nil // placeholder — existing show-signer-public-key remains as alias
		},
	}
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
