package cloud

import (
	"encoding/json"
	"time"

	"github.com/aliyunidaas/alibaba-cloud-idaas/cloud/alibaba_cloud"
	"github.com/aliyunidaas/alibaba-cloud-idaas/cloud/aws"
	"github.com/aliyunidaas/alibaba-cloud-idaas/cloud/cloud_account"
	"github.com/aliyunidaas/alibaba-cloud-idaas/idaaslog"
	"github.com/pkg/errors"
)

func ConvertCloudAccountTokenAlibabaCloudStsTokenToAlibabaStsToken(t *cloud_account.CloudAccountTokenAlibabaCloudStsToken) *alibaba_cloud.StsToken {
	if t == nil {
		return nil
	}
	return &alibaba_cloud.StsToken{
		Mode:            "StsToken",
		AccessKeyId:     t.AccessKeyId,
		AccessKeySecret: t.AccessKeySecret,
		StsToken:        t.StsToken,
		Expiration:      t.Expiration,
	}
}

// TencentCloudCliCredential is the JSON format written to
// ~/.tccli/<profile>.credential for the TencentCloud CLI (tccli).
// tccli has no external credential process (unlike aliyun-cli/aws-cli),
// so this file is refreshed on every fetch-token --output.
type TencentCloudCliCredential struct {
	SecretId  string `json:"secretId"`
	SecretKey string `json:"secretKey"`
	Token     string `json:"token"`
}

// MarshalTencentCloudCliCredential converts a TencentCloud STS token to the
// tccli credential JSON (the only consumer format, hence type-driven output).
func MarshalTencentCloudCliCredential(t *cloud_account.CloudAccountTokenTencentCloudStsToken) (string, error) {
	if t == nil {
		return "", errors.New("nil TencentCloud STS token")
	}
	credential := &TencentCloudCliCredential{
		SecretId:  t.SecretId,
		SecretKey: t.SecretKey,
		Token:     t.SessionToken,
	}
	data, err := json.Marshal(credential)
	if err != nil {
		return "", errors.Wrap(err, "marshal TencentCloud cli credential failed")
	}
	return string(data), nil
}

func ConvertCloudAccountTokenAwsStsTokenToAwsStsToken(t *cloud_account.CloudAccountTokenAwsStsToken) *aws.AwsStsToken {
	if t == nil {
		return nil
	}
	expiration, err := time.Parse(time.RFC3339, t.Expiration)
	if err != nil {
		idaaslog.Error.PrintfLn("[SHOULD NOT HAPPEN] Parse expiration %s error: %v", t.Expiration, err)
		// should not happen, use 10 mins later
		now := time.Now()
		expiration = now.Add(10 * time.Minute)
	}
	return &aws.AwsStsToken{
		Version:         1,
		AccessKeyId:     t.AccessKeyId,
		SecretAccessKey: t.SecretAccessKey,
		SessionToken:    t.SessionToken,
		Expiration:      expiration,
	}
}
