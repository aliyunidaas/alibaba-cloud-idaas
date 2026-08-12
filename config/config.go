package config

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/aliyunidaas/alibaba-cloud-idaas/idaaslog"
	"github.com/aliyunidaas/alibaba-cloud-idaas/utils"
	"github.com/pkg/errors"
)

const (
	Version1 = "1"
)

type CloudCredentialConfig struct {
	Version        string                     `json:"version"` // current version always ("1" - Version1)
	CurrentProfile string                     `json:"current_profile,omitempty"`
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
			cloudStsConfig.WarnDeprecatedFields(tempProfile)
			return tempProfile, &cloudStsConfig
		}
		if profileDebase64, err := base64.StdEncoding.DecodeString(profile); err == nil {
			if json.Unmarshal(profileDebase64, &cloudStsConfig) == nil {
				cloudStsConfig.WarnDeprecatedFields(tempProfile)
				return tempProfile, &cloudStsConfig
			}
		}
	}
	return profile, nil
}

type CloudStsConfig struct {
	AlibabaCloud *AlibabaCloudStsConfig   `json:"alibaba_cloud_sts,omitempty"`   // optional, AlibabaCloud, Aws, OidcToken or CloudAccount one required
	Aws          *AwsCloudStsConfig       `json:"aws_sts,omitempty"`             // optional, see AlibabaCloud
	OidcToken    *OidcTokenProviderConfig `json:"oidc_token,omitempty"`          // optional, see AlibabaCloud
	CloudAccount *CloudAccountTokenConfig `json:"cloud_account_token,omitempty"` // optional, see AlibabaCloud
	Credential   *CredentialConfig        `json:"credential,omitempty"`          // optional, static credential via Developer API
	Agent        *AgentConfig             `json:"agent,omitempty"`               // optional, see AlibabaCloud
	Environments []string                 `json:"environments,omitempty"`        // optional, environments for execute
	Comment      string                   `json:"comment,omitempty"`             // optional
}

// OidcTokenProviders collects every OIDC token provider referenced by this profile,
// regardless of which cloud/credential provider owns it.
func (c *CloudStsConfig) OidcTokenProviders() []*OidcTokenProviderConfig {
	if c == nil {
		return nil
	}
	candidates := []*OidcTokenProviderConfig{c.OidcToken}
	if c.AlibabaCloud != nil {
		candidates = append(candidates, c.AlibabaCloud.OidcTokenProvider)
	}
	if c.Aws != nil {
		candidates = append(candidates, c.Aws.OidcTokenProvider)
	}
	if c.CloudAccount != nil {
		candidates = append(candidates, c.CloudAccount.AccessTokenProvider)
	}
	if c.Credential != nil {
		candidates = append(candidates, c.Credential.AccessTokenProvider)
	}
	if c.Agent != nil {
		candidates = append(candidates, c.Agent.AccessTokenProvider)
	}

	var providers []*OidcTokenProviderConfig
	for _, provider := range candidates {
		if provider != nil {
			providers = append(providers, provider)
		}
	}
	return providers
}

// WarnDeprecatedFields logs the deprecated config keys used by this profile. The values
// still take effect, but users should migrate to the current key names.
func (c *CloudStsConfig) WarnDeprecatedFields(profile string) {
	for _, provider := range c.OidcTokenProviders() {
		clientCredentials := provider.OidcTokenProviderClientCredentials
		if clientCredentials == nil {
			continue
		}
		if clientCredentials.ClientAssertionSinger != nil && clientCredentials.ClientAssertionSigner == nil {
			idaaslog.Warn.PrintfLn("Profile %s uses deprecated config key `client_assertion_singer`, "+
				"please rename it to `client_assertion_signer`", profile)
		}
	}
}

type CloudAccountTokenConfig struct {
	// Endpoint and region: https://api.aliyun.com/product/Eiam-developerapi
	// Endpoint e.g. https://eiam-developerapi.cn-hangzhou.aliyuncs.com/v2/idaas_***/cloudAccountRoles/_/actions/obtainAccessCredential
	CloudAccountRegion         string                   `json:"cloud_account_region,omitempty"`      // Deprecated: use developer_api_endpoint (region is embedded in the host)
	CloudAccountInstanceId     string                   `json:"cloud_account_instance_id,omitempty"` // Deprecated: use instance_id
	CloudAccountEndpoint       string                   `json:"cloud_account_endpoint,omitempty"`    // Deprecated: use developer_api_endpoint
	InstanceId                 string                   `json:"instance_id,omitempty"`               // recommend
	DeveloperApiEndpoint       string                   `json:"developer_api_endpoint,omitempty"`    // recommend
	CloudAccountRoleExternalId string                   `json:"cloud_account_role_external_id,omitempty"`
	AccessTokenProvider        *OidcTokenProviderConfig `json:"access_token_provider,omitempty"`
}

