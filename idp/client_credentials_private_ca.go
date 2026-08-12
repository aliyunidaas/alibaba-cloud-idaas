package idp

import (
	"os"

	"github.com/aliyunidaas/alibaba-cloud-idaas/config"
	"github.com/aliyunidaas/alibaba-cloud-idaas/oidc"
	"github.com/pkg/errors"
)

func FetchAccessTokenClientCredentialsPrivateCa(credentialConfig *config.OidcTokenProviderClientCredentialsConfig, options *oidc.OidcCommonOptions) (*oidc.TokenResponse, error) {
	tokenEndpoint := credentialConfig.TokenEndpoint
	clientAssertionPrivateCaConfig := credentialConfig.ClientAssertionPrivateCaConfig

	certificate, certificateErr := readCertificate(
		clientAssertionPrivateCaConfig.Certificate, clientAssertionPrivateCaConfig.CertificateFile)
	if certificateErr != nil {
		return nil, certificateErr
	}
	certificateChain, certificateChainErr := readCertificate(
		clientAssertionPrivateCaConfig.CertificateChain, clientAssertionPrivateCaConfig.CertificateChainFile)
	if certificateChainErr != nil {
		return nil, certificateChainErr
	}

	jwtSigner, err := config.NewExJwtSignerFromConfig(clientAssertionPrivateCaConfig.CertificateKeySigner)
	if err != nil {
		return nil, errors.Wrap(err, "new jwt signer failed")
	}
	fetchTokenX509JwtBearerOptions := &oidc.FetchTokenX509JwtBearerOptions{
		FetchTokenCommonOptions: buildFetchTokenCommonOptions(credentialConfig, options),
		ClientX509:              certificate,
		ClientX509Chain:         certificateChain,
		JwtSigner:               jwtSigner,
	}
	tokenResponse, errorResponse, err := oidc.FetchTokenX509JwtBearer(tokenEndpoint, fetchTokenX509JwtBearerOptions)
	return parseFetchAccessToken(tokenResponse, errorResponse, err)
}

func readCertificate(certificate, certificateFile string) (string, error) {
	if certificate != "" {
		return certificate, nil
	}
	if certificateFile != "" {
		content, err := os.ReadFile(certificateFile)
		if err != nil {
			return "", errors.Wrapf(err, "read certificate file %s failed", certificateFile)
		}
		return string(content), nil
	}
	return "", errors.New("no certificate provided")
}
