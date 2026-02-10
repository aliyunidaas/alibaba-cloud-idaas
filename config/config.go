package config

import (
	"encoding/base64"
	"encoding/json"
	"fmt"

	"github.com/aliyunidaas/alibaba-cloud-idaas/idaaslog"
	"github.com/aliyunidaas/alibaba-cloud-idaas/utils"
	"github.com/pkg/errors"
)

const (
	Version1 = "1"
)

type CloudCredentialConfig struct {
	Version        string                     `json:"version"` // current version always ("1" - Version1)
	CurrentProfile string                     `json:"current_profile"`
	Profile        map[string]*CloudStsConfig `json:"profile"` // required
}

func FindProfile(configFilename, profile string, ignoreParseFromProfile bool) (string, *CloudStsConfig, error) {
	var tempProfile string
	var cloudStsConfig *CloudStsConfig
	if !ignoreParseFromProfile {
		tempProfile, cloudStsConfig = TryParseProfileFromInput(profile)
		if cloudStsConfig != nil {
			return tempProfile, cloudStsConfig, nil
		}
	}
	cloudCredentialConfig, err := LoadCloudCredentialConfig(configFilename)
	if err != nil {
		return profile, nil, err
	}
	profile, cloudStsConfig = cloudCredentialConfig.FindProfile(profile)
	if cloudStsConfig == nil {
		return profile, nil, fmt.Errorf("profile: %s not found", profile)
	}
	return profile, cloudStsConfig, nil
}

func (c *CloudCredentialConfig) FindProfile(profile string) (string, *CloudStsConfig) {
	if c == nil {
		return "", nil
	}
	idaaslog.Debug.PrintfLn("Init profile: %s", profile)
	if profile == "" {
		if c.CurrentProfile != "" {
			profile = c.CurrentProfile
			idaaslog.Info.PrintfLn("Current profile: %s", profile)
		} else {
			profile = "default"
			idaaslog.Info.PrintfLn("Default profile: %s", profile)
		}
	}
	p, ok := c.Profile[profile]
	if ok {
		idaaslog.Info.PrintfLn("Profile found: %s", profile)
		return profile, p
	} else {
		idaaslog.Info.PrintfLn("Profile not found: %s", profile)
		return profile, nil
	}
}

func TryParseProfileFromInput(profile string) (string, *CloudStsConfig) {
	if profile != "" {
		tempProfile := fmt.Sprintf("temp-%s", utils.Sha256ToHex(profile))
		var cloudStsConfig CloudStsConfig
		if json.Unmarshal([]byte(profile), &cloudStsConfig) == nil {
			return tempProfile, &cloudStsConfig
		}
		if profileDebase64, err := base64.StdEncoding.DecodeString(profile); err == nil {
			if json.Unmarshal(profileDebase64, &cloudStsConfig) == nil {
				return tempProfile, &cloudStsConfig
			}
		}
	}
	return profile, nil
}

type CloudStsConfig struct {
	AlibabaCloud *AlibabaCloudStsConfig   `json:"alibaba_cloud_sts"`   // optional, AlibabaCloud, Aws, OidcToken or CloudAccount one required
	Aws          *AwsCloudStsConfig       `json:"aws_sts"`             // optional, see AlibabaCloud
	OidcToken    *OidcTokenProviderConfig `json:"oidc_token"`          // optional, see AlibabaCloud
	CloudAccount *CloudAccountTokenConfig `json:"cloud_account_token"` // optional, see AlibabaCloud
	Environments []string                 `json:"environments"`        // optional, environments for execute
	Comment      string                   `json:"comment"`             // optional
}

type CloudAccountTokenConfig struct {
	// Endpoint and region: https://api.aliyun.com/product/Eiam-developerapi
	// Endpoint e.g. https://eiam-developerapi.cn-hangzhou.aliyuncs.com/v2/idaas_***/cloudAccountRoles/_/actions/obtainAccessCredential
	CloudAccountRegion         string                   `json:"cloud_account_region"`
	CloudAccountInstanceId     string                   `json:"cloud_account_instance_id"`
	CloudAccountEndpoint       string                   `json:"cloud_account_endpoint"`
	CloudAccountRoleExternalId string                   `json:"cloud_account_role_external_id"`
	AccessTokenProvider        *OidcTokenProviderConfig `json:"access_token_provider"`
}

