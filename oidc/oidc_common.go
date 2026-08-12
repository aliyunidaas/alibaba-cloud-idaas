package oidc

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/aliyunidaas/alibaba-cloud-idaas/constants"
	"github.com/aliyunidaas/alibaba-cloud-idaas/idaaslog"
	"github.com/aliyunidaas/alibaba-cloud-idaas/utils"
	"github.com/pkg/errors"
)

const (
	ClientAssertionTypeJwtBearer     = "urn:ietf:params:oauth:client-assertion-type:jwt-bearer"
	ClientAssertionTypePkcs7Bearer   = "urn:cloud:idaas:params:oauth:client-assertion-type:pkcs7-bearer"
	ClientAssertionTypeIdTokenBearer = "urn:cloud:idaas:params:oauth:client-assertion-type:id-token-bearer"
	ClientAssertionTypeX509JwtBearer = "urn:cloud:idaas:params:oauth:client-assertion-type:x509-jwt-bearer"

	GrantTypeClientCredentials = "client_credentials"
	GrantTypeRefreshToken      = "refresh_token"
	GrantTypeDeviceCode        = "urn:ietf:params:oauth:grant-type:device_code"
	GrantTypeTokenExchange     = "urn:ietf:params:oauth:grant-type:token-exchange"

	TokenTypeAccessToken = "urn:ietf:params:oauth:token-type:access_token"

	ErrorCodeAuthorizationPending = "authorization_pending"
	ErrorCodeSlowDown             = "slow_down"
	ErrorAccessDenied             = "access_denied"
)

const (
	TokenIdToken     = "id_token"
	TokenAccessToken = "access_token"
)

type OidcCommonOptions struct {
	GrantType string
	Scope     string
	// for RFC8693 OAuth 2.0 Token Exchange
	SubjectTokenType   string
	SubjectToken       string
	RequestedTokenType string
}

type FetchTokenCommonOptions struct {
	TokenEndpoint string
	ClientId      string
	GrantType     string
	Scope         string

	// for RFC8693 OAuth 2.0 Token Exchange
	SubjectTokenType   string
	SubjectToken       string
	RequestedTokenType string

	ApplicationFederatedCredentialName string
}

// TokenResponse
// expires_at - Alibaba Cloud IDaaS Spec
// specification: RFC6749
type TokenResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	TokenType    string `json:"token_type"`
	ExpiresIn    int64  `json:"expires_in"`
	ExpiresAt    int64  `json:"expires_at"`
	Scope        string `json:"scope"`
	IdToken      string `json:"id_token"`
}

// DeviceCodeResponse
// specification: RFC8628
type DeviceCodeResponse struct {
	DeviceCode              string `json:"device_code"`
	UserCode                string `json:"user_code"`
	VerificationUri         string `json:"verification_uri"`
	VerificationUriComplete string `json:"verification_uri_complete"`
	ExpiresIn               int64  `json:"expires_in"`
	ExpiresAt               int64  `json:"expires_at"`
	Interval                int64  `json:"interval"`
}

// ErrorResponse
// specification: RFC6749
type ErrorResponse struct {
	StatusCode       int    `json:"status_code"`
	Error            string `json:"error"`
	ErrorDescription string `json:"error_description"`
	ErrorUri         string `json:"error_uri"`
	RequestId        string `json:"request_id"`
}

// FormatMessage renders a human-readable, actionable message for an OAuth2 error response,
// tolerating empty descriptions and appending a hint for known error codes.
func (e *ErrorResponse) FormatMessage() string {
	if e == nil {
		return "unknown error (empty error response)"
	}
	code := e.Error
	if code == "" {
		code = "unknown_error"
	}
	msg := code
	if e.ErrorDescription != "" {
		msg += ": " + e.ErrorDescription
	}
	var meta []string
	if e.StatusCode != 0 {
		meta = append(meta, fmt.Sprintf("status=%d", e.StatusCode))
	}
	if e.RequestId != "" {
		meta = append(meta, "request_id="+e.RequestId)
	}
	if len(meta) > 0 {
		msg += " (" + strings.Join(meta, ", ") + ")"
	}
	if hint := errorHint(code, e.ErrorDescription); hint != "" {
		msg += "\n  hint: " + hint
	}
	return msg
}

