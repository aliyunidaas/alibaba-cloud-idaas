package credential

import (
	"encoding/json"
	"fmt"
	"net/url"
	"strings"

	"github.com/aliyunidaas/alibaba-cloud-idaas/idaaslog"
	"github.com/aliyunidaas/alibaba-cloud-idaas/utils"
	"github.com/pkg/errors"
)

// Credential
// e.g.
//  "instanceId": "idaas_wrwsx2jca7tiagdcn7crbh5diy",
//  "credentialId": "cred_01ko8mot7ic0ltskmashfa1ohbhoo",
//  "status": "enabled",
//  "credentialIdentifier": "default_model",
//  "credentialName": "default-model",
//  "credentialScenarioLabel": "llm",
//  "credentialType": "api_key",
//  "credentialCreationType": "user_custom",
//  "credentialContent": {
//    "apiKeyContent": {
//      "apiKey": "sk-001"
//    }
//  },
//  "createTime": 1770281757979,
//  "updateTime": 1770281757979
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

func FetchCredential(fetchCredentialEndpoint, credentialIdentifier, accessToken string) (*Credential, error) {
	client := utils.BuildHttpClient()
	endpoint := fetchCredentialEndpoint
	if strings.Contains(fetchCredentialEndpoint, "?") {
		endpoint += "&"
	} else {
		endpoint += "?"
	}
	endpoint += fmt.Sprintf("credentialIdentifier=%s", url.QueryEscape(credentialIdentifier))
	headers := map[string]string{
		"Authorization": "Bearer " + accessToken,
	}
	credentialJson, err := utils.FetchAsString(client, utils.HttpMethodGet, endpoint, headers)
	if err != nil {
		if strings.Contains(err.Error(), "credential_not_found") {
			return nil, nil
		}
		return nil, errors.Wrapf(err,
			"Fetch credential failed, endpoint: %s, credential identifier: %s", fetchCredentialEndpoint, credentialIdentifier)
	}
	idaaslog.Unsafe.PrintfLn("Fetch credential: %s", credentialJson)

	var credential Credential
	err = json.Unmarshal([]byte(credentialJson), &credential)
	if err != nil {
		return nil, errors.Wrapf(err, "Unmarshal credential failed")
	}

	return &credential, nil
}
