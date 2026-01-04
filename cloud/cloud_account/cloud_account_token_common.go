package cloud_account

import (
	"encoding/json"
	"time"

	"github.com/aliyunidaas/alibaba-cloud-idaas/idaaslog"
	"github.com/pkg/errors"
)

type CloudAccountToken struct {
	CloudAccountId             string `json:"cloudAccountId"`
	CloudAccountRoleId         string `json:"cloudAccountRoleId"`
	CloudAccountRoleName       string `json:"cloudAccountRoleName"`
	CloudAccountRoleExternalId string `json:"cloudAccountRoleExternalId"`
	CloudAccountVendorType     string `json:"cloudAccountVendorType"`

	CloudAccountRoleAccessCredential *CloudAccountTokenCloudAccountRoleAccessCredential `json:"cloudAccountRoleAccessCredential"`
}

type CloudAccountTokenCloudAccountRoleAccessCredential struct {
	AccessCredentialExpiresAt int64 `json:"accessCredentialExpiresAt"`

	AlibabaCloudStsToken *CloudAccountTokenAlibabaCloudStsToken `json:"alibabaCloudStsToken"`
}

type CloudAccountTokenAlibabaCloudStsToken struct {
	AccessKeyId     string `json:"accessKeyId"`
	AccessKeySecret string `json:"accessKeySecret"`
	StsToken        string `json:"securityToken"`
	Expiration      string `json:"expiration"`
}

func (t *CloudAccountToken) Marshal() (string, error) {
	if t == nil {
		return "null", nil
	}
	tokenBytes, err := json.Marshal(t)
	if err != nil {
		return "", errors.Wrap(err, "marshal Cloud Account token failed")
	}
	return string(tokenBytes), nil
}

func UnmarshalCloudAccountToken(token string) (*CloudAccountToken, error) {
	var cloudAccountToken CloudAccountToken
	err := json.Unmarshal([]byte(token), &cloudAccountToken)
	if err != nil {
		return nil, errors.Wrapf(err, "unmarshal Cloud Account token: %s failed", token)
	}
	return &cloudAccountToken, nil
}

func (t *CloudAccountToken) IsValidAtLeastThreshold(thresholdDuration time.Duration) bool {
	if t.CloudAccountRoleAccessCredential == nil {
		return false
	}
	accessCredentialExpiresAt := t.CloudAccountRoleAccessCredential.AccessCredentialExpiresAt
	idaaslog.Debug.PrintfLn("Check is valid, expiration: %s, threshold: %d ms",
		accessCredentialExpiresAt, thresholdDuration.Milliseconds())
	valid := (accessCredentialExpiresAt - time.Now().Unix()) > int64(thresholdDuration.Seconds())
	idaaslog.Info.PrintfLn("Check is valid: %s", valid)
	return valid
}

func (t *CloudAccountToken) IsAlibabaCloudToken() bool {
	if t.CloudAccountRoleAccessCredential != nil {
		return t.CloudAccountRoleAccessCredential.AlibabaCloudStsToken != nil
	}
	return false
}