// errorHint maps known IDaaS OAuth2 errors to actionable guidance.
// It inspects the description first (generic codes like invalid_request carry the real meaning there).
func errorHint(code, description string) string {
	if strings.Contains(description, "has not been authorized by resource server") {
		return "broker 客户端应用未被授权到目标资源服务器。请让管理员为该应用（--client-id）配置到资源服务器 urn:cloud:idaas:pam 的 M2M 委派授权（delegated scope: cloud_account_role:obtain_access_credential），再重试 init。"
	}
	switch code {
	case "operation_denied_by_license":
		return "该操作被实例 License 限制：常见于收费应用运行在免费/能力扩展版实例上被禁用，或应用的 M2M 能力未开通。请确认实例 License 版本与应用授权，或联系管理员调整 License / 开通对应能力。"
	case "invalid_client":
		return "客户端无效或未启用所需授权类型。请检查 --client-id 是否正确、应用是否为公共客户端并已开启 device_code 授权。"
	case "invalid_scope", "scope_not_found":
		return "scope 无效或客户端未被授权到目标资源服务器。请确认应用已被委派到 PAM（urn:cloud:idaas:pam）的 cloud_account_role:obtain_access_credential。"
	case "access_denied":
		return "用户拒绝授权或无访问权限。"
	case "expired_token":
		return "设备码已过期，请重新执行 init 登录。"
	}
	return ""
}

// OpenIdConfiguration
// specification: https://openid.net/specs/openid-connect-discovery-1_0.html
type OpenIdConfiguration struct {
	Issuer                            string   `json:"issuer"`
	AuthorizationEndpoint             string   `json:"authorization_endpoint"`
	DeviceAuthorizationEndpoint       string   `json:"device_authorization_endpoint"`
	TokenEndpoint                     string   `json:"token_endpoint"`
	UserinfoEndpoint                  string   `json:"userinfo_endpoint"`
	RevocationEndpoint                string   `json:"revocation_endpoint"`
	JwksUri                           string   `json:"jwks_uri"`
	ScopesSupported                   []string `json:"scopes_supported"`
	ResponseTypesSupported            []string `json:"response_types_supported"`
	SubjectTypesSupported             []string `json:"subject_types_supported"`
	IdTokenSigningAlgValuesSupported  []string `json:"id_token_signing_alg_values_supported"`
	ClaimsSupported                   []string `json:"claims_supported"`
	CodeChallengeMethodsSupported     []string `json:"code_challenge_methods_supported"`
	TokenEndpointAuthMethodsSupported []string `json:"token_endpoint_auth_methods_supported"`
	GrantTypesSupported               []string `json:"grant_types_supported"`
	ResponseModesSupported            []string `json:"response_modes_supported"`
	RequestUriParameterSupported      bool     `json:"request_uri_parameter_supported"`
}

type FetchTokenOptions struct {
	// for RFC6749
	ClientId     string
	ClientSecret string
	GrantType    string
	Scope        string
	RefreshToken string

	// for RFC8628
	DeviceCode string

	// for RFC7523
	ClientAssertionType string
	ClientAssertion     string

	// for RFC8693 OAuth 2.0 Token Exchange
	SubjectTokenType   string
	SubjectToken       string
	RequestedTokenType string

	// for Alibaba Cloud IDaaS Identity Anywhere
	ClientX509                         string
	ClientX509Chain                    string
	ApplicationFederatedCredentialName string
}

type FetchOpenIdConfigurationOptions struct {
	ForceNew bool
}

// FetchToken
// specifications:
// - RFC6749
// - RFC8628
// - RFC7523
// - RFC8693
func FetchToken(tokenEndpoint string, options *FetchTokenOptions) (*TokenResponse, *ErrorResponse, error) {
	statusCode, tokenResponse, errorResponse, err := innerFetchToken(tokenEndpoint, options)
	isServerError := statusCode >= 500 && statusCode < 600
	if isServerError {
		idaaslog.Error.PrintfLn(
			"server error in fetching token, try more once, status code: %d, token response: %v, error response: %v, err: %v",
			statusCode, tokenResponse, errorResponse, err)
		time.Sleep(100 * time.Millisecond)
		statusCode, tokenResponse, errorResponse, err = innerFetchToken(tokenEndpoint, options)
		idaaslog.Info.PrintfLn("retry fetch token status code: %d", statusCode)
	}
	return tokenResponse, errorResponse, err
}

