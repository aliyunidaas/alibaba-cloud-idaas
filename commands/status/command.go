package status

import (
	"fmt"
	"net/http"
	"time"

	"github.com/aliyunidaas/alibaba-cloud-idaas/commands/common"
	"github.com/aliyunidaas/alibaba-cloud-idaas/config"
	"github.com/aliyunidaas/alibaba-cloud-idaas/utils"
	"github.com/urfave/cli/v2"
)

func BuildCommand() *cli.Command {
	return &cli.Command{
		Name:  "status",
		Usage: "Show current profile, login state, and serve daemon status",
		Flags: []cli.Flag{
			&cli.StringFlag{Name: "profile", Aliases: []string{"p"}, Usage: "Profile (default: IDAAS_PROFILE or current_profile)"},
			&cli.StringFlag{Name: "config", Aliases: []string{"c"}, Usage: "IDaaS config file"},
			&cli.BoolFlag{Name: "json", Usage: "Machine-readable JSON output"},
		},
		Action: func(c *cli.Context) error {
			return runStatus(c.String("profile"), c.String("config"), c.Bool("json"))
		},
	}
}

func runStatus(flagProfile, configFile string, jsonOutput bool) error {
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
		return fmt.Errorf("read config: %w", err)
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
	// Determine provider type
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

	// Check serve daemon
	serveRunning := checkServe()

	// Output
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
