package get_secret

import (
	"encoding/json"
	"fmt"

	"github.com/aliyunidaas/alibaba-cloud-idaas/cloud/credential"
	"github.com/aliyunidaas/alibaba-cloud-idaas/commands/agent/util"
	"github.com/aliyunidaas/alibaba-cloud-idaas/idp"
	"github.com/aliyunidaas/alibaba-cloud-idaas/utils"
	"github.com/itchyny/gojq"
	"github.com/urfave/cli/v2"
)

const (
	ScopeSecretObtain = "urn:cloud:idaas:pam|credential:obtain"
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
		Usage:   "Scope, format audience|scope-value, default: " + ScopeSecretObtain,
	}
	stringFlagJsonQuery = &cli.StringFlag{
		Name:    "json-query",
		Aliases: []string{"q"},
		Usage:   "JSON query, e.g. .default_model.value.apiKeyContent.apiKey",
	}
	stringSliceFlagName = &cli.StringSliceFlag{
		Name:    "name",
		Aliases: []string{"n"},
		Usage:   "Secret name",
	}
	boolFlagRaw = &cli.BoolFlag{
		Name:  "raw",
		Usage: "Output raw response",
	}
	boolFlagStringRaw = &cli.BoolFlag{
		Name:  "string-raw",
		Usage: "Output raw JSON string, only if the value is string type",
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
		stringFlagJsonQuery,
		stringSliceFlagName,
		boolFlagRaw,
		boolFlagStringRaw,
		boolFlagForceNew,
	}
	return &cli.Command{
		Name:    "get-secret",
		Aliases: []string{"secret"},
		Usage:   "Get agent secret",
		Flags:   flags,
		Action: func(context *cli.Context) error {
			configFilename := context.String("config")
			profile := context.String("profile")
			scope := context.String("scope")
			jsonQuery := context.String("json-query")
			names := context.StringSlice("name")
			raw := context.Bool("raw")
			stringRaw := context.Bool("string-raw")
			forceNew := context.Bool("force-new")
			return getAgentSecrets(configFilename, profile, scope, jsonQuery, names, raw, stringRaw, forceNew)
		},
	}
}

type SecretResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
	Value   any    `json:"value,omitempty"`
}

func getAgentSecrets(configFilename, profile, scope, jsonQuery string, names []string, raw, stringRaw, forceNew bool) error {
	if len(names) == 0 {
		return fmt.Errorf("no secret names provided")
	}

	if scope == "" {
		scope = ScopeSecretObtain
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

	// endpoint
	credentialEndpoint, err := agentConfig.GetCredentialEndpoint()
	if err != nil {
		return err
	}

	var secrets = map[string]*SecretResponse{}
	for _, name := range names {
		cred, err := credential.FetchCredential(name, &credential.CredentialOptions{
			Endpoint:  credentialEndpoint,
			AccessKey: accessToken,
		})
		if err != nil {
			secrets[name] = &SecretResponse{
				Success: false,
				Message: fmt.Sprintf("fetch secret error: %s", err),
				Value:   nil,
			}
		} else if cred == nil {
			secrets[name] = &SecretResponse{
				Success: false,
				Message: "not_found",
				Value:   nil,
			}
		} else {
			if raw {
				secrets[name] = &SecretResponse{
					Success: true,
					Message: "ok",
					Value:   cred,
				}
			} else {
				secrets[name] = &SecretResponse{
					Success: true,
					Message: "ok",
					Value:   cred.CredentialContent,
				}
			}
		}
	}
	secretsJson, err := json.Marshal(secrets)
	if err != nil {
		return err
	}

	err = printSecrets(secretsJson, jsonQuery, stringRaw)
	if err != nil {
		return err
	}

	return nil
}

func printSecrets(secretsJsonBytes []byte, jsonQuery string, stringRaw bool) error {
	if jsonQuery == "" {
		utils.Stdout.Println(string(secretsJsonBytes))
	} else {
		query, err := gojq.Parse(jsonQuery)
		if err != nil {
			return err
		}
		var clonedSecrets map[string]interface{}
		err = json.Unmarshal(secretsJsonBytes, &clonedSecrets)
		if err != nil {
			return err
		}
		iter := query.Run(clonedSecrets)
		for {
			v, ok := iter.Next()
			if !ok {
				break
			}
			if err, ok := v.(error); ok {
				if err, ok := err.(*gojq.HaltError); ok && err.Value() == nil {
					break
				}
				return err
			}
			if stringRaw {
				vStr, ok := v.(string)
				if ok {
					utils.Stdout.Print(vStr)
					continue
				}
			}
			filterJson, err := json.Marshal(v)
			if err != nil {
				return err
			}
			utils.Stdout.Println(string(filterJson))
		}
	}
	return nil
}
