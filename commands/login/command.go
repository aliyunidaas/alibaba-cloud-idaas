package login

import (
	"strings"

	"github.com/aliyunidaas/alibaba-cloud-idaas/commands/common"
	"github.com/aliyunidaas/alibaba-cloud-idaas/config"
	"github.com/aliyunidaas/alibaba-cloud-idaas/utils"
	"github.com/pkg/errors"
	"github.com/urfave/cli/v2"
)

var (
	stringFlagInstance = &cli.StringFlag{
		Name:    "instance",
		Aliases: []string{"i"},
		Usage:   "IDaaS instance domain (first-time login mode)",
	}
	stringFlagProfile = &cli.StringFlag{
		Name:    "profile",
		Aliases: []string{"p"},
		Usage:   "Existing profile name (refresh mode)",
	}
	stringFlagScope = &cli.StringFlag{
		Name:    "scope",
		Aliases: []string{"s"},
		Usage:   "Space-delimited audience|scope-value pairs (default: urn:cloud:idaas:pam|.all)",
	}
	stringFlagClientId = &cli.StringFlag{
		Name:  "client-id",
		Usage: "Broker client application id (default: from discovery or iap_developer)",
	}
	stringFlagConfig = &cli.StringFlag{
		Name:    "config",
		Aliases: []string{"c"},
		Usage:   "IDaaS config file (default ~/.aliyun/alibaba-cloud-idaas.json)",
	}
	boolFlagForceNew = &cli.BoolFlag{
		Name:    "force-new",
		Aliases: []string{"N"},
		Usage:   "Force device-code login, ignore cached token",
	}
)

func BuildCommand() *cli.Command {
	return &cli.Command{
		Name:  "login",
		Usage: "Device-code login to IDaaS instance and cache access token",
		Flags: []cli.Flag{
			stringFlagInstance,
			stringFlagProfile,
			stringFlagScope,
			stringFlagClientId,
			stringFlagConfig,
			boolFlagForceNew,
		},
		Action: func(c *cli.Context) error {
			instance := c.String("instance")
			profile := c.String("profile")
			scope := c.String("scope")
			clientId := c.String("client-id")
			configFile := c.String("config")
			forceNew := c.Bool("force-new")
			if instance == "" && profile == "" {
				return errors.New("either --instance or --profile is required")
			}
			if instance != "" && profile != "" {
				return errors.New("--instance and --profile are mutually exclusive")
			}
			return runLogin(instance, profile, scope, clientId, configFile, forceNew)
		},
	}
}

func runLogin(instance, profile, scope, clientId, configFile string, forceNew bool) error {
	if instance != "" {
		return loginByInstance(instance, scope, clientId, forceNew)
	}
	return loginByProfile(profile, scope, clientId, configFile, forceNew)
}

func loginByInstance(instance, scope, clientId string, forceNew bool) error {
	instance = normalizeDomain(instance)
	httpClient := utils.BuildHttpClient()

	utils.Stdout.Printf("Discovering instance %s ...\n", instance)
	discovery, err := common.FetchInstanceDiscovery(httpClient, instance)
	if err != nil {
		return errors.Wrap(err, "instance discovery failed")
	}
	if err := common.ValidateDiscovery(discovery); err != nil {
		return err
	}
	issuer := discovery.DefaultAuthorizationServerIssuer
	if clientId == "" {
		clientId = common.ResolveClientId(discovery)
	}
	utils.Stdout.Printf("Logging in (issuer=%s, client=%s) ...\n", issuer, clientId)
	utils.Stdout.Println("Complete SSO/MFA in browser ...")
	_, err = common.DoLogin(issuer, clientId, scope, forceNew)
	if err != nil {
		return errors.Wrap(err, "login failed")
	}
	utils.Stdout.Println("Login successful.")
	return nil
}

func loginByProfile(profile, scope, clientId, configFile string, forceNew bool) error {
	if configFile == "" {
		var err error
		configFile, err = config.GetDefaultCloudCredentialConfigFile()
		if err != nil {
			return err
		}
	}
	_, cloudStsConfig, err := config.FindProfile(configFile, profile, false)
	if err != nil {
		return errors.Wrapf(err, "find profile `%s`", profile)
	}
	var provider *config.OidcTokenProviderConfig
	if cloudStsConfig.CloudAccount != nil && cloudStsConfig.CloudAccount.AccessTokenProvider != nil {
		provider = cloudStsConfig.CloudAccount.AccessTokenProvider
	} else if cloudStsConfig.OidcToken != nil {
		provider = cloudStsConfig.OidcToken
	} else {
		return errors.New("profile has no OIDC token provider (device_code)")
	}
	if provider.OidcTokenProviderDeviceCode == nil {
		return errors.New("profile has no device_code provider")
	}
	dc := provider.OidcTokenProviderDeviceCode
	issuer := dc.Issuer
	if clientId == "" {
		clientId = dc.ClientId
	}
	if scope == "" {
		scope = dc.Scope
	}
	utils.Stdout.Printf("Refreshing login (profile=%s, issuer=%s) ...\n", profile, issuer)
	utils.Stdout.Println("Complete SSO/MFA in browser ...")
	_, err = common.DoLogin(issuer, clientId, scope, forceNew)
	if err != nil {
		return errors.Wrap(err, "login failed")
	}
	utils.Stdout.Println("Login successful.")
	return nil
}

func normalizeDomain(domain string) string {
	domain = strings.TrimSpace(domain)
	domain = strings.TrimPrefix(domain, "https://")
	domain = strings.TrimPrefix(domain, "http://")
	domain = strings.TrimRight(domain, "/")
	return domain
}
