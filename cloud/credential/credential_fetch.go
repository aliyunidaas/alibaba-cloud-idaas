package credential

import (
	"encoding/json"
	"strings"

	"github.com/aliyunidaas/alibaba-cloud-idaas/config"
	"github.com/aliyunidaas/alibaba-cloud-idaas/idaaslog"
	"github.com/aliyunidaas/alibaba-cloud-idaas/idp"
	"github.com/aliyunidaas/alibaba-cloud-idaas/oidc"
	"github.com/aliyunidaas/alibaba-cloud-idaas/utils"
	"github.com/pkg/errors"
)

// Credential
// e.g.
//
//	"instanceId": "idaas_wrwsx2jca7tiagdcn7crxxxxx",
//	"credentialId": "cred_01ko8mot7ic0ltskmashfa1oxxxxx",
//	"status": "enabled",
//	"credentialIdentifier": "default_model",
//	"credentialName": "default-model",
//	"credentialScenarioLabel": "llm",
//	"credentialType": "api_key",
//	"credentialCreationType": "user_custom",
//	"credentialContent": {
//	  "apiKeyContent": {
//	    "apiKey": "sk-001"
//	  }
//	},
//	"createTime": 1770281757979,
//	"updateTime": 1770281757979
type Credential struct {
	InstanceId              string            `json:"instanceId"`
	CredentialId            string            `json:"credentialId"`
	Status                  string            `json:"status"`
	CredentialIdentifier    string            `json:"credentialIdentifier"`
	CredentialName          string            `json:"credentialName"`
	CredentialScenarioLabel string            `json:"credentialScenarioLabel"`
	CredentialType          string            `json:"credentialType"`
	CredentialCreationType  string            `json:"credentialCreationType"`
	CreateTime              int64             `json:"createTime"`
	UpdateTime              int64             `json:"updateTime"`
	CredentialContent       CredentialContent `json:"credentialContent"`
}

type CredentialContent struct {
	ApiKeyContent      *ApiKeyContent      `json:"apiKeyContent"`
	OauthClientContent *OauthClientContent `json:"oauthClientContent"`
}

type OauthClientContent struct {
	ClientId     string `json:"clientId"`
	ClientSecret string `json:"clientSecret"`
}

type ApiKeyContent struct {
	ApiKey string `json:"apiKey"`
}

type CredentialOptions struct {
	Endpoint  string
	AccessKey string
}

type CreateCredentialRequest struct {
	CredentialIdentifier    string                                    `json:"credentialIdentifier"`
	CredentialName          string                                    `json:"credentialName"`
	CredentialScenarioLabel string                                    `json:"credentialScenarioLabel"`
	CredentialType          string                                    `json:"credentialType"`
	CredentialContent       *CreateCredentialRequestCredentialContent `json:"credentialContent"`
	Description             string                                    `json:"description"`
	CredentialExternalId    string                                    `json:"credentialExternalId"`
	ReturnCiphertext        bool                                      `json:"returnCiphertext,omitempty"`
}

type CreateCredentialResponse struct {
	CredentialIdentifier string `json:"credentialIdentifier"`
	CredentialCiphertext string `json:"credentialCiphertext"`
}

type DecryptCredentialRequest struct {
	CredentialIdentifier string `json:"credentialIdentifier"`
	CredentialCiphertext string `json:"credentialCiphertext"`
}

type DecryptCredentialResponse struct {
	CredentialIdentifier string `json:"credentialIdentifier"`
	CredentialPlaintext  string `json:"credentialPlaintext"`
}

type CreateCredentialRequestCredentialContent struct {
	ApiKeyContent *CreateCredentialRequestCredentialContentApiKeyContent `json:"apiKeyContent"`
}

type CreateCredentialRequestCredentialContentApiKeyContent struct {
	ApiKey string `json:"apiKey"`
}

func CreateCredentialApiKey(credentialIdentifier, credentialDisplayName, credentialScenarioLabel, credentialValue, credentialDescription, credentialExternalId string, returnCiphertext bool, options *CredentialOptions) (*CreateCredentialResponse, error) {
	client := utils.BuildHttpClient()
	endpoint := options.Endpoint
	headers := map[string]string{
		"Content-Type":  "application/json",
		"Authorization": "Bearer " + options.AccessKey,
	}
	request := &CreateCredentialRequest{
		CredentialIdentifier:    credentialIdentifier,
		CredentialName:          credentialDisplayName,
		CredentialScenarioLabel: credentialScenarioLabel,
		CredentialType:          "api_key",
		CredentialContent: &CreateCredentialRequestCredentialContent{
			ApiKeyContent: &CreateCredentialRequestCredentialContentApiKeyContent{
				ApiKey: credentialValue,
			},
		},
		Description:          credentialDescription,
		CredentialExternalId: credentialExternalId,
		ReturnCiphertext:     returnCiphertext,
	}
	requestBytes, err := json.Marshal(request)
	if err != nil {
		return nil, errors.Wrap(err, "failed to marshal request")
	}
	responseJson, err := utils.FetchWithBodyAsString(client, utils.HttpMethodPost, endpoint, headers, requestBytes)
	if err != nil {
		return nil, errors.Wrapf(err,
			"Create credential failed, endpoint: %s, credential identifier: %s", options.Endpoint, credentialIdentifier)
	}
	idaaslog.Unsafe.PrintfLn("Create credential completed for: %s", credentialIdentifier)

	if returnCiphertext {
		var response CreateCredentialResponse
		err = json.Unmarshal([]byte(responseJson), &response)
		if err != nil {
			return nil, errors.Wrap(err, "failed to unmarshal create credential response")
		}
		return &response, nil
	}
	return nil, nil
}

