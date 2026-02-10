package show_signer_public_key

import (
	"crypto/x509"
	"encoding/pem"
	"fmt"

	"github.com/aliyunidaas/alibaba-cloud-idaas/config"
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
)

func BuildCommand() *cli.Command {
	flags := []cli.Flag{
		stringFlagConfig,
		stringFlagProfile,
	}
	return &cli.Command{
		Name:  "show-signer-public-key",
		Usage: "Show ex signer public key",
		Flags: flags,
		Action: func(context *cli.Context) error {
			configFilename := context.String("config")
			profile := context.String("profile")
			return showPublicKey(configFilename, profile)
		},
	}
}

func showPublicKey(configFilename, profile string) error {
	profile, cloudStsConfig, err := config.FindProfile(configFilename, profile, false)
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
	if cloudStsConfig.CloudAccount != nil {
		if cloudStsConfig.CloudAccount.AccessTokenProvider != nil {
			oidcTokenProvider = cloudStsConfig.CloudAccount.AccessTokenProvider
		}
	}
	if oidcTokenProvider != nil {
		oidcTokenProviderClientCredentials := oidcTokenProvider.OidcTokenProviderClientCredentials
		if oidcTokenProviderClientCredentials != nil {
			// client assertion signer
			if oidcTokenProviderClientCredentials.ClientAssertionSinger != nil {
				return printExSingerPublicKey(oidcTokenProviderClientCredentials.ClientAssertionSinger)
			}
			// client assertion private CA signer
			clientAssertionPrivateCaConfig := oidcTokenProviderClientCredentials.ClientAssertionPrivateCaConfig
			if clientAssertionPrivateCaConfig != nil && clientAssertionPrivateCaConfig.CertificateKeySigner != nil {
				return printExSingerPublicKey(clientAssertionPrivateCaConfig.CertificateKeySigner)
			}
		}
	}
	return fmt.Errorf("ext signer not found")
}

func printExSingerPublicKey(exSingerConfig *config.ExSingerConfig) error {
	extJwtSigner, err := config.NewExJwtSignerFromConfig(exSingerConfig)
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