func innerFetchToken(tokenEndpoint string, options *FetchTokenOptions) (int, *TokenResponse, *ErrorResponse, error) {
	parameter := map[string]string{}
	parameter["client_id"] = options.ClientId
	if options.ClientSecret != "" {
		parameter["client_secret"] = options.ClientSecret
	}
	if options.GrantType != "" {
		parameter["grant_type"] = options.GrantType
	}
	if options.DeviceCode != "" {
		parameter["device_code"] = options.DeviceCode
	}
	if options.Scope != "" {
		parameter["scope"] = options.Scope
	}
	if options.SubjectTokenType != "" {
		parameter["subject_token_type"] = options.SubjectTokenType
	}
	if options.SubjectToken != "" {
		parameter["subject_token"] = options.SubjectToken
	}
	if options.RequestedTokenType != "" {
		parameter["requested_token_type"] = options.RequestedTokenType
	}
	if options.RefreshToken != "" {
		parameter["refresh_token"] = options.RefreshToken
	}
	if options.ClientAssertionType != "" {
		parameter["client_assertion_type"] = options.ClientAssertionType
	}
	if options.ClientAssertion != "" {
		parameter["client_assertion"] = options.ClientAssertion
	}
	if options.ClientX509 != "" {
		parameter["client_x509"] = options.ClientX509
	}
	if options.ClientX509Chain != "" {
		parameter["client_x509_chain"] = options.ClientX509Chain
	}
	if options.ApplicationFederatedCredentialName != "" {
		parameter["application_federated_credential_name"] = options.ApplicationFederatedCredentialName
	}
	idaaslog.Unsafe.PrintfLn("Fetch token: %s, with parameter: %+v", tokenEndpoint, parameter)
	statusCode, token, err := utils.PostHttp(tokenEndpoint, parameter)
	if err != nil {
		idaaslog.Error.PrintfLn("Failed to fetch token, error: %v", err)
		return statusCode, nil, nil, errors.Wrapf(err, "failed to fetch token from: %s", tokenEndpoint)
	}
	if statusCode != http.StatusOK {
		idaaslog.Error.PrintfLn("Failed to fetch token, status: %d", statusCode)
		idaaslog.Unsafe.PrintfLn("Failed fetch status code: %d, response: %s", statusCode, token)
		errorResponse, err := parseErrorResponse(statusCode, token)
		if err != nil {
			return statusCode, nil, nil, errors.Wrapf(err, "failed to parse error response: %s", token)
		}
		return statusCode, nil, errorResponse, nil
	}
	idaaslog.Unsafe.PrintfLn("Successfully fetched token: %s", token)
	var tokenResponse TokenResponse
	err = json.Unmarshal([]byte(token), &tokenResponse)
	if err != nil {
		return statusCode, nil, nil, errors.Wrapf(err, "failed to unmarshal token response: %s", token)
	}
	return statusCode, &tokenResponse, nil, nil
}

// FetchOpenIdConfiguration
// specification: https://openid.net/specs/openid-connect-discovery-1_0.html
func FetchOpenIdConfiguration(issuer string, fetchOptions *FetchOpenIdConfigurationOptions) (*OpenIdConfiguration, error) {
	discovery := issuer + "/.well-known/openid-configuration"
	idaaslog.Info.PrintfLn("OIDC discovery URL: %s", discovery)
	options := &utils.ReadCacheOptions{
		Context: map[string]interface{}{
			"issuer":    issuer,
			"discovery": discovery,
		},
		FetchContent: func() (int, string, error) {
			idaaslog.Debug.PrintfLn("GET discovery from URL: %s", discovery)
			return utils.GetHttp(discovery)
		},
		// OpenID configuration allows expired
		AllowExpired: true,
		ForceNew:     fetchOptions.ForceNew,
	}
	cacheKey := utils.Sha256ToHex(issuer)
	openIdConfigurationJson, err := utils.ReadCacheFileWithEncryptionCallback(constants.CategoryOidc, cacheKey, options)
	if err != nil {
		idaaslog.Error.PrintfLn("Failed to fetch OpenID configuration, error: %v", err)
		return nil, errors.Wrap(err, "read cache file with encryption callback")
	}
	idaaslog.Debug.PrintfLn("OpenID configuration: %s", openIdConfigurationJson)
	var openIdConfiguration OpenIdConfiguration
	err = json.Unmarshal([]byte(openIdConfigurationJson), &openIdConfiguration)
	if err != nil {
		idaaslog.Error.PrintfLn("Parse OpenID configuration %s, error: %v", openIdConfigurationJson, err)
		return nil, errors.Wrap(err, "parse OpenID configuration")
	}
	return &openIdConfiguration, nil
}

func parseErrorResponse(statusCode int, response string) (*ErrorResponse, error) {
	var errorResponse ErrorResponse
	err := json.Unmarshal([]byte(response), &errorResponse)
	if err != nil {
		idaaslog.Error.PrintfLn("Failed to parse error response: %s, error: %v", response, err)
		return nil, errors.Wrapf(err, "failed to unmarshal error response: %s", response)
	}
	errorResponse.StatusCode = statusCode
	return &errorResponse, nil
}
