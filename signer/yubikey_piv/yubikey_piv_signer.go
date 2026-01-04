//go:build !disable_yubikey_piv
// +build !disable_yubikey_piv

package yubikey_piv

import (
	"crypto"
	"io"
	"log"
	"os"
	"strconv"
	"strings"

	"github.com/aliyunidaas/alibaba-cloud-idaas/constants"
	"github.com/aliyunidaas/alibaba-cloud-idaas/signer"
	"github.com/pkg/errors"

	"github.com/go-piv/piv-go/piv"
)

func YubiKeyPivSingerEnabled() bool {
	return true
}

type YubiKeyPivSigner struct {
	pin       string
	pinPolicy piv.PINPolicy
	slot      piv.Slot
	publicKey crypto.PublicKey
}

func NewYubiKeyPivSigner(slotId, pin, pinPolicy string) (*YubiKeyPivSigner, error) {
	pivSlot, found := parseSlotId(slotId)
	if !found {
		log.Println(pivSlot, found)
		return nil, errors.Errorf("invalid slot id: %s", slotId)
	}
	pivPinPolicy, found := parsePinPolicy(pinPolicy)
	if !found {
		return nil, errors.Errorf("invalid pinPolicy: %s", pinPolicy)
	}
	yubikey, err := findYubiKey()
	if err != nil {
		return nil, err
	}
	defer yubikey.Close()
	cert, err := yubikey.Certificate(pivSlot)
	if err != nil {
		return nil, err
	}
	return &YubiKeyPivSigner{
		pin:       pin,
		pinPolicy: pivPinPolicy,
		slot:      pivSlot,
		publicKey: cert.PublicKey,
	}, nil
}

func (s *YubiKeyPivSigner) Public() (crypto.PublicKey, error) {
	return s.publicKey, nil
}

func (s *YubiKeyPivSigner) Sign(rand io.Reader, alg signer.JwtSignAlgorithm, message []byte) ([]byte, error) {
	hash := alg.GetHash()
	hasher := hash.New()
	_, err := hasher.Write(message)
	if err != nil {
		return nil, errors.Wrap(err, "failed to hash message")
	}

	hashed := hasher.Sum(message[:0])
	return s.SignDigest(rand, alg, hashed)
}

func (s *YubiKeyPivSigner) SignDigest(rand io.Reader, alg signer.JwtSignAlgorithm, digest []byte) ([]byte, error) {
	yubikey, err := findYubiKey()
	if err != nil {
		return nil, errors.Wrap(err, "failed to find YubiKey")
	}
	defer yubikey.Close()
	auth := piv.KeyAuth{
		PIN:       s.pin,
		PINPolicy: s.pinPolicy,
		PINPrompt: func() (string, error) {
			pin := os.Getenv(constants.EnvYubiKeyPin)
			if pin == "" {
				return "", errors.Errorf("%s not set", constants.EnvYubiKeyPin)
			}
			return pin, nil
		},
	}
	priv, err := yubikey.PrivateKey(s.slot, s.publicKey, auth)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get private key, slot: %v", s.slot)
	}
	pivSigner, ok := priv.(crypto.Signer)
	if !ok {
		return nil, errors.Errorf("expected private key to implement crypto.Signer")
	}

	hash := alg.GetHash()
	if len(digest) != hash.Size() {
		return nil, errors.Errorf("Algorithm: %s requires digest length: %d, provided length: %d",
			alg.GetHashStrName(), hash.Size(), len(digest))
	}

	signature, err := pivSigner.Sign(rand, digest, hash)
	if err != nil {
		return nil, errors.Wrap(err, "failed to sign message")
	}
	return signature, nil
}

func findYubiKey() (*piv.YubiKey, error) {
	cards, err := piv.Cards()
	if err != nil {
		return nil, errors.Errorf("unable to open YubiKey: %v", err)
	}
	var yk *piv.YubiKey
	for _, card := range cards {
		if strings.Contains(strings.ToLower(card), "yubikey") {
			if yk, err = piv.Open(card); err != nil {
				return nil, errors.Wrap(err, "failed to open YubiKey")
			}
			break
		}
	}
	if yk == nil {
		return nil, errors.Errorf("yubikey not found, please make sure the key is inserted %v", err)
	}
	return yk, nil
}

func parsePinPolicy(pinPolicy string) (piv.PINPolicy, bool) {
	switch pinPolicy {
	case "never":
		return piv.PINPolicyNever, true
	case "once":
		return piv.PINPolicyOnce, true
	case "always":
		return piv.PINPolicyAlways, true
	}
	return piv.PINPolicyOnce, false
}

func parseSlotId(slotId string) (piv.Slot, bool) {
	switch slotId {
	case "authentication":
	case "auth":
		return piv.SlotAuthentication, true
	case "signature":
	case "sign":
		return piv.SlotSignature, true
	case "cardAuthentication":
	case "cardAuth":
		return piv.SlotCardAuthentication, true
	}
	lowerSlotId := strings.ToLower(slotId)
	if strings.HasPrefix(lowerSlotId, "r") {
		slotIndex := strings.TrimPrefix(lowerSlotId, "r")
		retriedSlot, err := strconv.ParseInt(slotIndex, 10, 32)
		if err == nil {
			return piv.RetiredKeyManagementSlot(uint32(retriedSlot + 0x81))
		}
	}
	return piv.Slot{}, false
}
