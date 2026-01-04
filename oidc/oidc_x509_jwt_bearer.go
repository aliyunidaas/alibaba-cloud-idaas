package oidc

import (
	"time"

	"github.com/aliyunidaas/alibaba-cloud-idaas/signer"
	"github.com/aliyunidaas/alibaba-cloud-idaas/utils"
	"github.com/pkg/errors"
)

type FetchTokenX509JwtBearerOptions struct {
	*FetchTokenCommonOptions
	ClientX509      string
	ClientX509Chain string
	JwtSigner       signer.JwtSigner
}

func FetchTokenX509JwtBearer(tokenEndpoint string, options *FetchTokenX509JwtBearerOptions) (*TokenResponse, *ErrorResponse, error) {
	jwtSingerOptions := &signer.JwtSignerOptions{
		Issuer:   options.ClientId,
		Audience: options.TokenEndpoint,
		Subject:  options.ClientId,
		Validity: 5 * time.Minute,
		AutoJti:  true,
	}
	utils.Stderr.Println("Ready to sign the JWT token. If required, interact with your security token to proceed.")
	jwtToken, err := options.JwtSigner.SignJwtWithOptions(nil, jwtSingerOptions)
	if err != nil {
		return nil, nil, errors.Wrapf(err, "failed to fetch token from: %s", tokenEndpoint)
	}

	fetchTokenOptions := &FetchTokenOptions{
		ClientId:                           options.ClientId,
		GrantType:                          options.GrantType,
		Scope:                              options.Scope,
		ClientAssertionType:                ClientAssertionTypeX509JwtBearer,
		ClientAssertion:                    jwtToken,
		ClientX509:                         options.ClientX509,
		ClientX509Chain:                    options.ClientX509Chain,
		ApplicationFederatedCredentialName: options.ApplicationFederatedCredentialName,
	}

	return FetchToken(tokenEndpoint, fetchTokenOptions)
}
