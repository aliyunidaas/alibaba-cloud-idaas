package access_token

import (
	"fmt"

	"github.com/aliyunidaas/alibaba-cloud-idaas/commands/agent/util"
	"github.com/aliyunidaas/alibaba-cloud-idaas/idp"
	"github.com/aliyunidaas/alibaba-cloud-idaas/utils"
	"github.com/urfave/cli/v2"
)

var (
	stringFlagConfig = &cli.StringFlag{
		Name:    "config",
		Aliases: []string{"c"},
		Usage:   "IDaaS Config",
	}
	stringFlagProfile = &cli.StringFlag{
		Name:    "profile",
		Aliases: []string{"p"},
		Usage:   "IDaaS Profile",
	}
	stringFlagScope = &cli.StringFlag{
		Name:    "scope",
		Aliases: []string{"s"},
		Usage:   "Scope, default read from config file, format audience|scope-value, e.g. urn:cloud:idaas:pam|.all",
	}
	boolFlagForceNew = &cli.BoolFlag{
		Name:    "force-new",
		Aliases: []string{"N"},
		Usage:   "Force fetch access token",
	}
)

func BuildSubcommand() *cli.Command {
	flags := []cli.Flag{
		stringFlagConfig,
		stringFlagProfile,
		stringFlagScope,
		boolFlagForceNew,
	}
	return &cli.Command{
		Name:  "access-token",
		Usage: "Get agent access token",
		Flags: flags,
		Action: func(context *cli.Context) error {
			configFilename := context.String("config")
			profile := context.String("profile")
			scope := context.String("scope")
			forceNew := context.Bool("force-new")
			return getAgentAccessToken(configFilename, profile, scope, forceNew)
		},
	}
}

func getAgentAccessToken(configFilename, profile, scope string, forceNew bool) error {
	agentConfig, err := util.GetClonedAgentConfig(configFilename, profile, scope)
	if err != nil {
		return err
	}

	fetchOidcTokenOptions := &idp.FetchOidcTokenOptions{
		ForceNew: forceNew,
		CacheKey: agentConfig.AccessTokenProvider.GetCacheKey(),
	}
	accessToken, err := idp.FetchOidcToken(profile, agentConfig.AccessTokenProvider, fetchOidcTokenOptions)
	if err != nil {
		return fmt.Errorf("fetch access token error: %s", err)
	}

	utils.Stdout.Print(accessToken)
	return nil
}