func DecryptCredential(credentialIdentifier, credentialCiphertext string, options *CredentialOptions) (*DecryptCredentialResponse, error) {
	client := utils.BuildHttpClient()
	endpoint := options.Endpoint
	headers := map[string]string{
		"Content-Type":  "application/json",
		"Authorization": "Bearer " + options.AccessKey,
	}
	request := &DecryptCredentialRequest{
		CredentialIdentifier: credentialIdentifier,
		CredentialCiphertext: credentialCiphertext,
	}
	requestBytes, err := json.Marshal(request)
	if err != nil {
		return nil, errors.Wrap(err, "failed to marshal decrypt request")
	}
	responseJson, err := utils.FetchWithBodyAsString(client, utils.HttpMethodPost, endpoint, headers, requestBytes)
	if err != nil {
		return nil, errors.Wrapf(err,
			"Decrypt credential failed, endpoint: %s, credential identifier: %s", options.Endpoint, credentialIdentifier)
	}
	idaaslog.Unsafe.PrintfLn("Decrypt credential completed for: %s", credentialIdentifier)

	var response DecryptCredentialResponse
	err = json.Unmarshal([]byte(responseJson), &response)
	if err != nil {
		return nil, errors.Wrap(err, "failed to unmarshal decrypt credential response")
	}
	return &response, nil
}

func FetchCredential(credentialIdentifier string, options *CredentialOptions) (*Credential, error) {
	client := utils.BuildHttpClient()
	endpoint := utils.NewUrlBuilder(options.Endpoint)
	endpoint.AddQuery("credentialIdentifier", credentialIdentifier)
	headers := map[string]string{
		"Authorization": "Bearer " + options.AccessKey,
	}
	credentialJson, err := utils.FetchAsString(client, utils.HttpMethodGet, endpoint.BuildUrl(), headers)
	if err != nil {
		if strings.Contains(err.Error(), "credential_not_found") {
			return nil, nil
		}
		return nil, errors.Wrapf(err,
			"Fetch credential failed, endpoint: %s, credential identifier: %s", options.Endpoint, credentialIdentifier)
	}
	idaaslog.Unsafe.PrintfLn("Fetch credential: %s", credentialJson)

	var credential Credential
	err = json.Unmarshal([]byte(credentialJson), &credential)
	if err != nil {
		return nil, errors.Wrapf(err, "Unmarshal credential failed")
	}

	return &credential, nil
}

func (c *Credential) Marshal() (string, error) {
	if c == nil {
		return "null", nil
	}
	b, err := json.Marshal(c)
	if err != nil {
		return "", errors.Wrap(err, "marshal credential failed")
	}
	return string(b), nil
}

type FetchCredentialWithConfigOptions struct {
	ForceNew bool
}

// FetchCredentialWithConfig fetches a static credential using a config.CredentialConfig
// (Developer API obtain endpoint + access_token). The access token provider is forced to access_token.
func FetchCredentialWithConfig(profile string, credentialConfig *config.CredentialConfig,
	options *FetchCredentialWithConfigOptions) (*Credential, error) {
	if credentialConfig == nil {
		return nil, errors.New("credential config is required")
	}
	if credentialConfig.AccessTokenProvider == nil {
		return nil, errors.New("access_token_provider is required")
	}
	if credentialConfig.CredentialIdentifier == "" {
		return nil, errors.New("credential_identifier is required")
	}
	endpoint, err := credentialConfig.GetObtainCredentialEndpoint()
	if err != nil {
		return nil, err
	}
	// MUST be an access token for Developer API
	credentialConfig.AccessTokenProvider.TokenType = oidc.TokenAccessToken
	accessToken, err := idp.FetchOidcToken(profile, credentialConfig.AccessTokenProvider, &idp.FetchOidcTokenOptions{
		ForceNew: options.ForceNew,
		CacheKey: credentialConfig.AccessTokenProvider.GetCacheKey(),
	})
	if err != nil {
		return nil, errors.Wrap(err, "fetch access token failed")
	}
	return FetchCredential(credentialConfig.CredentialIdentifier, &CredentialOptions{
		Endpoint:  endpoint,
		AccessKey: accessToken,
	})
}
