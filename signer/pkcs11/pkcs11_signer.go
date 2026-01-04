//go:build !disable_pkcs11
// +build !disable_pkcs11

package pkcs11

import (
	"crypto"
	"io"
	"log"
	"os"

	"github.com/ThalesGroup/crypto11"
	"github.com/aliyunidaas/alibaba-cloud-idaas/constants"
	"github.com/aliyunidaas/alibaba-cloud-idaas/signer"
	"github.com/miekg/pkcs11"
	"github.com/pkg/errors"
)

const (
	MacOsYubikeyLibrary = "/usr/local/lib/libykcs11.dylib"
	MacOsOpenscLibrary  = "/Library/OpenSC/lib/opensc-pkcs11.so"
)

func Pkcs11SingerEnabled() bool {
	return true
}

type Pkcs11Signer struct {
	config   *crypto11.Config
	keyLabel string
}

func NewPkcs11Signer(pkcs11LibPath, tokenLabel, pin, keyLabel string) (*Pkcs11Signer, error) {
	if pkcs11LibPath == "" {
		return nil, errors.New("pkcs11 lib path is empty")
	}
	if tokenLabel == "" {
		return nil, errors.New("pkcs11 token label is empty")
	}
	if keyLabel == "" {
		return nil, errors.New("pkcs11 key label is empty")
	}
	finalPin := pin
	if pin == "" {
		finalPin = os.Getenv(constants.EnvPkcs11Pin)
	}
	config := &crypto11.Config{
		Path:       pkcs11LibPath,
		TokenLabel: tokenLabel,
		Pin:        finalPin,
	}
	return &Pkcs11Signer{
		config:   config,
		keyLabel: keyLabel,
	}, nil
}

func (s *Pkcs11Signer) Public() (crypto.PublicKey, error) {
	ctx, err := crypto11.Configure(s.config)
	if err != nil {
		return nil, errors.Wrap(err, "failed to configure crypto")
	}
	defer ctx.Close()

	privKey, err := ctx.FindKeyPair(nil, []byte(s.keyLabel))
	if err != nil {
		return nil, errors.Wrapf(err, "failed to find private key, key label: %s", s.keyLabel)
	}
	pubKey := privKey.Public()
	return pubKey, nil
}

func (s *Pkcs11Signer) Sign(rand io.Reader, alg signer.JwtSignAlgorithm, message []byte) ([]byte, error) {
	hash := alg.GetHash()
	hasher := hash.New()
	_, err := hasher.Write(message)
	if err != nil {
		return nil, errors.Wrap(err, "failed to hash message")
	}

	hashed := hasher.Sum(message[:0])
	return s.SignDigest(rand, alg, hashed)
}

func (s *Pkcs11Signer) SignDigest(rand io.Reader, alg signer.JwtSignAlgorithm, digest []byte) ([]byte, error) {
	ctx, err := crypto11.Configure(s.config)
	if err != nil {
		return nil, err
	}
	defer ctx.Close()

	pkcsSigner, err := ctx.FindKeyPair(nil, []byte(s.keyLabel))
	if err != nil {
		return nil, err
	}

	hash := alg.GetHash()
	if len(digest) != hash.Size() {
		return nil, errors.Errorf("Algorithm: %s requires digest length: %d, provided length: %d",
			alg.GetHashStrName(), hash.Size(), len(digest))
	}

	signature, err := pkcsSigner.Sign(rand, digest, hash)
	if err != nil {
		return nil, errors.Wrap(err, "failed to sign message")
	}
	return signature, nil
}

func PrintPkcs11Info(module string) error {
	log.Printf("Module: %s\n", module)
	p := pkcs11.New(module)
	err := p.Initialize()
	if err != nil {
		return err
	}
	defer p.Destroy()
	defer p.Finalize()
	slots, err := p.GetSlotList(true)
	if err != nil {
		return errors.Wrap(err, "failed to get slot list")
	}
	if len(slots) == 0 {
		return errors.New("no PKCS#11 slot found")
	}
	firstSlot := slots[0]
	slotInfo, err := p.GetSlotInfo(firstSlot)
	if err != nil {
		return errors.Wrap(err, "failed to get slot info")
	}
	log.Printf("SlotInfo:\n"+
		"- SlotDescription: %s\n"+
		"- ManufacturerID: %s\n"+
		"- Flags: %d\n"+
		"- HardwareVersion: %v\n"+
		"- HardwareVersion: %v",
		slotInfo.SlotDescription,
		slotInfo.ManufacturerID,
		slotInfo.Flags,
		slotInfo.HardwareVersion,
		slotInfo.FirmwareVersion,
	)
	tokenInfo, err := p.GetTokenInfo(firstSlot)
	if err != nil {
		return errors.Wrap(err, "failed to get token info")
	}
	log.Printf("TokenInfo:\n"+
		"- Label: %s\n"+
		"- ManufacturerID: %s\n"+
		"- Model: %s\n"+
		"- SerialNumber: %s\n"+
		"- Flags: %d\n"+
		"- MaxSessionCount: %d\n"+
		"- SessionCount: %d\n"+
		"- MaxRwSessionCount: %d\n"+
		"- RwSessionCount: %d\n"+
		"- MaxPinLen: %d\n"+
		"- MinPinLen: %d\n"+
		"- TotalPublicMemory: %d\n"+
		"- FreePublicMemory: %d\n"+
		"- TotalPrivateMemory: %d\n"+
		"- FreePrivateMemory: %d\n"+
		"- HardwareVersion: %v\n"+
		"- FirmwareVersion: %v\n"+
		"- UTCTime: %s\n",
		tokenInfo.Label,
		tokenInfo.ManufacturerID,
		tokenInfo.Model,
		tokenInfo.SerialNumber,
		tokenInfo.Flags,
		tokenInfo.MaxSessionCount,
		tokenInfo.SessionCount,
		tokenInfo.MaxRwSessionCount,
		tokenInfo.RwSessionCount,
		tokenInfo.MaxPinLen,
		tokenInfo.MinPinLen,
		tokenInfo.TotalPublicMemory,
		tokenInfo.FreePublicMemory,
		tokenInfo.TotalPrivateMemory,
		tokenInfo.FreePrivateMemory,
		tokenInfo.HardwareVersion,
		tokenInfo.FirmwareVersion,
		tokenInfo.UTCTime,
	)
	return nil
}
