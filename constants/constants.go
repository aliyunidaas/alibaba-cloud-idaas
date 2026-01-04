package constants

const (
	AlibabaCloudIdaasCliVersion = "0.1.0-preview7"

	DefaultAudienceAlibabaCloudIdaas = "alibaba-cloud-idaas-v2"

	DotAliyunDir         = ".aliyun"
	AlibabaCloudIdaasDir = "alibaba-cloud-idaas"
	LogDir               = "__log"

	// CategoryCloudToken cloud token, e.g. Alibaba Cloud STS Token
	CategoryCloudToken = "cloud_token"
	// CategoryOidc OIDC OpenID Configuration
	CategoryOidc = "oidc"
	// CategoryOidcToken ID Token or Access Token(JWT)
	CategoryOidcToken = "oidc_token"
	// CategoryTokenResponse Token Response with Refresh Token
	CategoryTokenResponse = "token_response"

	AlibabaCloudIdaasConfigFile = "alibaba-cloud-idaas.json"

	EnvUserAgent            = "ALIBABA_CLOUD_IDAAS_USER_AGENT"
	EnvUnsafeDebug          = "ALIBABA_CLOUD_IDAAS_UNSAFE_DEBUG"
	EnvUnsafeConsolePrint   = "ALIBABA_CLOUD_IDAAS_UNSAFE_CONSOLE_PRINT"
	EnvPkcs11Pin            = "ALIBABA_CLOUD_IDAAS_PKSC11_PIN"
	EnvYubiKeyPin           = "ALIBABA_CLOUD_IDAAS_YUBIKEY_PIN"
	EnvPkcs8Password        = "ALIBABA_CLOUD_IDAAS_PKCS8_PASSWORD"
	EnvEnableEncryptWithMac = "ALIBABA_CLOUD_IDAAS_ENABLE_ENCRYPT_WITH_MAC"

	UrlIdaasProduct                = "https://www.aliyun.com/product/idaas"
	UrlAlibabaCloudIdaasRepository = "https://github.com/aliyunidaas/alibaba-cloud-idaas"

	ErrStopFallback = "ERROR:STOP_FALLBACK"
)
