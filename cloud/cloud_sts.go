package cloud

import (
	"fmt"
	"strings"

	"github.com/aliyunidaas/alibaba-cloud-idaas/cloud/alibaba_cloud"
	"github.com/aliyunidaas/alibaba-cloud-idaas/cloud/aws"
	"github.com/aliyunidaas/alibaba-cloud-idaas/cloud/oidc"
	"github.com/aliyunidaas/alibaba-cloud-idaas/config"
	"github.com/pkg/errors"
)

type FetchCloudStsOptions struct {
	ForceNew           bool
	FetchOidcTokenType oidc.FetchOidcTokenType
}

func FetchCloudStsFromDefaultConfig(profile string, options *FetchCloudStsOptions) (any, *config.CloudStsConfig, error) {
	profile, cloudStsConfig, err := config.FindProfile(profile)
	if err != nil {
		return nil, cloudStsConfig, fmt.Errorf("find profie `%s` error: %s", profile, err)
	}
	sts, err := FetchCloudSts(profile, cloudStsConfig, options)
	if err != nil {
		return nil, cloudStsConfig, err
	}
	return sts, cloudStsConfig, nil
}

func FetchCloudSts(profile string, cloudStsConfig *config.CloudStsConfig, options *FetchCloudStsOptions) (any, error) {
	hasAlibabaCloud := cloudStsConfig.AlibabaCloud != nil
	hasAws := cloudStsConfig.Aws != nil
	hasOidcToken := cloudStsConfig.OidcToken != nil

	var clouds []string
	if hasAlibabaCloud {
		clouds = append(clouds, "AlibabaCloud")
	}
	if hasAws {
		clouds = append(clouds, "Aws")
	}
	if hasOidcToken {
		clouds = append(clouds, "OidcToken")
	}

	if len(clouds) > 1 {
		return nil, fmt.Errorf("multiple counds: %s found for profile: %s",
			strings.Join(clouds, ", "), profile)
	}

	if hasAlibabaCloud {
		stsOptions := &alibaba_cloud.FetchStsWithOidcConfigOptions{
			ForceNew: options.ForceNew,
		}
		return alibaba_cloud.FetchStsWithOidcConfig(profile, cloudStsConfig.AlibabaCloud, stsOptions)
	}
	if hasAws {
		awsStsOptions := &aws.FetchAwsStsWithOidcConfigOptions{
			ForceNew: options.ForceNew,
		}
		return aws.FetchAwsStsWithOidcConfig(profile, cloudStsConfig.Aws, awsStsOptions)
	}
	if hasOidcToken {
		oidcTokenConfigOptions := &oidc.FetchOidcTokenConfigOptions{
			ForceNew:       options.ForceNew,
			FetchTokenType: options.FetchOidcTokenType,
		}
		return oidc.FetchOidcToken(profile, cloudStsConfig.OidcToken, oidcTokenConfigOptions)
	}
	return nil, errors.New("no cloud provider is set")
}