func (c *CloudAccountTokenConfig) GetCloudAccountEndpoint() (string, error) {
	if c == nil {
		return "", errors.New("nil cloud account token config")
	}
	if c.CloudAccountEndpoint != "" {
		return c.CloudAccountEndpoint, nil
	}
	if c.CloudAccountRegion == "" || c.CloudAccountInstanceId == "" {
		return "", errors.New("cloud account token config missing cloud account region or instance id")
	}
	return fmt.Sprintf("https://eiam-developerapi.%s.aliyuncs.com/v2/%s/cloudAccountRoles/_/actions/obtainAccessCredential", c.CloudAccountRegion, c.CloudAccountInstanceId), nil
}

type AlibabaCloudStsConfig struct {
	Region            string                   `json:"region"`
	StsEndpoint       string                   `json:"sts_endpoint"`        // required
	OidcProviderArn   string                   `json:"oidc_provider_arn"`   // required
	RoleArn           string                   `json:"role_arn"`            // required
	DurationSeconds   int64                    `json:"duration_seconds"`    // optional
	RoleSessionName   string                   `json:"role_session_name"`   // optional, generate role session name when absent
	OidcTokenProvider *OidcTokenProviderConfig `json:"oidc_token_provider"` // required at this moment
}

type AwsCloudStsConfig struct {
	Region            string                   `json:"region"`              // required
	RoleArn           string                   `json:"role_arn"`            // required
	DurationSeconds   int32                    `json:"duration_seconds"`    // optional
	RoleSessionName   string                   `json:"role_session_name"`   // optional, generate role session name when absent
	OidcTokenProvider *OidcTokenProviderConfig `json:"oidc_token_provider"` // required at this moment
}

type OidcTokenProviderConfig struct {
	TokenType                          string                                    `json:"token_type"`         // for device_code, id_token[default], access_token
	OidcTokenProviderClientCredentials *OidcTokenProviderClientCredentialsConfig `json:"client_credentials"` // optional *
	OidcTokenProviderDeviceCode        *OidcTokenProviderDeviceCodeConfig        `json:"device_code"`        // optional *
	// * only requires one
}

func (o *OidcTokenProviderConfig) GetCacheKey() string {
	return fmt.Sprintf("%s_%s", o.GetId(), o.Digest()[0:32])
}

func (c *OidcTokenProviderConfig) GetId() string {
	if c.OidcTokenProviderClientCredentials != nil {
		return c.OidcTokenProviderClientCredentials.ClientId
	}
	if c.OidcTokenProviderDeviceCode != nil {
		return c.OidcTokenProviderDeviceCode.ClientId
	}
	return "unknown_oidc"
}

func (c *OidcTokenProviderConfig) Marshal() string {
	if c == nil {
		return "\"null\""
	}
	configJson, err := json.Marshal(c)
	if err != nil {
		return "\"error:" + err.Error() + "\""
	}
	return string(configJson)
}

type OidcTokenProviderClientCredentialsConfig struct {
	TokenEndpoint                      string           `json:"token_endpoint"`                        // required
	ClientId                           string           `json:"client_id"`                             // required
	Scope                              string           `json:"scope"`                                 // optional
	ApplicationFederatedCredentialName string           `json:"application_federated_credential_name"` // optional
	ClientSecret                       string           `json:"client_secret"`                         // optional *
	ClientAssertionSinger              *ExSingerConfig  `json:"client_assertion_singer"`               // optional *
	ClientAssertionPkcs7Config         *Pkcs7Config     `json:"client_assertion_pkcs7"`                // optional *
	ClientAssertionPrivateCaConfig     *PrivateCaConfig `json:"client_assertion_private_ca"`           // optional *
	ClientAssertionOidcTokenConfig     *OidcTokenConfig `json:"client_assertion_oidc_token"`           // optional *
	// * requires one
}

