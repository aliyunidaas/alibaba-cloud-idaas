package openclaw_secret

import (
	"fmt"
	"io"
	"os"

	"github.com/aliyunidaas/alibaba-cloud-idaas/cloud/credential"
	"github.com/aliyunidaas/alibaba-cloud-idaas/cloud/openclaw"
	"github.com/aliyunidaas/alibaba-cloud-idaas/config"
	"github.com/aliyunidaas/alibaba-cloud-idaas/idaaslog"
	"github.com/aliyunidaas/alibaba-cloud-idaas/idp"
	"github.com/aliyunidaas/alibaba-cloud-idaas/oidc"
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
	boolFlagForceNew = &cli.BoolFlag{
		Name:    "force-new",
		Aliases: []string{"N"},
		Usage:   "Force fetch cloud token, ignore all cache",
	}
)

func BuildCommand() *cli.Command {
	flags := []cli.Flag{
		stringFlagConfig,
		stringFlagProfile,
		boolFlagForceNew,
	}
	return &cli.Command{
		Name:  "openclaw-secret",
		Usage: "Fetch OpenClaw agent secrets, spec: https://docs.openclaw.ai/gateway/secrets JSON format",
		Flags: flags,
		Action: func(context *cli.Context) error {
			configFilename := context.String("config")
			profile := context.String("profile")
			forceNew := context.Bool("force-new")

			return fetchOpenClawSecret(configFilename, profile, forceNew)
		},
	}
}

func fetchOpenClawSecret(configFilename, profile string, forceNew bool) error {
	profile, cloudStsConfig, err := config.FindProfile(configFilename, profile, false)
	if err != nil {
		return fmt.Errorf("find profile `%s` error: %s", profile, err)
	}

	if cloudStsConfig.Agent == nil {
		return fmt.Errorf("not agent profile `%s`", profile)
	}
	agentConfig := cloudStsConfig.Agent

	// endpoint
	credentialEndpoint, err := agentConfig.GetCredentialEndpoint()
	if err != nil {
		return err
	}

	stdin, err := io.ReadAll(os.Stdin)
	if err != nil {
		return fmt.Errorf("read request from stdin error: %s", err)
	}
	requestJson := string(stdin)
	idaaslog.Unsafe.PrintfLn("Std in: %s", requestJson)
	openClawSecretProviderRequest, err := openclaw.UnmarshalRequest(requestJson)
	if err != nil {
		return fmt.Errorf("parse request from stdin error: %s", err)
	}

	if openClawSecretProviderRequest.ProtocolVersion != openclaw.ProtocolVersion1 {
		return fmt.Errorf("unsupported protocol version: %s", openClawSecretProviderRequest.ProtocolVersion)
	}

	// access token
	fetchOidcTokenOptions := &idp.FetchOidcTokenOptions{
		ForceNew: forceNew,
		CacheKey: agentConfig.AccessTokenProvider.GetCacheKey(),
	}
	// MUST be Access Token for Cloud Account Token obtain
	agentConfig.AccessTokenProvider.TokenType = oidc.TokenAccessToken
	accessToken, err := idp.FetchOidcToken(profile, agentConfig.AccessTokenProvider, fetchOidcTokenOptions)
	if err != nil {
		return fmt.Errorf("fetch access token error: %s", err)
	}

	values := map[string]string{}
	errors := map[string]*openclaw.OpenClawSecretProviderResponseErrorMessage{}

	idaaslog.Unsafe.PrintfLn("Access token: %s", accessToken)

	for _, id := range openClawSecretProviderRequest.Ids {
		cred, err := credential.FetchCredential(credentialEndpoint, id, accessToken)
		if err != nil {
			idaaslog.Error.PrintfLn("failed to fetch %s, error: %s", id, err)
			errors[id] = &openclaw.OpenClawSecretProviderResponseErrorMessage{
				Message: fmt.Sprintf("fetch credential error: %s", err),
			}
		} else {
			if cred == nil {
				idaaslog.Error.PrintfLn("not found: %s", id)
				errors[id] = &openclaw.OpenClawSecretProviderResponseErrorMessage{
					Message: "not found",
				}
			} else if cred.CredentialContent.ApiKeyContent != nil {
				idaaslog.Unsafe.PrintfLn("fetch api key %s = %s", id, cred.CredentialContent.ApiKeyContent.ApiKey)
				values[id] = cred.CredentialContent.ApiKeyContent.ApiKey
			} else {
				idaaslog.Error.PrintfLn("not api key: %s", id)
				errors[id] = &openclaw.OpenClawSecretProviderResponseErrorMessage{
					Message: "not a api key",
				}
			}
		}
	}

	response := openclaw.OpenClawSecretProviderResponse{
		ProtocolVersion: openclaw.ProtocolVersion1,
		Values:          &values,
		Errors:          &errors,
	}
	if len(*response.Errors) == 0 {
		response.Errors = nil
	}
	responseJson, err := response.Marshal()
	if err != nil {
		return fmt.Errorf("marshal response error: %s", err)
	}
	idaaslog.Unsafe.PrintfLn("Std out: %s", responseJson)
	fmt.Println(responseJson)
	return nil
}
