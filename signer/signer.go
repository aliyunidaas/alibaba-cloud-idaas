package signer

import (
	"crypto"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"time"

	"github.com/aliyunidaas/alibaba-cloud-idaas/idaaslog"
	"github.com/aliyunidaas/alibaba-cloud-idaas/utils"
	"github.com/pkg/errors"
)

type JwtSignAlgorithm uint

const (
	RS256 JwtSignAlgorithm = 1 + iota
	RS384
	RS512
	ES256
	ES384
	ES512
)

func (a JwtSignAlgorithm) IsRsa() bool {
	return a == RS256 || a == RS384 || a == RS512
}

func (a JwtSignAlgorithm) IsEcc() bool {
	return a == ES256 || a == ES384 || a == ES512
}

func (a JwtSignAlgorithm) ToString() string {
	switch a {
	case RS256:
		return "RS256"
	case RS384:
		return "RS384"
	case RS512:
		return "RS512"
	case ES256:
		return "ES256"
	case ES384:
		return "ES384"
	case ES512:
		return "ES512"
	default:
		panic("SHOULD NOT HAPPEN")
	}
}

func (a JwtSignAlgorithm) GetHash() crypto.Hash {
	switch a {
	case ES256:
		return crypto.SHA256
	case RS256:
		return crypto.SHA256
	case ES384:
		return crypto.SHA384
	case RS384:
		return crypto.SHA384
	case ES512:
		return crypto.SHA512
	case RS512:
		return crypto.SHA512
	default:
		panic("SHOULD NOT HAPPEN")
	}
}

func (a JwtSignAlgorithm) GetHashStrName() string {
	switch a {
	case ES256:
		return "sha256"
	case RS256:
		return "sha256"
	case ES384:
		return "sha384"
	case RS384:
		return "sha384"
	case ES512:
		return "sha512"
	case RS512:
		return "sha512"
	default:
		panic("SHOULD NOT HAPPEN")
	}
}

func ParseJwtSignAlgorithm(s string) (JwtSignAlgorithm, error) {
	switch s {
	case "RS256":
		return RS256, nil
	case "RS384":
		return RS384, nil
	case "RS512":
		return RS512, nil
	case "ES256":
		return ES256, nil
	case "ES384":
		return ES384, nil
	case "ES512":
		return ES512, nil
	default:
		return 0, errors.Errorf("unsupported sign algorithm %s", s)
	}
}

type ExSigner interface {
	Public() (crypto.PublicKey, error)

	Sign(rand io.Reader, alg JwtSignAlgorithm, message []byte) ([]byte, error)

	SignDigest(rand io.Reader, alg JwtSignAlgorithm, digest []byte) ([]byte, error)
}

type JwtSigner interface {
	SignJwtWithOptions(claim map[string]interface{}, options *JwtSignerOptions) (string, error)

	SignJwt(header, claim map[string]interface{}) (string, error)
}

type JwtSignerOptions struct {
	Type      string
	Issuer    string
	Audience  string
	Subject   string
	AutoJti   bool
	CustomJti func() string
	Validity  time.Duration
	Header    map[string]interface{}
}

type ExJwtSigner struct {
	keyID  string
	alg    JwtSignAlgorithm
	singer ExSigner
}

func NewExJwtSigner(keyID string, alg JwtSignAlgorithm, signer ExSigner) *ExJwtSigner {
	return &ExJwtSigner{
		keyID:  keyID,
		alg:    alg,
		singer: signer,
	}
}

func (s *ExJwtSigner) SignJwtWithOptions(claim map[string]interface{}, options *JwtSignerOptions) (string, error) {
	if options == nil {
		idaaslog.Error.PrintfLn("SignJwtWithOptions: options is nil!")
		return "", errors.New("options cannot be nil")
	}
	header := options.Header
	if header == nil {
		idaaslog.Debug.PrintfLn("SignJwtWithOptions: header is nil, use empty")
		header = map[string]interface{}{}
	}
	if claim == nil {
		idaaslog.Debug.PrintfLn("SignJwtWithOptions: claim is nil, use empty")
		claim = map[string]interface{}{}
	}
	header["alg"] = s.alg.ToString()
	if options.Type == "" {
		header["typ"] = "JWT"
	} else {
		header["typ"] = options.Type
	}
	claim["iss"] = options.Issuer
	claim["aud"] = options.Audience
	claim["sub"] = options.Subject
	if options.AutoJti {
		claim["jti"] = generateJti()
	} else if options.CustomJti != nil {
		claim["jti"] = options.CustomJti()
	}
	nowSeconds := time.Now().Unix()
	claim["iat"] = nowSeconds
	claim["exp"] = nowSeconds + int64(options.Validity.Seconds())

	idaaslog.Debug.PrintfLn("SignJwtWithOptions: claim is %#v", claim)
	return s.SignJwt(header, claim)
}

func (s *ExJwtSigner) SignJwt(header, claim map[string]interface{}) (string, error) {
	if s.keyID != "" {
		header["kid"] = s.keyID
	}
	headerJson, err := json.Marshal(header)
	if err != nil {
		return "", errors.Wrap(err, "failed to marshal header")
	}
	claimJson, err := json.Marshal(claim)
	if err != nil {
		return "", errors.Wrap(err, "failed to marshal claim")
	}
	headerBase64Str := base64.RawURLEncoding.EncodeToString(headerJson)
	claimBase64Str := base64.RawURLEncoding.EncodeToString(claimJson)
	headerAndClaim := headerBase64Str + "." + claimBase64Str
	idaaslog.Debug.PrintfLn("SignJwt: header.claim is %#v", headerAndClaim)
	signature, err := s.singer.Sign(rand.Reader, s.alg, []byte(headerAndClaim))
	if err != nil {
		return "", errors.Wrapf(err, "sign header failed with header %s", headerAndClaim)
	}
	if s.alg.IsEcc() {
		idaaslog.Debug.PrintfLn("SignJwt: signing JWT signature is ecc, convert to RS")
		signature, err = utils.ParseECDSASignatureToRs(signature)
		if err != nil {
			return "", errors.Wrapf(err, "parse ecdsa signature fail: %x", signature)
		}
	}
	signatureBase64Str := base64.RawURLEncoding.EncodeToString(signature)
	jwt := headerAndClaim + "." + signatureBase64Str
	idaaslog.Debug.PrintfLn("SignJwt: jwt is %s", jwt)
	return jwt, nil
}

func (s *ExJwtSigner) GetExtSinger() ExSigner {
	return s.singer
}

func generateJti() string {
	nonce := make([]byte, 12)
	// IGNORE ERROR
	_, _ = io.ReadFull(rand.Reader, nonce)
	jti := fmt.Sprintf("jti-%s-%d", base64.RawURLEncoding.EncodeToString(nonce), time.Now().UnixMilli())
	idaaslog.Debug.PrintfLn("generateJti: jti is %s", jti)
	return jti
}
