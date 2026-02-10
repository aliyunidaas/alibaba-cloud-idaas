package execute

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/aliyunidaas/alibaba-cloud-idaas/cloud"
	"github.com/aliyunidaas/alibaba-cloud-idaas/cloud/alibaba_cloud"
	"github.com/aliyunidaas/alibaba-cloud-idaas/cloud/aws"
	"github.com/aliyunidaas/alibaba-cloud-idaas/cloud/cloud_account"
	"github.com/aliyunidaas/alibaba-cloud-idaas/cloud/oidc"
	"github.com/aliyunidaas/alibaba-cloud-idaas/commands/common"
	"github.com/aliyunidaas/alibaba-cloud-idaas/config"
	"github.com/aliyunidaas/alibaba-cloud-idaas/idaaslog"
	"github.com/urfave/cli/v2"
)

var (
	stringFlagConfig = &cli.StringFlag{
		Name:    "config",
		Aliases: []string{"c"},
		Usage:   "IDaaS Config",
	}
	stringFlagProfile = &cli.StringFlag{
		Name:    "profile",
		Aliases: []string{"p"},
		Usage:   "IDaaS Profile",
	}
	stringFlagEnvRegion = &cli.StringFlag{
		Name:    "env-region",
		Aliases: []string{"R"},
		Usage:   "Set environment region",
	}
	boolFlagForceNew = &cli.BoolFlag{
		Name:    "force-new",
		Aliases: []string{"N"},
		Usage:   "Force fetch cloud token, ignore all cache",
	}
	boolFlagForceNewCloudToken = &cli.BoolFlag{
		Name:  "force-new-cloud-token",
		Usage: "Force fetch cloud token (lower cache enabled)",
	}
	boolFlagShowToken = &cli.BoolFlag{
		Name:  "show-token",
		Usage: "Show cloud STS token",
	}
)

func BuildCommand() *cli.Command {
	flags := []cli.Flag{
		stringFlagConfig,
		stringFlagProfile,
		stringFlagEnvRegion,
		boolFlagForceNew,
		boolFlagForceNewCloudToken,
		boolFlagShowToken,
	}
	return &cli.Command{
		Name:    "execute",
		Aliases: []string{"exe", "exec"},
		Usage:   "Execute command",
		Flags:   flags,
		Action: func(context *cli.Context) error {
			configFilename := context.String("config")
			profile := context.String("profile")
			envRegion := context.String("env-region")
			forceNew := context.Bool("force-new")
			forceNewCloudToken := context.Bool("force-new-cloud-token")
			showToken := context.Bool("show-token")
			args := context.Args()
			return execute(configFilename, profile, showToken, forceNew, forceNewCloudToken, envRegion, args.Slice())
		},
	}
}

func execute(configFilename, profile string, showToken, forceNew, forceNewCloudToken bool, envRegion string, args []string) error {
	options := &cloud.FetchCloudStsOptions{
		ForceNew:           forceNew,
		ForceNewCloudToken: forceNewCloudToken,
	}
	sts, cloudStsConfig, err := cloud.FetchCloudStsFromDefaultConfig(configFilename, profile, options)
	if err != nil {
		return err
	}

	if showToken {
		_ = common.ShowToken(sts, oidc.FetchDefault, false, true)
	}

	alibabaCloudSts, ok := sts.(*alibaba_cloud.StsToken)
	if ok {
		environment, err := putEnvForStsToken(alibabaCloudSts, envRegion, cloudStsConfig)
		if err != nil {
			return err
		}
		return executeCommand(args, environment)
	}
	awsStsToken, ok := sts.(*aws.AwsStsToken)
	if ok {
		environment, err := putEnvForAwsStsToken(awsStsToken, envRegion, cloudStsConfig)
		if err != nil {
			return err
		}
		return executeCommand(args, environment)
	}
	cloudAccountToken, ok := sts.(*cloud_account.CloudAccountToken)
	if ok {
		credential := cloudAccountToken.CloudAccountRoleAccessCredential
		if credential == nil {
			return fmt.Errorf("invalid Cloud Account credential")
		}
		if cloudAccountToken.IsAlibabaCloudToken() {
			alibabaCloudSts := cloud.ConvertCloudAccountTokenAlibabaCloudStsTokenToAlibabaStsToken(credential.AlibabaCloudStsToken)
			environment, err := putEnvForStsToken(alibabaCloudSts, envRegion, cloudStsConfig)
			if err != nil {
				return err
			}
			return executeCommand(args, environment)
		}

		return fmt.Errorf("unknown Cloud Account token")
	}

	return fmt.Errorf("unknown cloud STS token type")
}

