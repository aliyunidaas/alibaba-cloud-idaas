package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/aliyunidaas/alibaba-cloud-idaas/commands/agent"
	"github.com/aliyunidaas/alibaba-cloud-idaas/commands/login"
	"github.com/aliyunidaas/alibaba-cloud-idaas/commands/logout"
	"github.com/aliyunidaas/alibaba-cloud-idaas/commands/onboard"
	"github.com/aliyunidaas/alibaba-cloud-idaas/commands/openclaw_secret"
	"github.com/aliyunidaas/alibaba-cloud-idaas/commands/qr"
	"github.com/aliyunidaas/alibaba-cloud-idaas/commands/serve"
	"github.com/aliyunidaas/alibaba-cloud-idaas/commands/show"
	"github.com/aliyunidaas/alibaba-cloud-idaas/commands/show_signer_public_key"
	"github.com/aliyunidaas/alibaba-cloud-idaas/commands/start_session"
	"github.com/aliyunidaas/alibaba-cloud-idaas/commands/status"
	"github.com/aliyunidaas/alibaba-cloud-idaas/commands/validate_jwt"
	"github.com/aliyunidaas/alibaba-cloud-idaas/utils/features"

	"github.com/aliyunidaas/alibaba-cloud-idaas/commands/clean_cache"
	"github.com/aliyunidaas/alibaba-cloud-idaas/commands/execute"
	"github.com/aliyunidaas/alibaba-cloud-idaas/commands/fetch_token"
	"github.com/aliyunidaas/alibaba-cloud-idaas/commands/show_cache"
	"github.com/aliyunidaas/alibaba-cloud-idaas/commands/show_profile"
	"github.com/aliyunidaas/alibaba-cloud-idaas/commands/show_token"
	"github.com/aliyunidaas/alibaba-cloud-idaas/commands/version"
	"github.com/aliyunidaas/alibaba-cloud-idaas/config"
	"github.com/aliyunidaas/alibaba-cloud-idaas/constants"
	"github.com/aliyunidaas/alibaba-cloud-idaas/idaaslog"
	"github.com/aliyunidaas/alibaba-cloud-idaas/utils"
	"github.com/urfave/cli/v2"
)

func main() {
	idaaslog.InitLog()
	defer idaaslog.CloseLog()

	err := innerMain()
	if err != nil {
		utils.Stderr.Fprintf("%v\n", err)
		os.Exit(-1)
	}
}

func innerMain() error {
	commands := []*cli.Command{
		fetch_token.BuildCommand(),
		onboard.BuildCommand(),
		login.BuildCommand(),
		logout.BuildCommand(),
		status.BuildCommand(),
		show.BuildCommand(),
		show_token.BuildCommand(),
		show_profile.BuildCommand(),
		version.BuildCommand(),
		clean_cache.BuildCommand(),
		execute.BuildCommand(),
		show_cache.BuildCommand(),
		show_signer_public_key.BuildCommand(),
		serve.BuildCommand(),
		qr.BuildCommand(),
		validate_jwt.BuildCommand(),
		openclaw_secret.BuildCommand(),
		agent.BuildCommand(),
	}
	if version.IsPreRelease() {
		commands = append(commands, start_session.BuildCommand())
	}
	app := &cli.App{
		Name:     "alibaba-cloud-idaas",
		Usage:    "Alibaba Cloud IDaaS command line util",
		Commands: commands,
		Action: func(context *cli.Context) error {
			printBanner()
			printConfigFileAndFeatures()
			printVerbose(context)
			if context.Args().First() != "" {
				utils.Stderr.Fprintf("\n%s\n",
					utils.Red("[ERROR] Bad sub command: "+context.Args().First(), true))
			}
			return nil
		},
	}
	if err := app.Run(os.Args); err != nil {
		idaaslog.Error.PrintfLn("%s", idaaslog.DumpError(err))
		utils.Stderr.Fprintf("%s\n", err.Error())
		os.Exit(1)
	}
	return nil
}

func printBanner() {
	logoAndVersion := "\n" +
		":::::::::::  :::::::::       :::          :::       ::::::::  \n" +
		"    :+:      :+:    :+:    :+: :+:      :+: :+:    :+:    :+: \n" +
		"    +:+      +:+    +:+   +:+   +:+    +:+   +:+   +:+        \n" +
		"    +#+      +#+    +:+  +#++:++#++:  +#++:++#++:  +#++:++#++ \n" +
		"    +#+      +#+    +#+  +#+     +#+  +#+     +#+         +#+ \n" +
		"    #+#      #+#    #+#  #+#     #+#  #+#     #+#  #+#    #+# \n" +
		"###########  #########   ###     ###  ###     ###   ########   v" + constants.Version
	println(logoAndVersion)

	println()
	fmt.Println("Alibaba Cloud IDaaS Command Line Utility, use --help for help message")
	println()
	fmt.Printf("Official product website: %s\n", constants.UrlIdaasProduct)
	fmt.Printf("Project repository: %s\n", constants.UrlAlibabaCloudIdaasRepository)
}

func printConfigFileAndFeatures() {
	println()
	configFilename, err := config.GetDefaultCloudCredentialConfigFile()
	if err != nil {
		fmt.Println("Default config location: ~/.aliyun/alibaba-cloud-idaas.json")
	} else {
		configFile, readConfigErr := config.ReadCloudCredentialConfig(configFilename)
		if readConfigErr != nil {
			fmt.Printf("Default config location: %s, %s\n",
				utils.Red(configFilename, true), utils.Red(utils.Bold(readConfigErr.Error(), true), true))
		} else if configFile == nil {
			fmt.Printf("Default config location: %s, %s\n",
				utils.Yellow(configFilename, true), utils.Bold(utils.Yellow("config not exists.", true), true))
		} else {
			fmt.Printf("Default config location: %s\n", utils.Green(configFilename, true))
		}
	}

	fmt.Printf("Features: [%s]\n", strings.Join(features.GetEnabledFeatures(), ", "))
}

func printVerbose(context *cli.Context) {
	if len(context.Args().Slice()) == 1 && context.Args().Slice()[0] == "verbose" {
		println()

		fmt.Printf("\nSupported environment variables:\n")
		fmt.Printf(" - %s  User agent when send OIDC/OAuth related requests\n", padEnv(constants.EnvUserAgent))
		fmt.Printf(" - %s  Log unsafe secure data\n", padEnv(constants.EnvUnsafeDebug))
		fmt.Printf(" - %s  Copy log to console stderr\n", padEnv(constants.EnvUnsafeConsolePrint))
		if features.IsPkcs11Enabled() {
			fmt.Printf(" - %s  PKCS#11 PIN\n", padEnv(constants.EnvPkcs11Pin))
		}
		if features.IsYubikeyEnabled() {
			fmt.Printf(" - %s  YubiKey PIV PIN\n", padEnv(constants.EnvYubiKeyPin))
		}
	}
}

func padEnv(str string) string {
	width := 40
	if width <= len(str) {
		return str
	}
	return str + strings.Repeat(" ", width-len(str))
}