func (c *CloudAccountTokenConfig) GetInstanceId() string {
	if c == nil {
		return ""
	}
	if c.InstanceId != "" {
		return c.InstanceId
	}
	return c.CloudAccountInstanceId
}

func (c *CloudAccountTokenConfig) GetEndpoint() string {
	if c == nil {
		return ""
	}
	if c.DeveloperApiEndpoint != "" {
		return c.DeveloperApiEndpoint
	}
	return c.CloudAccountEndpoint
}

func (c *CloudAccountTokenConfig) GetCloudAccountEndpoint() (string, error) {
	if c == nil {
		return "", errors.New("nil cloud account token config")
	}
	if c.CloudAccountEndpoint != "" {
		return c.CloudAccountEndpoint, nil
	}
	instanceId := c.GetInstanceId()
	if c.DeveloperApiEndpoint != "" {
		return buildDeveloperApiUrl(c.DeveloperApiEndpoint, instanceId, "/cloudAccountRoles/_/actions/obtainAccessCredential"), nil
	}
	if c.CloudAccountRegion == "" || instanceId == "" {
		return "", errors.New("cloud account token config missing cloud account region or instance id")
	}
	return fmt.Sprintf("https://eiam-developerapi.%s.aliyuncs.com/v2/%s/cloudAccountRoles/_/actions/obtainAccessCredential", c.CloudAccountRegion, instanceId), nil
}

type AgentConfig struct {
	InstanceId           string                   `json:"instance_id,omitempty"`
	DeveloperApiEndpoint string                   `json:"developer_api_endpoint,omitempty"`
	AccessTokenProvider  *OidcTokenProviderConfig `json:"access_token_provider,omitempty"`
}

func (c *AgentConfig) CreateUserExclusiveCredentialEndpoint() (string, error) {
	if c == nil {
		return "", errors.New("nil agent config")
	}
	if !checkEndpointAndInstanceId(c.DeveloperApiEndpoint, c.InstanceId) {
		return "", errors.New("agent config missing instance_id or developer_api_endpoint")
	}
	return buildDeveloperApiUrl(c.DeveloperApiEndpoint, c.InstanceId, "/credentials/_/actions/createUserExclusive"), nil
}

func (c *AgentConfig) DecryptUserExclusiveCredentialEndpoint() (string, error) {
	if c == nil {
		return "", errors.New("nil agent config")
	}
	if !checkEndpointAndInstanceId(c.DeveloperApiEndpoint, c.InstanceId) {
		return "", errors.New("agent config missing instance_id or developer_api_endpoint")
	}
	return buildDeveloperApiUrl(c.DeveloperApiEndpoint, c.InstanceId, "/credentials/_/actions/decryptUserExclusiveCredentialCiphertext"), nil
}

func (c *AgentConfig) GetCredentialEndpoint() (string, error) {
	if c == nil {
		return "", errors.New("nil agent config")
	}
	if !checkEndpointAndInstanceId(c.DeveloperApiEndpoint, c.InstanceId) {
		return "", errors.New("agent config missing instance_id or developer_api_endpoint")
	}
	return buildDeveloperApiUrl(c.DeveloperApiEndpoint, c.InstanceId, "/credentials/_/actions/obtain"), nil
}

func (c *AgentConfig) Clone() (*AgentConfig, error) {
	if c == nil {
		return nil, nil
	}
	configJson, err := json.Marshal(c)
	if err != nil {
		return nil, errors.Wrap(err, "error cloning AgentConfig")
	}
	var clonedConfig = &AgentConfig{}
	err = json.Unmarshal(configJson, &clonedConfig)
	if err != nil {
		return nil, errors.Wrap(err, "error cloning AgentConfig")
	}
	return clonedConfig, nil
}

