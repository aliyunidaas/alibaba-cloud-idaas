package token_exchange

import (
	"github.com/aliyunidaas/alibaba-cloud-idaas/cloud/oidc"
	"github.com/aliyunidaas/alibaba-cloud-idaas/commands/agent/util"
	oidc2 "github.com/aliyunidaas/alibaba-cloud-idaas/oidc"
	"github.com/aliyunidaas/alibaba-cloud-idaas/utils"
	"github.com/pkg/errors"
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
		Usage:   "Scope",
	}
	stringFlagSubjectTokenType = &cli.StringFlag{
		Name:    "subject-token-type",
		Aliases: []string{"T"},
		Usage:   "Token exchange subject token type, only supports: " + oidc2.TokenTypeAccessToken + " (default)",
	}
	stringFlagSubjectToken = &cli.StringFlag{
		Name:     "subject-token",
		Aliases:  []string{"S"},
		Required: true,
		Usage:    "Token exchange subject token",
	}
	stringFlagRequestTokenType = &cli.StringFlag{
		Name:    "request-token-type",
		Aliases: []string{"R"},
		Usage:   "Token exchange request token type, only supports: " + oidc2.TokenTypeAccessToken + " (default)",
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
		stringFlagSubjectTokenType,
		stringFlagSubjectToken,
		stringFlagRequestTokenType,
		boolFlagForceNew,
	}
	return &cli.Command{
		Name:  "token-exchange",
		Usage: "Agent token exchange",
		Flags: flags,
		Action: func(context *cli.Context) error {
			configFilename := context.String("config")
			profile := context.String("profile")
			scope := context.String("scope")
			subjectTokenType := context.String("subject-token-type")
			subjectToken := context.String("subject-token")
			requestTokenType := context.String("request-token-type")
			forceNew := context.Bool("force-new")
			return exchangeToken(configFilename, profile, subjectTokenType, subjectToken, requestTokenType, scope, forceNew)
		},
	}
}

func exchangeToken(configFilename, profile, subjectTokenType, subjectToken, requestTokenType, scope string, forceNew bool) error {
	isSupportedSubjectTokenType := subjectTokenType == "" || subjectTokenType == oidc2.TokenTypeAccessToken
	if !isSupportedSubjectTokenType {
		return errors.Errorf("invalid subject token type: %s", requestTokenType)
	}
	if subjectToken == "" {
		return errors.Errorf("empty subject token")
	}
	isSupportedRequestTokenType := requestTokenType == "" || requestTokenType == oidc2.TokenTypeAccessToken
	if !isSupportedRequestTokenType {
		return errors.Errorf("invalid request token type: %s", requestTokenType)
	}

	agentConfig, err := util.GetClonedAgentConfig(configFilename, profile, scope)
	if err != nil {
		return err
	}

	oidcTokenConfigOptions := &oidc.FetchOidcTokenConfigOptions{
		OidcCommonOptions: &oidc2.OidcCommonOptions{
			GrantType:        oidc2.GrantTypeTokenExchange, // only supports access token now
			Scope:            scope,
			SubjectTokenType: oidc2.TokenTypeAccessToken,
			SubjectToken:     subjectToken,
		},
		ForceNew:       forceNew,
		FetchTokenType: oidc.FetchAccessToken, // only supports access token now
	}
	oidcToken, err := oidc.FetchOidcToken(profile, agentConfig.AccessTokenProvider, oidcTokenConfigOptions)
	if err != nil {
		return err
	}
	utils.Stdout.Print(oidcToken.AccessToken)
	return nil
}