func executeCommand(args, environment []string) error {
	if len(args) == 0 {
		return fmt.Errorf("no command specified")
	}
	idaaslog.Debug.PrintfLn("Exec args: %+v", args)
	idaaslog.Unsafe.PrintfLn("Env: %s", strings.Join(environment, "\n"))
	cmd := exec.Command(args[0], args[1:]...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Env = environment
	return cmd.Run()
}

// Alibaba Cloud Terraform plugin credential order:
// 静态配置 > 环境变量 > Profile 静态配置 > ECS 服务角色 > Profile ECS 服务角色 > OIDC 角色扮演 > 角色扮演
// reference: https://help.aliyun.com/zh/terraform/terraform-authentication
// reference: https://help.aliyun.com/zh/sdk/developer-reference/v2-manage-access-credentials
func putEnvForStsToken(sts *alibaba_cloud.StsToken, envRegion string, cloudStsConfig *config.CloudStsConfig) ([]string, error) {
	osEnv := os.Environ()
	osEnv = addEnvironmentsFromConfig(osEnv, cloudStsConfig)

	idaaslog.Debug.PrintfLn("Found access key ID: %s", sts.AccessKeyId)
	osEnv = append(osEnv, "ALIBABA_CLOUD_ACCESS_KEY_ID="+sts.AccessKeyId)
	osEnv = append(osEnv, "ALIBABACLOUD_ACCESS_KEY_ID="+sts.AccessKeyId)
	osEnv = append(osEnv, "ALICLOUD_ACCESS_KEY_ID="+sts.AccessKeyId)
	osEnv = append(osEnv, "ALICLOUD_ACCESS_KEY="+sts.AccessKeyId)
	osEnv = append(osEnv, "ACCESS_KEY_ID="+sts.AccessKeyId)
	osEnv = append(osEnv, "OSS_ACCESS_KEY_ID="+sts.AccessKeyId)

	osEnv = append(osEnv, "ALICLOUD_SECRET_KEY="+sts.AccessKeySecret)
	osEnv = append(osEnv, "ALIBABA_CLOUD_ACCESS_KEY_SECRET="+sts.AccessKeySecret)
	osEnv = append(osEnv, "ALIBABACLOUD_ACCESS_KEY_SECRET="+sts.AccessKeySecret)
	osEnv = append(osEnv, "ALICLOUD_ACCESS_KEY_SECRET="+sts.AccessKeySecret)
	osEnv = append(osEnv, "ACCESS_KEY_SECRET="+sts.AccessKeySecret)
	osEnv = append(osEnv, "OSS_ACCESS_KEY_SECRET="+sts.AccessKeySecret)

	osEnv = append(osEnv, "ALIBABA_CLOUD_SECURITY_TOKEN="+sts.StsToken)
	osEnv = append(osEnv, "ALIBABACLOUD_SECURITY_TOKEN="+sts.StsToken)
	osEnv = append(osEnv, "ALICLOUD_SECURITY_TOKEN="+sts.StsToken)
	osEnv = append(osEnv, "SECURITY_TOKEN="+sts.StsToken)
	osEnv = append(osEnv, "OSS_SESSION_TOKEN="+sts.StsToken)

	if envRegion != "" {
		idaaslog.Debug.PrintfLn("Set region: %s", envRegion)
		osEnv = append(osEnv, "ALICLOUD_REGION="+envRegion)
		osEnv = append(osEnv, "ALIYUN_DEFAULT_REGION="+envRegion)
		osEnv = append(osEnv, "DEFAULT_REGION="+envRegion)
		osEnv = append(osEnv, "ALIBABA_CLOUD_DEFAULT_REGION="+envRegion)
		osEnv = append(osEnv, "REGION="+envRegion)
		osEnv = append(osEnv, "OSS_REGION="+envRegion)
	}

	return osEnv, nil
}

// reference: https://docs.aws.amazon.com/cli/v1/userguide/cli-configure-envvars.html
func putEnvForAwsStsToken(sts *aws.AwsStsToken, envRegion string, cloudStsConfig *config.CloudStsConfig) ([]string, error) {
	osEnv := os.Environ()
	osEnv = addEnvironmentsFromConfig(osEnv, cloudStsConfig)

	idaaslog.Debug.PrintfLn("Found access key ID: %s", sts.AccessKeyId)
	osEnv = append(osEnv, "AWS_ACCESS_KEY_ID="+sts.AccessKeyId)

	osEnv = append(osEnv, "AWS_SECRET_ACCESS_KEY="+sts.SecretAccessKey)

	osEnv = append(osEnv, "AWS_SESSION_TOKEN="+sts.SessionToken)

	if envRegion != "" {
		idaaslog.Debug.PrintfLn("Set region: %s", envRegion)
		osEnv = append(osEnv, "AWS_DEFAULT_REGION="+envRegion)
		osEnv = append(osEnv, "AWS_REGION="+envRegion)
	}

	return osEnv, nil
}

func addEnvironmentsFromConfig(environments []string, cloudStsConfig *config.CloudStsConfig) []string {
	if cloudStsConfig.Environments != nil {
		for _, env := range cloudStsConfig.Environments {
			environments = append(environments, env)
		}
	}
	return environments
}
