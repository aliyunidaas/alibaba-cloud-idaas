package version

import (
	"fmt"
	"strings"

	"github.com/aliyunidaas/alibaba-cloud-idaas/constants"
	"github.com/urfave/cli/v2"
)

var (
	Version string = "none"
)

func BuildCommand() *cli.Command {
	var flags []cli.Flag
	return &cli.Command{
		Name:  "version",
		Usage: "Version",
		Flags: flags,
		Action: func(context *cli.Context) error {
			return version()
		},
	}
}

func version() error {
	fmt.Printf("Version: %s\n", GetVersion())
	return nil
}

func GetVersion() string {
	if Version == "" || Version == "none" {
		return constants.AlibabaCloudIdaasCliVersion
	} else {
		return Version
	}
}

func IsPreRelease() bool {
	return strings.Contains(Version, "-pre-release")
}
