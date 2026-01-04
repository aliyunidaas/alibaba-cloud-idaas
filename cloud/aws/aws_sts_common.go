package aws

import (
	"encoding/json"
	"time"

	"github.com/aliyunidaas/alibaba-cloud-idaas/idaaslog"
	"github.com/pkg/errors"
)

type AwsStsToken struct {
	Version         int       `json:"Version"`
	AccessKeyId     string    `json:"AccessKeyId"`
	SecretAccessKey string    `json:"SecretAccessKey"`
	SessionToken    string    `json:"SessionToken"`
	Expiration      time.Time `json:"Expiration"`
}

func (t *AwsStsToken) Marshal() (string, error) {
	if t == nil {
		return "null", nil
	}
	tokenBytes, err := json.Marshal(t)
	if err != nil {
		return "", errors.Wrap(err, "marshal aws sts token failed")
	}
	return string(tokenBytes), nil
}

func UnmarshalStsToken(token string) (*AwsStsToken, error) {
	var awsStsToken AwsStsToken
	err := json.Unmarshal([]byte(token), &awsStsToken)
	if err != nil {
		return nil, errors.Wrapf(err, "unmarshal aws sts token: %s failed", token)
	}
	return &awsStsToken, nil
}

func (t *AwsStsToken) IsValidAtLeastThreshold(thresholdDuration time.Duration) bool {
	idaaslog.Debug.PrintfLn("Check is valid, expiration: %s, threshold: %d ms",
		t.Expiration, thresholdDuration.Milliseconds())
	valid := time.Until(t.Expiration) > thresholdDuration
	idaaslog.Info.PrintfLn("Check is valid: %s", valid)
	return valid
}
