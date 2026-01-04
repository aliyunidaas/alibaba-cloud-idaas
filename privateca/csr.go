package privateca

import (
	"crypto"
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"fmt"
	"io"

	"github.com/aliyunidaas/alibaba-cloud-idaas/signer"
	"github.com/aliyunidaas/alibaba-cloud-idaas/signer/external"
	"github.com/pkg/errors"
)

type ExCommandSignerWrap struct {
	ExCmdSigner signer.ExSigner
}

func NewExCommandSignerWrap(exSigner signer.ExSigner) *ExCommandSignerWrap {
	return &ExCommandSignerWrap{
		ExCmdSigner: exSigner,
	}
}

func (w *ExCommandSignerWrap) Public() crypto.PublicKey {
	publicKey, err := w.ExCmdSigner.Public()
	if err != nil {
		panic(fmt.Sprintf("error getting public key: %v", err))
	}
	return publicKey
}

func (w *ExCommandSignerWrap) Sign(rand io.Reader, digest []byte, opts crypto.SignerOpts) ([]byte, error) {
	if opts != nil && opts.HashFunc().String() != "SHA-256" {
		return nil, errors.Errorf("unsupported hash function: %s", opts.HashFunc().String())
	}
	return w.ExCmdSigner.SignDigest(rand, signer.ES256, digest)
}

// SignCertificateRequest
// aliyun cas CreateCustomCertificate \
// --region cn-hangzhou \
// --Csr='-----BEGIN CERTIFICATE REQUEST----- MII****** -----END CERTIFICATE REQUEST-----' \
// --ParentIdentifier '1f06093c-****-****-****-************' \
// --Validity 7d \
// --ApiPassthrough.Extensions.KeyUsage.DigitalSignature true \
// --ApiPassthrough.Extensions.ExtendedKeyUsages.1 clientAuth \
// --version 2020-06-30 \
// --force
func SignCertificateRequest(command, parameter string) (string, error) {
	subject := pkix.Name{
		Organization:       []string{"Example Inc"},
		OrganizationalUnit: []string{"IT"},
		CommonName:         "common-name-example",
	}

	template := x509.CertificateRequest{
		Subject:            subject,
		SignatureAlgorithm: x509.ECDSAWithSHA256,
	}

	exCommandSigner, err := external.NewExCommandSigner(command, parameter)
	if err != nil {
		return "", errors.Wrapf(err, "error creating new ex command signer")
	}

	cryptoSigner := NewExCommandSignerWrap(exCommandSigner)

	csrBytes, err := x509.CreateCertificateRequest(rand.Reader, &template, cryptoSigner)
	if err != nil {
		return "", errors.Wrapf(err, "error creating new certificate request")
	}

	csrPEM := pem.EncodeToMemory(&pem.Block{
		Type:  "CERTIFICATE REQUEST",
		Bytes: csrBytes,
	})

	return string(csrPEM), nil
}