type OidcTokenProviderDeviceCodeConfig struct {
	Issuer       string `json:"issuer"`        // required
	ClientId     string `json:"client_id"`     // required
	Scope        string `json:"scope"`         // optional, default openid
	ClientSecret string `json:"client_secret"` // optional, when public client
	AutoOpenUrl  bool   `json:"auto_open_url"` // optional, auto open in browser, use in local device
	ShowQrCode   bool   `json:"show_qr_code"`  // optional, show QR code, use in server
	SmallQrCode  bool   `json:"small_qr_code"` // optional, show small QR code, may cause compatible issue
}

// Pkcs7Config
// Alibaba Cloud, AWS, Azure
// reference:
// - https://www.alibabacloud.com/help/en/ecs/user-guide/use-instance-identities
// - https://docs.aws.amazon.com/AWSEC2/latest/UserGuide/verify-iid.html
type Pkcs7Config struct {
	Provider                    string `json:"provider"`                        // required, enums: alibaba_cloud, aws, azure ...
	AlibabaCloudMode            string `json:"alibaba_cloud_mode"`              // optional, normal(default), secure (security hardening)
	AlibabaCloudIdaasInstanceId string `json:"alibaba_cloud_idaas_instance_id"` // optional, should be IDaaS instance ID
}

type PrivateCaConfig struct {
	Certificate          string          `json:"certificate"`            // optional, certificate,base64 or PEM
	CertificateFile      string          `json:"certificate_file"`       // optional, certificate file @see Certificate, Certificate and CertificateFile requires one
	CertificateChain     string          `json:"certificate_chain"`      // optional, certificate chain, base64 or PEM, separator ","
	CertificateChainFile string          `json:"certificate_chain_file"` // optional, certificate chain file @see CertificateChain
	CertificateKeySigner *ExSingerConfig `json:"certificate_key_signer"` // optional, when private stored in external
}

// OidcTokenConfig
// reference:
// - https://cloud.google.com/compute/docs/instances/verifying-instance-identity
type OidcTokenConfig struct {
	Provider            string `json:"provider"`               // required, enums: gcp, custom
	GoogleVmIdentityUrl string `json:"google_vm_identity_url"` // optional, only for gcp
	GoogleVmIdentityAud string `json:"google_vm_identity_aud"` // optional, only for gcp
	OidcToken           string `json:"oidc_token"`             // optional, only for custom
	OidcTokenFile       string `json:"oidc_token_file"`        // optional, only for custom, OidcToken and OidcTokenFile requires one
}

type ExSingerConfig struct {
	KeyID           string                         `json:"key_id"`           // optional, PCA do not requires key_id
	Algorithm       string                         `json:"algorithm"`        // required, RS256, RS384, RS512, ES256, ES384, ES512
	Pkcs11          *ExSignerPkcs11Config          `json:"pkcs11"`           // optional *
	YubikeyPiv      *ExSignerYubikeyPivConfig      `json:"yubikey_piv"`      // optional *
	ExternalCommand *ExSignerExternalCommandConfig `json:"external_command"` // optional *
	KeyFile         *ExSingerKeyFileConfig         `json:"key_file"`         // optional *
	// * pkcs11, yubikey_piv, external_command, key_file requires one
}

type ExSignerPkcs11Config struct {
	LibraryPath string `json:"library_path"` // required
	TokenLabel  string `json:"token_label"`  // required
	KeyLabel    string `json:"key_label"`    // required
	Pin         string `json:"pin"`          // optional, or set env PKS11_PIN
}

type ExSignerYubikeyPivConfig struct {
	Slot      string `json:"slot"`       // required, auth,sign or rN
	Pin       string `json:"pin"`        // optional, or set env YUBIKEY_PIN
	PinPolicy string `json:"pin_policy"` // required, none, once or always
}

type ExSignerExternalCommandConfig struct {
	Command   string `json:"command"`   // required
	Parameter string `json:"parameter"` // required
}

type ExSingerKeyFileConfig struct {
	Key      string `json:"key"`      // optional *
	File     string `json:"file"`     // optional *
	Password string `json:"password"` // optional, for PKCS#8 encrypted private key
	// * key, file requires one
}
