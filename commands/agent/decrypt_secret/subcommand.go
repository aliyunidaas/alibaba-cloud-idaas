package decrypt_secret

import (
	"encoding/json"
	"fmt"

	"github.com/aliyunidaas/alibaba-cloud-idaas/cloud/credential"
	"github.com/aliyunidaas/alibaba-cloud-idaas/commands/agent/util"
	"github.com/aliyunidaas/alibaba-cloud-idaas/idp"
	"github.com/aliyunidaas/alibaba-cloud-idaas/utils"
	"github.com/urfave/cli/v2"
)

const (
	ScopeSecretDecrypt = "urn:cloud:idaas:pam|credential:decrypt"
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
		Usage:   "Scope, format audience|scope-value, default: " + ScopeSecretDecrypt,
	}
	stringFlagName = &cli.StringFlag{
		Name:     "name",
		Aliases:  []string{"n"},
		Required: true,
		Usage:    "Credential identifier",
	}
	stringFlagCiphertext = &cli.StringFlag{
		Name:     "ciphertext",
		Required: true,
		Usage:    "Credential ciphertext to decrypt",
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
		stringFlagName,
		stringFlagCiphertext,
		boolFlagForceNew,
	}
	return &cli.Command{
		Name:  "decrypt-secret",
		Usage: "Decrypt agent secret ciphertext",
		Flags: flags,
		Action: func(context *cli.Context) error {
			configFilename := context.String("config")
			profile := context.String("profile")
			scope := context.String("scope")
			name := context.String("name")
			ciphertext := context.String("ciphertext")
			forceNew := context.Bool("force-new")
			return decryptAgentSecret(configFilename, profile, scope, name, ciphertext, forceNew)
		},
	}
}

func decryptAgentSecret(configFilename, profile, scope, name, ciphertext string, forceNew bool) error {
	if scope == "" {
		scope = ScopeSecretDecrypt
	}
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

	decryptEndpoint, err := agentConfig.DecryptUserExclusiveCredentialEndpoint()
	if err != nil {
		return err
	}

	response, err := credential.DecryptCredential(name, ciphertext, &credential.CredentialOptions{
		Endpoint:  decryptEndpoint,
		AccessKey: accessToken,
	})
	if err != nil {
		return err
	}

	outputBytes, err := json.Marshal(response)
	if err != nil {
		return fmt.Errorf("failed to marshal response: %s", err)
	}
	utils.Stdout.Println(string(outputBytes))
	return nil
}
