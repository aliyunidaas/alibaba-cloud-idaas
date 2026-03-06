package show_profile

import (
	"fmt"
	"sort"
	"strconv"
	"strings"

	"github.com/aliyunidaas/alibaba-cloud-idaas/config"
	"github.com/aliyunidaas/alibaba-cloud-idaas/utils"
	"github.com/urfave/cli/v2"
)

var (
	stringFlagConfig = &cli.StringFlag{
		Name:    "config",
		Aliases: []string{"c"},
		Usage:   "IDaaS Config",
	}
	stringFlagProfileFilter = &cli.StringFlag{
		Name:    "profile-filter",
		Aliases: []string{"p"},
		Usage:   "IDaaS Profile filter",
	}
	boolFlagNoColor = &cli.BoolFlag{
		Name:  "no-color",
		Usage: "Output without color",
	}
)

func BuildCommand() *cli.Command {
	flags := []cli.Flag{
		stringFlagConfig,
		stringFlagProfileFilter,
		boolFlagNoColor,
	}
	return &cli.Command{
		Name:  "show-profiles",
		Usage: "Show profiles",
		Flags: flags,
		Action: func(context *cli.Context) error {
			configFilename := context.String("config")
			color := !context.Bool("no-color")
			profileFilter := context.String("profile-filter")
			return showProfiles(configFilename, profileFilter, color)
		},
	}
}