// DeveloperApiConfig is the shared base for Developer-API-backed providers:
// instance_id + developer_api_endpoint + access_token_provider. New providers embed it.
// (cloud_account_token / agent keep their own fields for backward compatibility.)
type DeveloperApiConfig struct {
	InstanceId           string                   `json:"instance_id,omitempty"`
	DeveloperApiEndpoint string                   `json:"developer_api_endpoint,omitempty"`
	AccessTokenProvider  *OidcTokenProviderConfig `json:"access_token_provider,omitempty"`
}

func (c *DeveloperApiConfig) BuildDeveloperApiUrl(path string) (string, error) {
	if c == nil {
		return "", errors.New("nil developer api config")
	}
	if !checkEndpointAndInstanceId(c.DeveloperApiEndpoint, c.InstanceId) {
		return "", errors.New("missing instance_id or developer_api_endpoint")
	}
	return buildDeveloperApiUrl(c.DeveloperApiEndpoint, c.InstanceId, path), nil
}

// CredentialConfig fetches a static credential via the Developer API.
type CredentialConfig struct {
	DeveloperApiConfig
	CredentialIdentifier string `json:"credential_identifier,omitempty"` // credential name or id
}

func (c *CredentialConfig) GetObtainCredentialEndpoint() (string, error) {
	if c == nil {
		return "", errors.New("nil credential config")
	}
	return c.BuildDeveloperApiUrl("/credentials/_/actions/obtain")
}

type AlibabaCloudStsConfig struct {
	Region            string                   `json:"region,omitempty"`
	StsEndpoint       string                   `json:"sts_endpoint,omitempty"`        // required
	OidcProviderArn   string                   `json:"oidc_provider_arn,omitempty"`   // required
	RoleArn           string                   `json:"role_arn,omitempty"`            // required
	DurationSeconds   int64                    `json:"duration_seconds,omitempty"`    // optional
	RoleSessionName   string                   `json:"role_session_name,omitempty"`   // optional, generate role session name when absent
	OidcTokenProvider *OidcTokenProviderConfig `json:"oidc_token_provider,omitempty"` // required at this moment
}

type AwsCloudStsConfig struct {
	Region            string                   `json:"region,omitempty"`              // required
	RoleArn           string                   `json:"role_arn,omitempty"`            // required
	DurationSeconds   int32                    `json:"duration_seconds,omitempty"`    // optional
	RoleSessionName   string                   `json:"role_session_name,omitempty"`   // optional, generate role session name when absent
	OidcTokenProvider *OidcTokenProviderConfig `json:"oidc_token_provider,omitempty"` // required at this moment
}

type OidcTokenProviderConfig struct {
	TokenType                          string                                    `json:"token_type,omitempty"`         // for device_code: access_token[default], id_token
	OidcTokenProviderClientCredentials *OidcTokenProviderClientCredentialsConfig `json:"client_credentials,omitempty"` // optional *
	OidcTokenProviderDeviceCode        *OidcTokenProviderDeviceCodeConfig        `json:"device_code,omitempty"`        // optional *
	OpenApi                            *OpenApiConfig                            `json:"open_api,omitempty"`           // optional *
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
	TokenEndpoint                      string           `json:"token_endpoint,omitempty"`                        // required
	ClientId                           string           `json:"client_id,omitempty"`                             // required
	Scope                              string           `json:"scope,omitempty"`                                 // optional
	ApplicationFederatedCredentialName string           `json:"application_federated_credential_name,omitempty"` // optional
	ClientSecret                       string           `json:"client_secret,omitempty"`                         // optional *
	ClientAssertionSigner              *ExSignerConfig  `json:"client_assertion_signer,omitempty"`               // optional *
	ClientAssertionSinger              *ExSignerConfig  `json:"client_assertion_singer,omitempty"`               // Deprecated: typo, use client_assertion_signer
	ClientAssertionPkcs7Config         *Pkcs7Config     `json:"client_assertion_pkcs7,omitempty"`                // optional *
	ClientAssertionPrivateCaConfig     *PrivateCaConfig `json:"client_assertion_private_ca,omitempty"`           // optional *
	ClientAssertionOidcTokenConfig     *OidcTokenConfig `json:"client_assertion_oidc_token,omitempty"`           // optional *
	// * requires one
}

// GetClientAssertionSigner returns the client assertion signer config, falling back to the
// deprecated misspelled `client_assertion_singer` field so that configs written for
// v0.1.x keep working.
func (c *OidcTokenProviderClientCredentialsConfig) GetClientAssertionSigner() *ExSignerConfig {
	if c == nil {
		return nil
	}
	if c.ClientAssertionSigner != nil {
		return c.ClientAssertionSigner
	}
	return c.ClientAssertionSinger
}

