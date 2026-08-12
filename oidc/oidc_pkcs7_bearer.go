package oidc

type FetchTokenPkcs7BearerOptions struct {
	*FetchTokenCommonOptions
	Pkcs7 string
}

func FetchTokenPkcs7Bearer(tokenEndpoint string, options *FetchTokenPkcs7BearerOptions) (*TokenResponse, *ErrorResponse, error) {
	fetchTokenOptions := &FetchTokenOptions{
		ClientId:                           options.ClientId,
		GrantType:                          options.GrantType,
		Scope:                              options.Scope,
		SubjectTokenType:                   options.SubjectTokenType,
		SubjectToken:                       options.SubjectToken,
		RequestedTokenType:                 options.RequestedTokenType,
		ClientAssertionType:                ClientAssertionTypePkcs7Bearer,
		ClientAssertion:                    options.Pkcs7,
		ApplicationFederatedCredentialName: options.ApplicationFederatedCredentialName,
	}

	return FetchToken(tokenEndpoint, fetchTokenOptions)
}
