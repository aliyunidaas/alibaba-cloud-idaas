package idp

import (
	openapi "github.com/alibabacloud-go/darabonba-openapi/v2/client"
	eiam20211201 "github.com/alibabacloud-go/eiam-20211201/v2/client"
	util "github.com/alibabacloud-go/tea-utils/v2/service"
	"github.com/alibabacloud-go/tea/tea"
	"github.com/aliyun/credentials-go/credentials"
	"github.com/aliyunidaas/alibaba-cloud-idaas/config"
	"github.com/aliyunidaas/alibaba-cloud-idaas/idaaslog"
	"github.com/aliyunidaas/alibaba-cloud-idaas/oidc"
	"github.com/pkg/errors"
)

func FetchAccessTokenOpenApi(openApiConfig *config.OpenApiConfig) (*oidc.TokenResponse, error) {
	credentialsConfig := convertToCredentialConfig(openApiConfig)
	idaaslog.Unsafe.PrintfLn("CredentialsConfig: %v", credentialsConfig)
	credential, err := credentials.NewCredential(credentialsConfig)
	if err != nil {
		return nil, errors.Wrap(err, "error creating credential")
	}

	clientConfig := &openapi.Config{
		Credential: credential,
	}
	if openApiConfig.OpenApiEndpoint == "" {
		clientConfig.Endpoint = tea.String("eiam.cn-hangzhou.aliyuncs.com")
	} else {
		clientConfig.Endpoint = tea.String(openApiConfig.OpenApiEndpoint)
	}
	eiamClient, err := eiam20211201.NewClient(clientConfig)
	if err != nil {
		return nil, errors.Wrap(err, "error creating client")
	}

	generateOauthTokenRequest := &eiam20211201.GenerateOauthTokenRequest{
		InstanceId:    tea.String(openApiConfig.InstanceId),
		ApplicationId: tea.String(openApiConfig.ApplicationId),
		Audience:      tea.String(openApiConfig.Audience),
		ScopeValues:   tea.StringSlice(openApiConfig.ScopeValues),
	}
	runtime := &util.RuntimeOptions{}

	idaaslog.Unsafe.PrintfLn("GenerateOauthTokenRequest: %v", generateOauthTokenRequest)
	resp, err := eiamClient.GenerateOauthTokenWithOptions(generateOauthTokenRequest, runtime)
	if err != nil {
		return nil, errors.Wrap(err, "error generating oauth token")
	}

	tokenResponse := resp.GetBody().GetTokenResponse()
	idaaslog.Unsafe.PrintfLn("TokenResponse: %v", tokenResponse)

	oidcTokenResponse := &oidc.TokenResponse{
		TokenType:   *tokenResponse.TokenType,
		AccessToken: *tokenResponse.AccessToken,
		ExpiresIn:   *tokenResponse.ExpiresIn,
		ExpiresAt:   *tokenResponse.ExpiresAt,
	}

	return oidcTokenResponse, nil
}

func convertToCredentialConfig(openApiConfig *config.OpenApiConfig) *credentials.Config {
	if openApiConfig.Type == "" {
		return nil
	}

	credentialsConfig := new(credentials.Config)

	if openApiConfig.Type != "" {
		credentialsConfig.SetType(openApiConfig.Type)
	}
	if openApiConfig.AccessKeyId != "" {
		credentialsConfig.SetAccessKeyId(openApiConfig.AccessKeyId)
	}
	if openApiConfig.AccessKeySecret != "" {
		credentialsConfig.SetAccessKeySecret(openApiConfig.AccessKeySecret)
	}
	if openApiConfig.SecurityToken != "" {
		credentialsConfig.SetSecurityToken(openApiConfig.SecurityToken)
	}

	if openApiConfig.OIDCProviderArn != "" {
		credentialsConfig.SetOIDCProviderArn(openApiConfig.OIDCProviderArn)
	}
	if openApiConfig.OIDCTokenFilePath != "" {
		credentialsConfig.SetOIDCTokenFilePath(openApiConfig.OIDCTokenFilePath)
	}
	if openApiConfig.RoleArn != "" {
		credentialsConfig.SetRoleArn(openApiConfig.RoleArn)
	}

	if openApiConfig.RoleArn != "" {
		credentialsConfig.SetRoleArn(openApiConfig.RoleArn)
	}
	if openApiConfig.RoleSessionName != "" {
		credentialsConfig.SetRoleSessionName(openApiConfig.RoleSessionName)
	}
	if openApiConfig.RoleSessionExpiration != 0 {
		credentialsConfig.SetRoleSessionExpiration(openApiConfig.RoleSessionExpiration)
	}
	if openApiConfig.Policy != "" {
		credentialsConfig.SetPolicy(openApiConfig.Policy)
	}
	if openApiConfig.ExternalId != "" {
		credentialsConfig.SetExternalId(openApiConfig.ExternalId)
	}
	if openApiConfig.STSEndpoint != "" {
		credentialsConfig.SetSTSEndpoint(openApiConfig.STSEndpoint)
	}

	if openApiConfig.RoleName != "" {
		credentialsConfig.SetRoleName(openApiConfig.RoleName)
	}

	if openApiConfig.Url != "" {
		credentialsConfig.SetURLCredential(openApiConfig.Url)
	}

	return credentialsConfig
}