type OidcTokenProviderDeviceCodeConfig struct {
	Issuer       string `json:"issuer,omitempty"`        // required
	ClientId     string `json:"client_id,omitempty"`     // required
	Scope        string `json:"scope,omitempty"`         // optional, default openid
	ClientSecret string `json:"client_secret,omitempty"` // optional, when public client
	AutoOpenUrl  bool   `json:"auto_open_url,omitempty"` // optional, auto open in browser, use in local device
	ShowQrCode   bool   `json:"show_qr_code,omitempty"`  // optional, show QR code, use in server
	SmallQrCode  bool   `json:"small_qr_code,omitempty"` // optional, show small QR code, may cause compatible issue
}

// OpenApiConfig
// reference:
// - https://github.com/aliyun/credentials-go
// - https://api.aliyun.com/api/Eiam/2021-12-01/GenerateOauthToken?RegionId=cn-hangzhou
type OpenApiConfig struct {
	InstanceId      string   `json:"instance_id"`
	ApplicationId   string   `json:"application_id"`
	ScopeValues     []string `json:"scope_values"`
	Audience        string   `json:"audience"`
	OpenApiEndpoint string   `json:"open_api_endpoint"`

	// Credential type, including access_key, sts, bearer, ecs_ram_role, ram_role_arn, rsa_key_pair, oidc_role_arn, credentials_uri
	Type            string `json:"type"`
	AccessKeyId     string `json:"access_key_id"`
	AccessKeySecret string `json:"access_key_secret"`
	SecurityToken   string `json:"security_token"`

	// Used when the type is ram_role_arn or oidc_role_arn
	OIDCProviderArn       string `json:"oidc_provider_arn"`
	OIDCTokenFilePath     string `json:"oidc_token"`
	RoleArn               string `json:"role_arn"`
	RoleSessionName       string `json:"role_session_name"`
	RoleSessionExpiration int    `json:"role_session_expiration"`
	Policy                string `json:"policy"`
	ExternalId            string `json:"external_id"`
	STSEndpoint           string `json:"sts_endpoint"`

	// Used when the type is ecs_ram_role
	RoleName string `json:"role_name"`

	// Used when the type is credentials_uri
	Url string `json:"url"`
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
	CertificateKeySigner *ExSignerConfig `json:"certificate_key_signer"` // optional, when private stored in external
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

type ExSignerConfig struct {
	KeyID           string                         `json:"key_id"`           // optional, PCA do not requires key_id
	Algorithm       string                         `json:"algorithm"`        // required, RS256, RS384, RS512, ES256, ES384, ES512
	Pkcs11          *ExSignerPkcs11Config          `json:"pkcs11"`           // optional *
	YubikeyPiv      *ExSignerYubikeyPivConfig      `json:"yubikey_piv"`      // optional *
	ExternalCommand *ExSignerExternalCommandConfig `json:"external_command"` // optional *
	KeyFile         *ExSignerKeyFileConfig         `json:"key_file"`         // optional *
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

type ExSignerKeyFileConfig struct {
	Key      string `json:"key"`      // optional *
	File     string `json:"file"`     // optional *
	Password string `json:"password"` // optional, for PKCS#8 encrypted private key
	// * key, file requires one
}

func checkEndpointAndInstanceId(endpoint string, instanceId string) bool {
	if isInstanceDomain(endpoint) {
		return true
	}
	return instanceId != ""
}

func buildDeveloperApiUrl(endpoint, instanceId, path string) string {
	if isInstanceDomain(endpoint) {
		// https://example.aliyunidaas.com/api/v2/resources/operation
		return fmt.Sprintf("%s/api/v2%s", utils.NormalizeDeveloperApiEndpoint(endpoint), path)
	}
	// https://eiam-developerapi.region-id.aliyuncs.com/v2/instance-id/resources/operation
	return fmt.Sprintf("%s/v2/%s%s", utils.NormalizeDeveloperApiEndpoint(endpoint), instanceId, path)
}

func isInstanceDomain(endpoint string) bool {
	lowerEndpoint := strings.ToLower(endpoint)
	return strings.Contains(lowerEndpoint, ".aliyunidaas.com") || strings.Contains(lowerEndpoint, ".cloud-idaas.com")
}
