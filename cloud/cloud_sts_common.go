package cloud

import (
	"github.com/aliyunidaas/alibaba-cloud-idaas/cloud/alibaba_cloud"
	"github.com/aliyunidaas/alibaba-cloud-idaas/cloud/cloud_account"
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
