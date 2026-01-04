//go:build disable_pkcs11
// +build disable_pkcs11

package pkcs11

import (
	"crypto"
	"io"

	"github.com/aliyunidaas/alibaba-cloud-idaas/signer"
	"github.com/pkg/errors"
)

func Pkcs11SingerEnabled() bool {
	return false
}

type Pkcs11Signer struct {
}

func NewPkcs11Signer(pkcs11LibPath, tokenLabel, pin, keyLabel string) (*Pkcs11Signer, error) {
	return nil, errors.New("PKCS#11 is not enabled")
}

func (s *Pkcs11Signer) Public() (crypto.PublicKey, error) {
	return nil, nil
}

func (s *Pkcs11Signer) Sign(rand io.Reader, alg signer.JwtSignAlgorithm, message []byte) ([]byte, error) {
	return nil, nil
}

func (s *Pkcs11Signer) SignDigest(rand io.Reader, alg signer.JwtSignAlgorithm, digest []byte) ([]byte, error) {
	return nil, nil
}
