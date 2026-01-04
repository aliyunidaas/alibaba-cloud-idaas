package oidc

import (
	"time"

	"github.com/aliyunidaas/alibaba-cloud-idaas/signer"
	"github.com/aliyunidaas/alibaba-cloud-idaas/utils"
	"github.com/pkg/errors"
)

type FetchTokenRfc7523Options struct {
	*FetchTokenCommonOptions
	JwtSigner signer.JwtSigner
}

func FetchTokenRfc7523(tokenEndpoint string, options *FetchTokenRfc7523Options) (*TokenResponse, *ErrorResponse, error) {
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
		ClientId:            options.ClientId,
		GrantType:           options.GrantType,
		Scope:               options.Scope,
		ClientAssertionType: ClientAssertionTypeJwtBearer,
		ClientAssertion:     jwtToken,
	}

	return FetchToken(tokenEndpoint, fetchTokenOptions)
}
