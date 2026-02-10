package constants

import (
	"os"
	"path/filepath"
)

var (
	ConfigRootDir  = getConfigRootDir()
	ConfigIdaasDir = getConfigIdaasDir()
	ConfigFilename = getConfigFilename()
)

const (
	AlibabaCloudIdaasCliVersion = "0.1.0-preview9"

	DefaultAudienceAlibabaCloudIdaas = "alibaba-cloud-idaas-v2"

	LogDir = "__log"

	// CategoryCloudToken cloud token, e.g. Alibaba Cloud STS Token
	CategoryCloudToken = "cloud_token"
	// CategoryOidc OIDC OpenID Configuration
	CategoryOidc = "oidc"
	// CategoryOidcToken ID Token or Access Token(JWT)
	CategoryOidcToken = "oidc_token"
	// CategoryTokenResponse Token Response with Refresh Token
	CategoryTokenResponse = "token_response"

	EnvUserAgent            = "ALIBABA_CLOUD_IDAAS_USER_AGENT"
	EnvUnsafeDebug          = "ALIBABA_CLOUD_IDAAS_UNSAFE_DEBUG"
	EnvUnsafeConsolePrint   = "ALIBABA_CLOUD_IDAAS_UNSAFE_CONSOLE_PRINT"
	EnvPkcs11Pin            = "ALIBABA_CLOUD_IDAAS_PKSC11_PIN"
	EnvYubiKeyPin           = "ALIBABA_CLOUD_IDAAS_YUBIKEY_PIN"
	EnvPkcs8Password        = "ALIBABA_CLOUD_IDAAS_PKCS8_PASSWORD"
	EnvEnableEncryptWithMac = "ALIBABA_CLOUD_IDAAS_ENABLE_ENCRYPT_WITH_MAC"
	EnvConfigFile           = "ALIBABA_CLOUD_IDAAS_CONFIG_FILE"

	UrlIdaasProduct                = "https://www.aliyun.com/product/idaas"
	UrlAlibabaCloudIdaasRepository = "https://github.com/aliyunidaas/alibaba-cloud-idaas"

	ErrStopFallback = "ERROR:STOP_FALLBACK"
)

const (
	dotAliyunDir     = ".aliyun"
	dotCloudIdaasDir = ".cloud_idaas"

	alibabaCloudIdaasDir = "alibaba-cloud-idaas"
	idaasCliDir          = "cloud-cli"

	alibabaCloudIdaasConfigFile = "alibaba-cloud-idaas.json"
	idaasCliConfigFile          = "idaas-cli.json"
)

func getConfigRootDir() string {
	if useCloudIdaasDirectory() {
		return dotCloudIdaasDir
	}
	return dotAliyunDir
}

func getConfigIdaasDir() string {
	if useCloudIdaasDirectory() {
		return idaasCliDir
	}
	return alibabaCloudIdaasDir
}

func getConfigFilename() string {
	if useCloudIdaasDirectory() {
		return idaasCliConfigFile
	}
	return alibabaCloudIdaasConfigFile
}

// from preview9
// use config file `~/.aliyun/alibaba-cloud-idaas.json` if config file exists
// or else use `~/.cloud_idaas/idaas-cli.json`
func useCloudIdaasDirectory() bool {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return false
	}
	configCredentialConfigFile := filepath.Join(homeDir, dotAliyunDir, alibabaCloudIdaasConfigFile)
	if _, err := os.Stat(configCredentialConfigFile); os.IsNotExist(err) {
		return true
	}
	return false
}
