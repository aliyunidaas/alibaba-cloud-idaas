package show_signer_public_key

import (
	"crypto/x509"
	"encoding/pem"
	"fmt"

	"github.com/aliyunidaas/alibaba-cloud-idaas/config"
	"github.com/urfave/cli/v2"
)

var (
	stringFlagProfile = &cli.StringFlag{
		Name:    "profile",
		Aliases: []string{"p"},
		Usage:   "IDaaS Profile",
	}
)

func BuildCommand() *cli.Command {
	flags := []cli.Flag{
		stringFlagProfile,
	}
	return &cli.Command{
		Name:  "show-signer-public-key",
		Usage: "Show ex signer public key",
		Flags: flags,
		Action: func(context *cli.Context) error {
			profile := context.String("profile")
			return showPublicKey(profile)
		},
	}
}

func showPublicKey(profile string) error {
	profile, cloudStsConfig, err := config.FindProfile(profile, false)
	if err != nil {
		return fmt.Errorf("find profile %s error %s", profile, err)
	}
	if cloudStsConfig == nil {
		return fmt.Errorf("profile %s does not exist", profile)
	}

	return printClientAssertionSignerPublicKey(cloudStsConfig)
}

func printClientAssertionSignerPublicKey(cloudStsConfig *config.CloudStsConfig) error {
	var oidcTokenProvider *config.OidcTokenProviderConfig
	if cloudStsConfig.AlibabaCloud != nil {
		if cloudStsConfig.AlibabaCloud.OidcTokenProvider != nil {
			oidcTokenProvider = cloudStsConfig.AlibabaCloud.OidcTokenProvider
		}
	}
	if cloudStsConfig.Aws != nil {
		if cloudStsConfig.Aws.OidcTokenProvider != nil {
			oidcTokenProvider = cloudStsConfig.Aws.OidcTokenProvider
		}
	}
	if oidcTokenProvider != nil && oidcTokenProvider.OidcTokenProviderClientCredentials != nil {
		oidcTokenProviderClientCredentials := oidcTokenProvider.OidcTokenProviderClientCredentials

		if oidcTokenProviderClientCredentials.ClientAssertionSinger != nil {
			clientAssertionSinger := oidcTokenProviderClientCredentials.ClientAssertionSinger
			extJwtSigner, err := config.NewExJwtSignerFromConfig(clientAssertionSinger)
			if err != nil {
				return err
			}
			extSinger := extJwtSigner.GetExtSinger()
			publicKey, err := extSinger.Public()
			if err != nil {
				return err
			}
			publicKeyDer, err := x509.MarshalPKIXPublicKey(publicKey)
			if err != nil {
				return err
			}
			publicKeyPem := pem.EncodeToMemory(&pem.Block{
				Type:  "PUBLIC KEY",
				Bytes: publicKeyDer,
			})

			fmt.Printf("%s", publicKeyPem)
			return nil
		}
	}
	return fmt.Errorf("ext signer not found")
}
