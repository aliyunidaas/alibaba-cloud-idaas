package put_secret

import (
	"encoding/json"
	"fmt"
	"os"
	"os/user"

	"github.com/aliyunidaas/alibaba-cloud-idaas/cloud/credential"
	"github.com/aliyunidaas/alibaba-cloud-idaas/commands/agent/util"
	"github.com/aliyunidaas/alibaba-cloud-idaas/idp"
	"github.com/aliyunidaas/alibaba-cloud-idaas/utils"
	"github.com/urfave/cli/v2"
)

const (
	ScopeSecretManage = "urn:cloud:idaas:pam|credential:manage"
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
		Usage:   "Scope, format audience|scope-value, default: " + ScopeSecretManage,
	}
	stringFlagName = &cli.StringFlag{
		Name:     "name",
		Aliases:  []string{"n"},
		Required: true,
		Usage:    "Secret name",
	}
	stringFlagDisplayName = &cli.StringFlag{
		Name:  "display-name",
		Usage: "Secret display name, optional, default same as name",
	}
	stringFlagLabel = &cli.StringFlag{
		Name:  "label",
		Usage: "Secret label, optional, e.g. llm(Large Language Model)、saas(SaaS)",
	}
	stringFlagDescription = &cli.StringFlag{
		Name:  "description",
		Usage: "Secret description, optional",
	}
	stringFlagExternalIdentifier = &cli.StringFlag{
		Name:  "external-identifier",
		Usage: "Secret external identifier",
	}
	stringFlagValue = &cli.StringFlag{
		Name:     "value",
		Aliases:  []string{"v"},
		Required: true,
		Usage:    "Secret value",
	}
	boolFlagForceNew = &cli.BoolFlag{
		Name:    "force-new",
		Aliases: []string{"N"},
		Usage:   "Force fetch access token",
	}
	boolFlagReturnCiphertext = &cli.BoolFlag{
		Name:  "return-ciphertext",
		Usage: "Return credential ciphertext from server",
	}
)

func BuildSubcommand() *cli.Command {
	flags := []cli.Flag{
		stringFlagConfig,
		stringFlagProfile,
		stringFlagScope,
		stringFlagName,
		stringFlagDisplayName,
		stringFlagDescription,
		stringFlagLabel,
		stringFlagValue,
		stringFlagExternalIdentifier,
		boolFlagForceNew,
		boolFlagReturnCiphertext,
	}
	return &cli.Command{
		Name:  "put-secret",
		Usage: "Put agent secret (api-key)",
		Flags: flags,
		Action: func(context *cli.Context) error {
			configFilename := context.String("config")
			profile := context.String("profile")
			scope := context.String("scope")
			name := context.String("name")
			displayName := context.String("display-name")
			label := context.String("label")
			description := context.String("description")
			externalId := context.String("external-identifier")
			value := context.String("value")
			forceNew := context.Bool("force-new")
			returnCiphertext := context.Bool("return-ciphertext")
			return putAgentSecret(configFilename, profile, scope, name, displayName, label, description, externalId, value, forceNew, returnCiphertext)
		},
	}
}

type PutSecretResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
}

func putAgentSecret(configFilename, profile, scope, name, displayName, label, description, externalId, value string, forceNew, returnCiphertext bool) error {
	if scope == "" {
		scope = ScopeSecretManage
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

	credentialEndpoint, err := agentConfig.CreateUserExclusiveCredentialEndpoint()
	if err != nil {
		return err
	}

	if displayName == "" {
		displayName = name
	}
	if description == "" {
		description = "API Key"
		currentUser, err := user.Current()
		if err == nil && currentUser != nil {
			description = description + "; user: " + currentUser.Username
		}
		hostname, err := os.Hostname()
		if err == nil {
			description = description + "; hostname: " + hostname
		}
	}

	response, err := credential.CreateCredentialApiKey(name, displayName, label, value, description, externalId, returnCiphertext, &credential.CredentialOptions{
		Endpoint:  credentialEndpoint,
		AccessKey: accessToken,
	})
	if err != nil {
		return err
	}

	if returnCiphertext {
		if response == nil {
			return fmt.Errorf("unexpected nil response when returnCiphertext is true")
		}
		if response.CredentialCiphertext == "" {
			errResponse := &PutSecretResponse{
				Success: false,
				Message: "server returned empty credentialCiphertext",
			}
			outputBytes, marshalErr := json.Marshal(errResponse)
			if marshalErr != nil {
				return fmt.Errorf("failed to marshal error response: %s", marshalErr)
			}
			utils.Stdout.Println(string(outputBytes))
			return fmt.Errorf("server returned empty credentialCiphertext")
		}
		outputBytes, marshalErr := json.Marshal(response)
		if marshalErr != nil {
			return fmt.Errorf("failed to marshal response: %s", marshalErr)
		}
		utils.Stdout.Println(string(outputBytes))
	} else {
		utils.Stdout.Println("{\"success\":true}")
	}
	return nil
}
