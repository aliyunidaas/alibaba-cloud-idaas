package show_token

import (
	"github.com/aliyunidaas/alibaba-cloud-idaas/cloud"
	"github.com/aliyunidaas/alibaba-cloud-idaas/cloud/oidc"
	"github.com/aliyunidaas/alibaba-cloud-idaas/commands/common"
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
	stringFlagOidcField = &cli.StringFlag{
		Name:  "oidc-field",
		Usage: "Fetch OIDC filed (id_token or access_token)",
	}
	boolFlagNoColor = &cli.BoolFlag{
		Name:  "no-color",
		Usage: "Output without color",
	}
	boolFlagForceNew = &cli.BoolFlag{
		Name:    "force-new",
		Aliases: []string{"N"},
		Usage:   "Force fetch cloud token, ignore all cache",
	}
	boolFlagForceNewCloudToken = &cli.BoolFlag{
		Name:  "force-new-cloud-token",
		Usage: "Force fetch cloud token (lower cache enabled)",
	}
)

func BuildCommand() *cli.Command {
	flags := []cli.Flag{
		stringFlagConfig,
		stringFlagProfile,
		stringFlagOidcField,
		boolFlagNoColor,
		boolFlagForceNew,
		boolFlagForceNewCloudToken,
	}
	return &cli.Command{
		Name:  "show-token",
		Usage: "Show cloud STS token",
		Flags: flags,
		Action: func(context *cli.Context) error {
			configFilename := context.String("config")
			profile := context.String("profile")
			oidcField := context.String("oidc-field")
			color := !context.Bool("no-color")
			forceNew := context.Bool("force-new")
			forceNewCloudToken := context.Bool("force-new-cloud-token")
			return fetchAndShowToken(configFilename, profile, oidcField, forceNew, forceNewCloudToken, color)
		},
	}
}

func fetchAndShowToken(configFilename, profile, oidcField string, forceNew, forceNewCloudToken bool, color bool) error {
	options := &cloud.FetchCloudStsOptions{
		ForceNew:           forceNew,
		ForceNewCloudToken: forceNewCloudToken,
	}
	oidcTokenType := oidc.GetOidcTokenType(oidcField)
	options.FetchOidcTokenType = oidcTokenType

	sts, _, err := cloud.FetchCloudStsFromDefaultConfig(configFilename, profile, options)
	if err != nil {
		return err
	}

	return common.ShowToken(sts, oidcTokenType, true, color)
}