func showProfiles(configFilename, profileFilter string, color bool) error {
	cloudCredentialConfig, err := config.LoadCloudCredentialConfig(configFilename)
	if err != nil {
		return err
	}
	var keys []string
	for k := range cloudCredentialConfig.Profile {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	filterSkipCount := 0
	for _, name := range keys {
		if profileFilter != "" && !strings.Contains(name, profileFilter) {
			filterSkipCount++
			continue
		}
		profile := cloudCredentialConfig.Profile[name]
		comment := ""
		if profile.Comment != "" {
			comment = fmt.Sprintf(" , with comment: %s", utils.Under(profile.Comment, color))
		}
		fmt.Printf("Profile: %s%s\n", utils.Bold(utils.Blue(utils.Under(name, color), color), color), comment)

		showAlibabaCloud(color, profile)
		showAws(color, profile)
		showOidc(color, profile)
		showCloudAccount(color, profile)
		showAgent(color, profile)

		println()
	}

	if filterSkipCount == 0 {
		fmt.Printf("Total found: %d profile(s)\n\n", len(keys))
	} else {
		fmt.Printf("Total found: %d profile(s), filter skipped: %d, displayed: %d\n\n",
			len(keys), filterSkipCount, len(keys)-filterSkipCount)
	}

	return nil
}

func showAlibabaCloud(color bool, profile *config.CloudStsConfig) {
	if profile.AlibabaCloud != nil {
		alibabaCloud := profile.AlibabaCloud
		fmt.Printf(" %s: %s\n", pad("OidcProviderArn"), utils.Green(alibabaCloud.OidcProviderArn, color))
		fmt.Printf(" %s: %s\n", pad("RoleArn"), utils.Green(alibabaCloud.RoleArn, color))
		if alibabaCloud.DurationSeconds > 0 {
			fmt.Printf("  %s: %s seconds\n", pad("DurationSeconds"), utils.Green(fmt.Sprintf("%d", alibabaCloud.DurationSeconds), color))
		}
		if alibabaCloud.RoleSessionName != "" {
			fmt.Printf("  %s: %s\n", pad("RoleSessionName"), utils.Green(alibabaCloud.RoleSessionName, color))
		}

		oidcTokenProvider := alibabaCloud.OidcTokenProvider
		showOidcTokenProvider(color, oidcTokenProvider)
	}
}

func showAws(color bool, profile *config.CloudStsConfig) {
	if profile.Aws != nil {
		aws := profile.Aws
		fmt.Printf(" %s: %s\n", pad("Region"), utils.Green(aws.Region, color))
		fmt.Printf(" %s: %s\n", pad("RoleArn"), utils.Green(aws.RoleArn, color))
		if aws.DurationSeconds > 0 {
			fmt.Printf("  %s: %s seconds\n", pad("DurationSeconds"), utils.Green(fmt.Sprintf("%d", aws.DurationSeconds), color))
		}
		if aws.RoleSessionName != "" {
			fmt.Printf("  %s: %s\n", pad("RoleSessionName"), utils.Green(aws.RoleSessionName, color))
		}

		oidcTokenProvider := aws.OidcTokenProvider
		showOidcTokenProvider(color, oidcTokenProvider)
	}
}

func showOidc(color bool, profile *config.CloudStsConfig) {
	if profile.OidcToken != nil {
		oidcToken := profile.OidcToken
		showOidcTokenProvider(color, oidcToken)
	}
}

func showOidcTokenProvider(color bool, oidcTokenProvider *config.OidcTokenProviderConfig) {
	if oidcTokenProvider != nil {
		deviceCode := oidcTokenProvider.OidcTokenProviderDeviceCode
		showDeviceCode(color, deviceCode)

		clientCredentials := oidcTokenProvider.OidcTokenProviderClientCredentials
		showClientCredentials(color, clientCredentials)

		openApi := oidcTokenProvider.OpenApi
		showOpenApi(color, openApi)
	}
}

func showCloudAccount(color bool, profile *config.CloudStsConfig) {
	if profile.CloudAccount != nil {
		cloudAccount := profile.CloudAccount
		if cloudAccount.CloudAccountRegion != "" {
			fmt.Printf(" %s: %s\n", pad("CloudAccountRegion"), utils.Green(cloudAccount.CloudAccountRegion, color))
		}
		if cloudAccount.GetInstanceId() != "" {
			fmt.Printf(" %s: %s\n", pad("InstanceId"), utils.Green(cloudAccount.GetInstanceId(), color))
		}
		if cloudAccount.GetEndpoint() != "" {
			fmt.Printf(" %s: %s\n", pad("Endpoint"), utils.Green(cloudAccount.GetEndpoint(), color))
		}
		fmt.Printf(" %s: %s\n", pad("CloudAccountRoleExternalId"), utils.Green(cloudAccount.CloudAccountRoleExternalId, color))

		accessTokenProvider := cloudAccount.AccessTokenProvider
		showOidcTokenProvider(color, accessTokenProvider)
	}
}

func showAgent(color bool, profile *config.CloudStsConfig) {
	if profile.Agent != nil {
		agent := profile.Agent
		if agent.InstanceId != "" {
			fmt.Printf(" %s: %s\n", pad("InstanceId"), utils.Green(agent.InstanceId, color))
		}
		if agent.DeveloperApiEndpoint != "" {
			fmt.Printf(" %s: %s\n", pad("Endpoint"), utils.Green(agent.DeveloperApiEndpoint, color))
		}

		accessTokenProvider := agent.AccessTokenProvider
		showOidcTokenProvider(color, accessTokenProvider)
	}
}

func showClientCredentials(color bool, clientCredentials *config.OidcTokenProviderClientCredentialsConfig) {
	if clientCredentials != nil {
		fmt.Printf(" %s: %s\n", pad("OIDC Token Provider"), utils.Green("Client Credentials", color))
		fmt.Printf(" - %s: %s\n", pad2("TokenEndpoint"), utils.Green(clientCredentials.TokenEndpoint, color))
		fmt.Printf(" - %s: %s\n", pad2("ClientId"), utils.Green(clientCredentials.ClientId, color))
		if clientCredentials.ClientSecret != "" {
			fmt.Printf(" - %s: %s\n", pad2("ClientSecret"), utils.Green("******", color))
		}
		fmt.Printf(" - %s: %s\n", pad2("Scope"), utils.Green(clientCredentials.Scope, color))
		clientAssertionSinger := clientCredentials.ClientAssertionSinger
		if clientAssertionSinger != nil {
			showPkcs11(color, clientAssertionSinger, "")
			showYubiKeyPiv(color, clientAssertionSinger, "")
			showExternalCommand(color, clientAssertionSinger, "")
			showKeyFile(color, clientAssertionSinger, "")
		}
		showOidcTokenConfig(color, clientCredentials)
		showPkcs7Config(color, clientCredentials)
		showPrivateCaConfig(color, clientCredentials)
	}
}

func showOpenApi(color bool, openCpiConfig *config.OpenApiConfig) {
	if openCpiConfig != nil {
		fmt.Printf(" %s: %s\n", pad("OIDC Token Provider"), utils.Green("Open API", color))
		fmt.Printf(" - %s: %s\n", pad2("InstanceId"), utils.Green(openCpiConfig.InstanceId, color))
		fmt.Printf(" - %s: %s\n", pad2("ApplicationId"), utils.Green(openCpiConfig.ApplicationId, color))
		fmt.Printf(" - %s: %s\n", pad2("Audience"), utils.Green(openCpiConfig.Audience, color))
		fmt.Printf(" - %s: %s\n", pad2("ScopeValues"), utils.Green(strings.Join(openCpiConfig.ScopeValues, ","), color))
		if openCpiConfig.OpenApiEndpoint != "" {
			fmt.Printf(" - %s: %s\n", pad2("OpenApiEndpoint"), utils.Green(openCpiConfig.OpenApiEndpoint, color))
		}
		if openCpiConfig.Type != "" {
			fmt.Printf(" - %s: %s\n", pad2("Type"), utils.Green(openCpiConfig.Type, color))
		}
		if openCpiConfig.AccessKeyId != "" {
			fmt.Printf(" - %s: %s\n", pad2("AccessKeyId"), utils.Green(openCpiConfig.AccessKeyId, color))
		}
		if openCpiConfig.AccessKeySecret != "" {
			fmt.Printf(" - %s: %s\n", pad2("AccessKeySecret"), utils.Green("******", color))
		}
		if openCpiConfig.SecurityToken != "" {
			fmt.Printf(" - %s: %s\n", pad2("SecurityToken"), utils.Green("******", color))
		}
		if openCpiConfig.OIDCProviderArn != "" {
			fmt.Printf(" - %s: %s\n", pad2("OIDCProviderArn"), utils.Green(openCpiConfig.OIDCProviderArn, color))
		}
		if openCpiConfig.OIDCTokenFilePath != "" {
			fmt.Printf(" - %s: %s\n", pad2("OIDCTokenFilePath"), utils.Green(openCpiConfig.OIDCTokenFilePath, color))
		}
		if openCpiConfig.RoleArn != "" {
			fmt.Printf(" - %s: %s\n", pad2("RoleArn"), utils.Green(openCpiConfig.RoleArn, color))
		}
		if openCpiConfig.RoleSessionName != "" {
			fmt.Printf(" - %s: %s\n", pad2("RoleSessionName"), utils.Green(openCpiConfig.RoleSessionName, color))
		}
		if openCpiConfig.RoleSessionExpiration != 0 {
			fmt.Printf(" - %s: %s\n", pad2("RoleSessionExpiration"), utils.Green(strconv.Itoa(openCpiConfig.RoleSessionExpiration), color))
		}
		if openCpiConfig.Policy != "" {
			fmt.Printf(" - %s: %s\n", pad2("Policy"), utils.Green(openCpiConfig.Policy, color))
		}
		if openCpiConfig.ExternalId != "" {
			fmt.Printf(" - %s: %s\n", pad2("ExternalId"), utils.Green(openCpiConfig.ExternalId, color))
		}
		if openCpiConfig.STSEndpoint != "" {
			fmt.Printf(" - %s: %s\n", pad2("STSEndpoint"), utils.Green(openCpiConfig.STSEndpoint, color))
		}
		if openCpiConfig.RoleName != "" {
			fmt.Printf(" - %s: %s\n", pad2("RoleName"), utils.Green(openCpiConfig.RoleName, color))
		}
		if openCpiConfig.Url != "" {
			fmt.Printf(" - %s: %s\n", pad2("Url"), utils.Green(openCpiConfig.Url, color))
		}
	}
}

func showOidcTokenConfig(color bool, clientCredentials *config.OidcTokenProviderClientCredentialsConfig) {
	oidcTokenConfig := clientCredentials.ClientAssertionOidcTokenConfig
	if oidcTokenConfig != nil {
		fmt.Printf(" - %s: %s\n", pad2("AppFedCredentialName"), utils.Green(clientCredentials.ApplicationFederatedCredentialName, color))
		fmt.Printf(" - %s: %s\n", pad2("Assertion"), utils.Green("OIDC Token", color))
		fmt.Printf("   - %s: %s\n", pad3("Provider"), utils.Green(oidcTokenConfig.Provider, color))
		fmt.Printf("   - %s: %s\n", pad3("GoogleVmIdentityUrl"), utils.Green(oidcTokenConfig.GoogleVmIdentityUrl, color))
		fmt.Printf("   - %s: %s\n", pad3("GoogleVmIdentityAud"), utils.Green(oidcTokenConfig.GoogleVmIdentityAud, color))
		fmt.Printf("   - %s: %s\n", pad3("OidcToken"), utils.Green(oidcTokenConfig.OidcToken, color))
		fmt.Printf("   - %s: %s\n", pad3("OidcTokenFile"), utils.Green(oidcTokenConfig.OidcTokenFile, color))
	}
}

func showPkcs7Config(color bool, clientCredentials *config.OidcTokenProviderClientCredentialsConfig) {
	pkcs7Config := clientCredentials.ClientAssertionPkcs7Config
	if pkcs7Config != nil {
		fmt.Printf(" - %s: %s\n", pad2("AppFedCredentialName"), utils.Green(clientCredentials.ApplicationFederatedCredentialName, color))
		fmt.Printf(" - %s: %s\n", pad2("Assertion"), utils.Green("PKCS#7", color))
		fmt.Printf("   - %s: %s\n", pad3("Provider"), utils.Green(pkcs7Config.Provider, color))
		fmt.Printf("   - %s: %s\n", pad3("AlibabaCloudMode"), utils.Green(pkcs7Config.AlibabaCloudMode, color))
		fmt.Printf("   - %s: %s\n", pad3("AlibabaCloudIdaasInstanceId"), utils.Green(pkcs7Config.AlibabaCloudIdaasInstanceId, color))
	}
}

func showPrivateCaConfig(color bool, clientCredentials *config.OidcTokenProviderClientCredentialsConfig) {
	privateCaConfig := clientCredentials.ClientAssertionPrivateCaConfig
	if privateCaConfig != nil {
		fmt.Printf(" - %s: %s\n", pad2("AppFedCredentialName"), utils.Green(clientCredentials.ApplicationFederatedCredentialName, color))
		fmt.Printf(" - %s: %s\n", pad2("Assertion"), utils.Green("Private CA", color))
		fmt.Printf("   - %s: %s\n", pad3("Certificate"), utils.Green(privateCaConfig.Certificate, color))
		fmt.Printf("   - %s: %s\n", pad3("CertificateFile"), utils.Green(privateCaConfig.CertificateFile, color))
		fmt.Printf("   - %s: %s\n", pad3("CertificateChain"), utils.Green(privateCaConfig.CertificateChain, color))
		fmt.Printf("   - %s: %s\n", pad3("CertificateChainFile"), utils.Green(privateCaConfig.CertificateChainFile, color))

		certificateKeySigner := privateCaConfig.CertificateKeySigner
		if certificateKeySigner != nil {
			showPkcs11(color, certificateKeySigner, "  ")
			showYubiKeyPiv(color, certificateKeySigner, "  ")
			showExternalCommand(color, certificateKeySigner, "  ")
			showKeyFile(color, certificateKeySigner, "  ")
		}
	}
}

func showKeyFile(color bool, clientAssertionSinger *config.ExSingerConfig, prefix string) {
	keyFile := clientAssertionSinger.KeyFile
	if keyFile != nil {
		fmt.Printf("%s - %s: %s\n", prefix, pad2("Singer"), utils.Green("Key File", color))
		if keyFile.Key != "" {
			if strings.Contains(keyFile.Key, "ENCRYPTED") {
				fmt.Printf("%s   - %s: %s\n", prefix, pad3("Key"), utils.Green(keyFile.Key, color))
			} else {
				fmt.Printf("%s   - %s: %s\n", prefix, pad3("Key"), utils.Green("******", color))
			}
		}
		if keyFile.File != "" {
			if strings.Contains(keyFile.File, "ENCRYPTED") {
				fmt.Printf("%s   - %s: %s\n", prefix, pad3("File"), utils.Green(keyFile.File, color))
			} else {
				fmt.Printf("%s   - %s: %s\n", prefix, pad3("File"), utils.Green("******", color))
			}
		}
		if keyFile.Password != "" {
			fmt.Printf("%s   - %s: %s\n", prefix, pad3("Password"), utils.Green("******", color))
		}
	}
}

func showExternalCommand(color bool, clientAssertionSinger *config.ExSingerConfig, prefix string) {
	externalCommand := clientAssertionSinger.ExternalCommand
	if externalCommand != nil {
		fmt.Printf("%s - %s: %s\n", prefix, pad2("Singer"), utils.Green("External Command", color))
		fmt.Printf("%s   - %s: %s\n", prefix, pad3("Command"), utils.Green(externalCommand.Command, color))
		fmt.Printf("%s   - %s: %s\n", prefix, pad3("Parameter"), utils.Green(externalCommand.Parameter, color))
	}
}

func showYubiKeyPiv(color bool, clientAssertionSinger *config.ExSingerConfig, prefix string) {
	yubikeyPiv := clientAssertionSinger.YubikeyPiv
	if yubikeyPiv != nil {
		fmt.Printf("%s - %s: %s\n", prefix, pad2("Singer"), utils.Green("YubiKey PIV", color))
		fmt.Printf("%s   - %s: %s\n", prefix, pad3("Slot"), utils.Green(yubikeyPiv.Slot, color))
		if yubikeyPiv.Pin != "" {
			fmt.Printf("%s   - %s: %s\n", prefix, pad3("Pin"), utils.Green("******", color))
		}
		fmt.Printf("%s   - %s: %s\n", prefix, pad3("PinPolicy"), utils.Green(yubikeyPiv.PinPolicy, color))
	}
}

func showPkcs11(color bool, clientAssertionSinger *config.ExSingerConfig, prefix string) {
	pkcs11 := clientAssertionSinger.Pkcs11
	if pkcs11 != nil {
		fmt.Printf("%s - %s: %s\n", prefix, pad2("Singer"), utils.Green("PKCS#11", color))
		fmt.Printf("%s   - %s: %s\n", prefix, pad3("LibraryPath"), utils.Green(pkcs11.LibraryPath, color))
		fmt.Printf("%s   - %s: %s\n", prefix, pad3("TokenLabel"), utils.Green(pkcs11.TokenLabel, color))
		fmt.Printf("%s   - %s: %s\n", prefix, pad3("KeyLabel"), utils.Green(pkcs11.KeyLabel, color))
		if pkcs11.Pin != "" {
			fmt.Printf("%s   - %s: %s\n", prefix, pad3("Pin"), utils.Green("******", color))
		}
	}
}

func showDeviceCode(color bool, deviceCode *config.OidcTokenProviderDeviceCodeConfig) {
	if deviceCode != nil {
		fmt.Printf(" %s: %s\n", pad("OIDC Token Provider"), utils.Green("Device Code", color))
		fmt.Printf(" - %s: %s\n", pad2("Issuer"), utils.Green(deviceCode.Issuer, color))
		fmt.Printf(" - %s: %s\n", pad2("ClientId"), utils.Green(deviceCode.ClientId, color))
		if deviceCode.ClientSecret != "" {
			fmt.Printf(" - %s: %s\n", pad2("ClientSecret"), utils.Green("******", color))
		}
		fmt.Printf(" - %s: %s\n", pad2("Scope"), utils.Green(deviceCode.Scope, color))
		fmt.Printf(" - %s: %s\n", pad2("AutoOpenUrl"),
			utils.Green(fmt.Sprintf("%v", deviceCode.AutoOpenUrl), color))
		fmt.Printf(" - %s: %s\n", pad2("ShowQrCode"),
			utils.Green(fmt.Sprintf("%v", deviceCode.AutoOpenUrl), color))
		fmt.Printf(" - %s: %s\n", pad2("SmallQrCode"),
			utils.Green(fmt.Sprintf("%v", deviceCode.AutoOpenUrl), color))
	}
}

func pad(str string) string {
	return padWith(str, 24)
}

func pad2(str string) string {
	return padWith(str, 24-2)
}

func pad3(str string) string {
	return padWith(str, 24-2-2)
}

func padWith(str string, width int) string {
	if len(str) >= width {
		return str
	}
	return str + strings.Repeat(" ", width-len(str))
}
