package config

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
)

func (c *CloudStsConfig) Digest() string {
	if c == nil {
		return ""
	}
	// Comment do not effect digest(cache)
	return digest(c.AlibabaCloud.Digest(), c.Aws.Digest(), c.OidcToken.Digest(), c.CloudAccount.Digest())
}

func (c *CloudAccountTokenConfig) Digest() string {
	if c == nil {
		return ""
	}
	return digest(c.CloudAccountEndpoint, c.CloudAccountRoleExternalId, c.AccessTokenProvider.Digest())
}

func (c *AlibabaCloudStsConfig) Digest() string {
	if c == nil {
		return ""
	}
	return digest(c.Region, c.StsEndpoint, c.OidcProviderArn, c.RoleArn,
		fmt.Sprintf("%d", c.DurationSeconds), c.RoleSessionName, c.OidcTokenProvider.Digest())
}

func (c *AwsCloudStsConfig) Digest() string {
	if c == nil {
		return ""
	}
	return digest(c.Region, c.RoleArn, fmt.Sprintf("%d", c.DurationSeconds),
		c.RoleSessionName, c.OidcTokenProvider.Digest())
}

func (c *OidcTokenProviderConfig) Digest() string {
	if c == nil {
		return ""
	}
	return digest(c.TokenType, c.OidcTokenProviderClientCredentials.Digest(),
		c.OidcTokenProviderDeviceCode.Digest())
}

func (c *OidcTokenProviderClientCredentialsConfig) Digest() string {
	if c == nil {
		return ""
	}
	// ClientSecret do note effect digest(cache)
	return digest(c.TokenEndpoint, c.ClientId, c.Scope, c.ApplicationFederatedCredentialName,
		c.ClientAssertionSinger.Digest(),
		c.ClientAssertionPkcs7Config.Digest(),
		c.ClientAssertionPrivateCaConfig.Digest(),
		c.ClientAssertionOidcTokenConfig.Digest())
}

func (c *OidcTokenProviderDeviceCodeConfig) Digest() string {
	if c == nil {
		return ""
	}
	// ClientSecret, AutoOpenUrl, ShowQrCode, SmallQrCode do not effect digest(cache)
	return digest(c.Issuer, c.ClientId, c.Scope)
}

func (c *Pkcs7Config) Digest() string {
	if c == nil {
		return ""
	}
	return digest(c.Provider, c.AlibabaCloudMode)
}

func (c *PrivateCaConfig) Digest() string {
	if c == nil {
		return ""
	}
	return digest(c.Certificate, fileModTime(c.CertificateFile),
		c.CertificateKeySigner.Digest(),
		c.CertificateChain, fileModTime(c.CertificateChainFile))
}

func (c *OidcTokenConfig) Digest() string {
	if c == nil {
		return ""
	}
	return digest(c.Provider, c.OidcToken, fileModTime(c.OidcTokenFile))
}

func (c *ExSingerConfig) Digest() string {
	if c == nil {
		return ""
	}
	return digest(c.KeyID, c.Algorithm,
		c.Pkcs11.Digest(), c.YubikeyPiv.Digest(),
		c.ExternalCommand.Digest(), c.KeyFile.Digest())
}

func (c *ExSignerPkcs11Config) Digest() string {
	if c == nil {
		return ""
	}
	// Pin do not effect digest(cache)
	return digest(c.LibraryPath, c.TokenLabel, c.KeyLabel)
}

func (c *ExSignerYubikeyPivConfig) Digest() string {
	if c == nil {
		return ""
	}
	// Pin, PinPolicy do note effect digest(cache)
	return digest(c.Slot)
}

func (c *ExSignerExternalCommandConfig) Digest() string {
	if c == nil {
		return ""
	}
	return digest(c.Command, c.Parameter)
}

func (c *ExSingerKeyFileConfig) Digest() string {
	if c == nil {
		return ""
	}
	return digest(c.Key, c.File, fileModTime(c.File))
}

func digest(args ...string) string {
	h := sha256.New()
	for _, a := range args {
		h.Write([]byte(a))
	}
	return hex.EncodeToString(h.Sum(nil))
}

func fileModTime(filename string) string {
	if filename == "" {
		return ""
	}
	fileInfo, err := os.Stat(filename)
	if err != nil {
		return ""
	}
	return fmt.Sprintf("%x", fileInfo.ModTime().Unix())
}
