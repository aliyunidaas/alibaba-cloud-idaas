package fetch_token

import (
	"fmt"
	"os"

	"github.com/aliyunidaas/alibaba-cloud-idaas/cloud"
	"github.com/aliyunidaas/alibaba-cloud-idaas/cloud/alibaba_cloud"
	"github.com/aliyunidaas/alibaba-cloud-idaas/cloud/aws"
	"github.com/aliyunidaas/alibaba-cloud-idaas/cloud/oidc"
	"github.com/aliyunidaas/alibaba-cloud-idaas/utils"
	"github.com/pkg/errors"
	"github.com/urfave/cli/v2"
)

var (
	stringFlagProfile = &cli.StringFlag{
		Name:    "profile",
		Aliases: []string{"p"},
		Usage:   "IDaaS Profile",
	}
	stringFlagFormat = &cli.StringFlag{
		Name:    "format",
		Aliases: []string{"f"},
		Usage:   "Cloud STS format, values aliyuncli(default), ossutilv2",
	}
	stringFlagOidcField = &cli.StringFlag{
		Name:  "oidc-field",
		Usage: "Fetch OIDC filed (id_token or access_token)",
	}
	stringFlagOutput = &cli.StringFlag{
		Name:    "output",
		Aliases: []string{"o"},
		Usage:   "Output to file",
	}
	boolFlagForceNew = &cli.BoolFlag{
		Name:    "force-new",
		Aliases: []string{"N"},
		Usage:   "Force fetch cloud STS token, ignore cache (including OpenId configuration etc.)",
	}
)

func BuildCommand() *cli.Command {
	flags := []cli.Flag{
		stringFlagProfile,
		stringFlagFormat,
		stringFlagOidcField,
		stringFlagOutput,
		boolFlagForceNew,
	}
	return &cli.Command{
		Name:  "fetch-token",
		Usage: "Fetch cloud STS token",
		Flags: flags,
		Action: func(context *cli.Context) error {
			profile := context.String("profile")
			format := context.String("format")
			oidcField := context.String("oidc-field")
			output := context.String("output")
			forceNew := context.Bool("force-new")

			return fetchToken(profile, format, oidcField, output, forceNew)
		},
	}
}

func fetchToken(profile, format, oidcField, output string, forceNew bool) error {
	options := &cloud.FetchCloudStsOptions{
		ForceNew: forceNew,
	}
	oidcTokenType := oidc.GetOidcTokenType(oidcField)
	options.FetchOidcTokenType = oidcTokenType

	sts, _, err := cloud.FetchCloudStsFromDefaultConfig(profile, options)
	if err != nil {
		return err
	}

	var stdOutput string
	var stdOutputErr error
	printNewLine := true

	if alibabaCloudSts, ok := sts.(*alibaba_cloud.StsToken); ok {
		stdOutput, stdOutputErr = alibabaCloudSts.MarshalWithFormat(format)
	} else if awsStsToken, ok := sts.(*aws.AwsStsToken); ok {
		stdOutput, stdOutputErr = awsStsToken.Marshal()
	} else if oidcToken, ok := sts.(*oidc.OidcToken); ok {
		if oidcTokenType == oidc.FetchIdToken {
			printNewLine = false
			stdOutput = oidcToken.IdToken
		} else if oidcTokenType == oidc.FetchAccessToken {
			printNewLine = false
			stdOutput = oidcToken.AccessToken
		} else {
			stdOutput, stdOutputErr = oidcToken.Marshal()
		}
	} else {
		return fmt.Errorf("unknown cloud STS token type")
	}

	if stdOutputErr != nil {
		return stdOutputErr
	}
	if output == "" {
		if printNewLine {
			utils.Stdout.Println(stdOutput)
		} else {
			utils.Stdout.Print(stdOutput)
		}
	} else {
		// write to file output
		err := writeFilePreservePerm(output, []byte(stdOutput), 0644)
		if err != nil {
			return errors.Errorf("Write to file: %s, error: %v", output, err)
		}
	}
	return nil
}

func writeFilePreservePerm(filename string, data []byte, perm os.FileMode) error {
	if _, err := os.Stat(filename); err == nil {
		f, err := os.OpenFile(filename, os.O_WRONLY|os.O_TRUNC, 0)
		if err != nil {
			return err
		}
		defer f.Close()
		_, err = f.Write(data)
		return err
	} else if os.IsNotExist(err) {
		return os.WriteFile(filename, data, perm)
	} else {
		// other error
		return err
	}
}
