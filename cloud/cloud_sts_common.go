package cloud

import (
	"time"

	"github.com/aliyunidaas/alibaba-cloud-idaas/cloud/alibaba_cloud"
	"github.com/aliyunidaas/alibaba-cloud-idaas/cloud/aws"
	"github.com/aliyunidaas/alibaba-cloud-idaas/cloud/cloud_account"
	"github.com/aliyunidaas/alibaba-cloud-idaas/idaaslog"
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
