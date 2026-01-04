//go:build disable_yubikey_piv
// +build disable_yubikey_piv

package yubikey_piv

import (
	"crypto"
	"io"

	"github.com/aliyunidaas/alibaba-cloud-idaas/signer"
	"github.com/pkg/errors"
)

func YubiKeyPivSingerEnabled() bool {
	return false
}

type YubiKeyPivSigner struct {
}

func NewYubiKeyPivSigner(slotId, pin, pinPolicy string) (*YubiKeyPivSigner, error) {
	return nil, errors.New("YubiKey PIV is not enabled")
}

func (s *YubiKeyPivSigner) Public() (crypto.PublicKey, error) {
	return nil, nil
}

func (s *YubiKeyPivSigner) Sign(rand io.Reader, alg signer.JwtSignAlgorithm, message []byte) ([]byte, error) {
	return nil, nil
}

func (s *YubiKeyPivSigner) SignDigest(rand io.Reader, alg signer.JwtSignAlgorithm, digest []byte) ([]byte, error) {
	return nil, nil
}
