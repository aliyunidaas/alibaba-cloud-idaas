package alibaba_cloud

import (
	"encoding/json"
	"time"

	"github.com/aliyunidaas/alibaba-cloud-idaas/idaaslog"
	"github.com/pkg/errors"
)

const (
	FormatAliyuncli = "aliyuncli"
	FormatOssutilv2 = "ossutilv2"
)

type StsToken struct {
	Mode            string `json:"mode"`
	AccessKeyId     string `json:"access_key_id"`
	AccessKeySecret string `json:"access_key_secret"`
	StsToken        string `json:"sts_token"`
	Expiration      string `json:"expiration"`
}

// StsTokenOssutilv2
// https://help.aliyun.com/zh/oss/developer-reference/configure-ossutil2
type StsTokenOssutilv2 struct {
	AccessKeyId     string `json:"AccessKeyId"`
	AccessKeySecret string `json:"AccessKeySecret"`
	StsToken        string `json:"SecurityToken"`
	Expiration      string `json:"Expiration"`
}

// StsTokenCredentialsUri
// https://github.com/aliyun/aliyun-cli/blob/master/config/profile.go
// com.aliyun.credentials.provider.URLCredentialProvider
// com.aliyun.oss.common.auth.CustomSessionCredentialsFetcher#CustomSessionCredentialsFetcher
type StsTokenCredentialsUri struct {
	Code            string `json:"Code"`
	StatusCode      string `json:"StatusCode"`
	AccessKeyId     string `json:"AccessKeyId"`
	AccessKeySecret string `json:"AccessKeySecret"`
	StsToken        string `json:"SecurityToken"`
	Expiration      string `json:"Expiration"`
}

func (t *StsToken) ConvertToOssutilv2() *StsTokenOssutilv2 {
	stsTokenOssutilv2 := &StsTokenOssutilv2{
		AccessKeyId:     t.AccessKeyId,
		AccessKeySecret: t.AccessKeySecret,
		StsToken:        t.StsToken,
		Expiration:      t.Expiration,
	}
	return stsTokenOssutilv2
}

func (t *StsToken) ConvertToCredentialsUri() *StsTokenCredentialsUri {
	stsTokenCredentialsUri := &StsTokenCredentialsUri{
		Code:            "Success",
		StatusCode:      "200",
		AccessKeyId:     t.AccessKeyId,
		AccessKeySecret: t.AccessKeySecret,
		StsToken:        t.StsToken,
		Expiration:      t.Expiration,
	}
	return stsTokenCredentialsUri
}

func (t *StsToken) Marshal() (string, error) {
	if t == nil {
		return "null", nil
	}
	tokenBytes, err := json.Marshal(t)
	if err != nil {
		return "", errors.Wrap(err, "marshal sts token failed")
	}
	return string(tokenBytes), nil
}

func (t *StsToken) MarshalWithFormat(format string) (string, error) {
	var token any
	if format == "" || format == FormatAliyuncli {
		token = t
	} else if format == FormatOssutilv2 {
		token = t.ConvertToOssutilv2()
	} else {
		return "", errors.New("unknown format " + format)
	}
	tokenBytes, err := json.Marshal(token)
	if err != nil {
		return "", errors.Wrap(err, "marshal sts token failed")
	}
	return string(tokenBytes), nil
}

func UnmarshalStsToken(token string) (*StsToken, error) {
	var stsToken StsToken
	err := json.Unmarshal([]byte(token), &stsToken)
	if err != nil {
		return nil, errors.Wrapf(err, "unmarshal sts token: %s failed", token)
	}
	return &stsToken, nil
}

func (t *StsToken) IsValidAtLeastThreshold(thresholdDuration time.Duration) bool {
	idaaslog.Debug.PrintfLn("Check is valid, expiration: %s, threshold: %d ms",
		t.Expiration, thresholdDuration.Milliseconds())
	expiration, err := time.Parse(time.RFC3339Nano, t.Expiration)
	if err != nil {
		idaaslog.Error.PrintfLn("Error parsing expiration: %s", t.Expiration)
		return false
	}
	valid := time.Until(expiration) > thresholdDuration
	idaaslog.Info.PrintfLn("Check is valid: %s", valid)
	return valid
}
