package aws

import (
	"context"
	"fmt"
	"time"

	"github.com/aliyunidaas/alibaba-cloud-idaas/cloud/cloud_common"
	"github.com/aliyunidaas/alibaba-cloud-idaas/config"
	"github.com/aliyunidaas/alibaba-cloud-idaas/constants"
	"github.com/aliyunidaas/alibaba-cloud-idaas/idaaslog"
	"github.com/aliyunidaas/alibaba-cloud-idaas/idp"
	"github.com/aliyunidaas/alibaba-cloud-idaas/utils"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/aws/retry"
	"github.com/aws/aws-sdk-go-v2/service/sts"
	"github.com/pkg/errors"
)

type FetchAwsStsWithOidcConfigOptions struct {
	ForceNew           bool
	ForceNewCloudToken bool
}

type FetchAwsStsWithOidcOptions struct {
	Region          string
	RoleArn         string
	DurationSeconds int32
	RoleSessionName string
	FetchOidcToken  func() (string, error)
	ForceNew        bool
}

func FetchAwsStsWithOidcConfig(profile string, awsCloudStsConfig *config.AwsCloudStsConfig,
	configOptions *FetchAwsStsWithOidcConfigOptions) (
	*AwsStsToken, error) {

	if awsCloudStsConfig.OidcTokenProvider == nil {
		return nil, errors.New("OidcTokenProvider is required")
	}
	options := &FetchAwsStsWithOidcOptions{
		Region:          awsCloudStsConfig.Region,
		RoleArn:         awsCloudStsConfig.RoleArn,
		RoleSessionName: awsCloudStsConfig.RoleSessionName,
		DurationSeconds: awsCloudStsConfig.DurationSeconds,
		FetchOidcToken: func() (string, error) {
			fetchOidcTokenOptions := &idp.FetchOidcTokenOptions{
				ForceNew: configOptions.ForceNew,
				CacheKey: awsCloudStsConfig.OidcTokenProvider.GetCacheKey(),
			}
			return idp.FetchOidcToken(profile, awsCloudStsConfig.OidcTokenProvider, fetchOidcTokenOptions)
		},
		ForceNew: configOptions.ForceNew || configOptions.ForceNewCloudToken,
	}
	return FetchStsWithOidc(profile, awsCloudStsConfig, options)
}

func FetchStsWithOidc(profile string, awsCloudStsConfig *config.AwsCloudStsConfig, options *FetchAwsStsWithOidcOptions) (*AwsStsToken, error) {
	digest := awsCloudStsConfig.Digest()
	readCacheFileOptions := &utils.ReadCacheOptions{
		Context: map[string]interface{}{
			"profile": profile,
			"digest":  digest,
			"config":  awsCloudStsConfig,
		},
		FetchContent: func() (int, string, error) {
			return fetchContent(options)
		},
		IsContentExpiringOrExpired: func(s *utils.StringWithTime) bool {
			return isContentExpiringOrExpired(s)
		},
		IsContentExpired: func(s *utils.StringWithTime) bool {
			return isContentExpired(s)
		},
		ForceNew: options.ForceNew,
	}

	cacheKey := fmt.Sprintf("%s_%s", profile, digest[0:32])
	idaaslog.Debug.PrintfLn("Cache key: %s %s", constants.CategoryCloudToken, cacheKey)
	stsTokenStr, err := utils.ReadCacheFileWithEncryptionCallback(
		constants.CategoryCloudToken, cacheKey, readCacheFileOptions)
	if err != nil {
		idaaslog.Error.PrintfLn("Error fetch cloud_token token with OIDC: %v", err)
		return nil, err
	}
	return UnmarshalStsToken(stsTokenStr)
}

func fetchContent(options *FetchAwsStsWithOidcOptions) (int, string, error) {
	client, err := createAwsStsClient(options.Region)
	if err != nil {
		idaaslog.Error.PrintfLn("Error creating aws sts client: %v", err)
		return 600, "", err
	}
	oidcToken, err := options.FetchOidcToken()
	if err != nil {
		idaaslog.Error.PrintfLn("Error fetching oidc token: %v", err)
		return 600, "", err
	}
	stsResponse, err := assumeRoleWithWebIdentity(client, oidcToken, options)
	if err != nil {
		idaaslog.Error.PrintfLn("Error assuming role: %v", err)
		return 600, "", err
	}
	credentials := stsResponse.Credentials
	awsStsToken := &AwsStsToken{
		Version:         1,
		AccessKeyId:     *credentials.AccessKeyId,
		SecretAccessKey: *credentials.SecretAccessKey,
		SessionToken:    *credentials.SessionToken,
		Expiration:      *credentials.Expiration,
	}
	stsTokenJson, err := awsStsToken.Marshal()
	if err != nil {
		idaaslog.Error.PrintfLn("Error marshaling sts aws token: %v", err)
		return 600, "", err
	}
	return 200, stsTokenJson, nil
}

func isContentExpiringOrExpired(s *utils.StringWithTime) bool {
	stsToken, err := UnmarshalStsToken(s.Content)
	if err != nil {
		return true
	}
	valid := stsToken.IsValidAtLeastThreshold(20 * time.Minute)
	idaaslog.Debug.PrintfLn("Check AWS STS is expiring or expired: %s", !valid)
	return !valid
}

func isContentExpired(s *utils.StringWithTime) bool {
	stsToken, err := UnmarshalStsToken(s.Content)
	if err != nil {
		return true
	}
	valid := stsToken.IsValidAtLeastThreshold(3 * time.Minute)
	idaaslog.Debug.PrintfLn("Check AWS STS is expired: %s", !valid)
	return !valid
}

func assumeRoleWithWebIdentity(client *sts.Client, oidcToken string, options *FetchAwsStsWithOidcOptions) (
	*sts.AssumeRoleWithWebIdentityOutput, error) {

	var roleSessionName string
	if options.RoleSessionName != "" {
		roleSessionName = options.RoleSessionName
	} else {
		roleSessionName = cloud_common.GenerateRoleSessionName(oidcToken)
		idaaslog.Info.PrintfLn(
			"Assume role session name not specified, use role session name %s", roleSessionName)
	}
	idaaslog.Debug.PrintfLn("Assume role, RoleArn: %s, RoleSessionName: %s",
		options.RoleArn, options.RoleSessionName)
	assumeRoleWithWebIdentityInput := &sts.AssumeRoleWithWebIdentityInput{
		RoleArn:          aws.String(options.RoleArn),
		RoleSessionName:  aws.String(roleSessionName),
		WebIdentityToken: aws.String(oidcToken),
	}
	if options.DurationSeconds > 0 {
		assumeRoleWithWebIdentityInput.DurationSeconds = aws.Int32(options.DurationSeconds)
	}
	idaaslog.Unsafe.PrintfLn("Assume role with web identity input: %+v, OIDC Token: %s",
		assumeRoleWithWebIdentityInput, oidcToken)
	stsResponse, err := client.AssumeRoleWithWebIdentity(context.TODO(), assumeRoleWithWebIdentityInput)
	if err != nil {
		idaaslog.Error.PrintfLn("Error assume role with OIDC: %v", err)
	}
	return stsResponse, err
}

func createAwsStsClient(region string) (*sts.Client, error) {
	if region == "" {
		return nil, errors.New("no region specified")
	}
	cfg := aws.Config{
		Region: region,
		Retryer: func() aws.Retryer {
			return retry.AddWithMaxAttempts(retry.NewStandard(), 3)
		},
	}
	client := sts.NewFromConfig(cfg)
	return client, nil
}
