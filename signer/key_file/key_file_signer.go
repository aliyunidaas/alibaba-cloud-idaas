package key_file

import (
	"crypto"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rsa"
	"crypto/x509"
	"encoding/base64"
	"encoding/pem"
	"io"
	"os"
	"strings"

	"github.com/aliyunidaas/alibaba-cloud-idaas/constants"
	"github.com/aliyunidaas/alibaba-cloud-idaas/signer"
	"github.com/pkg/errors"
	"github.com/youmark/pkcs8"
)

type KeyFileSigner struct {
	key      string
	file     string
	password string
}

func NewKeyFileSigner(key, file string, password string) (*KeyFileSigner, error) {
	if key == "" && file == "" {
		return nil, errors.New("key and file requires one")
	}
	if key != "" && file != "" {
		return nil, errors.New("only one of key or file can be specified")
	}
	return &KeyFileSigner{
		key:      key,
		file:     file,
		password: password,
	}, nil
}

func (s *KeyFileSigner) Public() (crypto.PublicKey, error) {
	privateKey, err := s.loadKey()
	if err != nil {
		return nil, err
	}
	switch key := privateKey.(type) {
	case *rsa.PrivateKey:
		return key.Public(), nil
	case *ecdsa.PrivateKey:
		return key.Public(), nil
	default:
		return nil, errors.Errorf("unsupported private key type: %T", privateKey)
	}
}

func (s *KeyFileSigner) Sign(rand io.Reader, alg signer.JwtSignAlgorithm, message []byte) ([]byte, error) {
	hash := alg.GetHash()
	hasher := hash.New()
	_, err := hasher.Write(message)
	if err != nil {
		return nil, errors.Wrap(err, "failed to hash message")
	}

	hashed := hasher.Sum(message[:0])

	return s.SignDigest(rand, alg, hashed)
}

func (s *KeyFileSigner) SignDigest(rand io.Reader, alg signer.JwtSignAlgorithm, digest []byte) ([]byte, error) {
	privateKey, err := s.loadKey()
	if err != nil {
		return nil, err
	}

	hash := alg.GetHash()
	if len(digest) != hash.Size() {
		return nil, errors.Errorf("Algorithm: %s requires digest length: %d, provided length: %d",
			alg.GetHashStrName(), hash.Size(), len(digest))
	}

	var signature []byte
	var signErr error
	switch key := privateKey.(type) {
	case *rsa.PrivateKey:
		if !alg.IsRsa() {
			return nil, errors.Errorf("unsupported algorithm: %s", alg.ToString())
		}
		signature, signErr = key.Sign(rand, digest, hash)
	case *ecdsa.PrivateKey:
		if !alg.IsEcc() {
			return nil, errors.Errorf("unsupported algorithm: %s", alg.ToString())
		}
		if alg == signer.ES256 && key.Params().Name != elliptic.P256().Params().Name {
			return nil, errors.Errorf("unsupported algorithm: %s for curve: %s", alg.ToString(), key.Params().Name)
		}
		if alg == signer.ES384 && key.Params().Name != elliptic.P384().Params().Name {
			return nil, errors.Errorf("unsupported algorithm: %s for curve: %s", alg.ToString(), key.Params().Name)
		}
		if alg == signer.ES512 && key.Params().Name != elliptic.P521().Params().Name {
			return nil, errors.Errorf("unsupported algorithm: %s for curve: %s", alg.ToString(), key.Params().Name)
		}
		signature, signErr = key.Sign(rand, digest, hash)
	default:
		return nil, errors.Errorf("unsupported private key type: %T", privateKey)
	}
	if signErr != nil {
		return nil, errors.Wrap(signErr, "failed to sign message")
	}
	return signature, nil
}

func (s *KeyFileSigner) loadKey() (crypto.PrivateKey, error) {
	content, err := loadKeyOrFile(s.key, s.file)
	if err != nil {
		return nil, err
	}
	var privateKeyBytes []byte
	if strings.Contains(content, "-----") {
		// must be PEM format
		block, _ := pem.Decode([]byte(content))
		if block.Type == "ENCRYPTED PRIVATE KEY" {
			password := s.password
			if password == "" {
				envPkcs8Password := os.Getenv(constants.EnvPkcs8Password)
				if envPkcs8Password == "" {
					return nil, errors.New("encrypted private key requires password")
				}
				password = envPkcs8Password
			}

			privateKey, err := pkcs8.ParsePKCS8PrivateKey(block.Bytes, []byte(password))
			if err != nil {
				return nil, errors.Wrap(err, "failed to parse PKCS8 private key")
			}
			return privateKey, nil
		}
		privateKeyBytes = block.Bytes
	} else {
		// try base64 format, must be plaintext key
		privateKeyBytes, err = base64.StdEncoding.DecodeString(content)
		if err != nil {
			return nil, errors.Wrap(err, "failed to base64 decode private key")
		}
	}
	privateKey, err := parsePrivateKey(privateKeyBytes)
	if err != nil {
		return nil, errors.Wrap(err, "failed to parse private key")
	}
	return privateKey, nil
}

func parsePrivateKey(derData []byte) (crypto.PrivateKey, error) {
	key, err := x509.ParsePKCS8PrivateKey(derData)
	if err == nil {
		return key, nil
	}

	rsaKey, err2 := x509.ParsePKCS1PrivateKey(derData)
	if err2 == nil {
		return rsaKey, nil
	}

	ecdsaKey, err3 := x509.ParseECPrivateKey(derData)
	if err3 == nil {
		return ecdsaKey, nil
	}

	return nil, errors.Errorf("unable to parse private key: %v, %v, %v", err, err2, err3)
}

func loadKeyOrFile(key, file string) (string, error) {
	if key == "" {
		return loadFile(file)
	}
	return key, nil
}

func loadFile(file string) (string, error) {
	content, err := os.ReadFile(file)
	if err != nil {
		return "", errors.Wrapf(err, "failed to read file %s", file)
	}
	return string(content), nil
}
