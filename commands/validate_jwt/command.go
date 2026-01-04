package validate_jwt

import (
	"crypto"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"strings"
	"time"

	"github.com/aliyunidaas/alibaba-cloud-idaas/utils"
	"github.com/urfave/cli/v2"
)

var (
	stringFlagProfile = &cli.StringFlag{
		Name:     "token",
		Aliases:  []string{"t"},
		Required: true,
		Usage:    "JWT Token",
	}
)

func BuildCommand() *cli.Command {
	flags := []cli.Flag{
		stringFlagProfile,
	}
	return &cli.Command{
		Name:    "validate-jwt",
		Aliases: []string{"jwt"},
		Usage:   "Dead simple JWT token validation tool (IMPORTANT: only supports RS256)",
		Flags:   flags,
		Action: func(context *cli.Context) error {
			token := context.String("token")
			return validateJwt(token)
		},
	}
}

func validateJwt(token string, ) error {
	jwtParts := strings.Split(token, ".")
	if len(jwtParts) != 3 {
		return errors.New("invalid JWT token")
	}

	headerBytes, err := base64.RawURLEncoding.DecodeString(jwtParts[0])
	if err != nil {
		return errors.New("invalid JWT header")
	}
	var header map[string]any
	if err = json.Unmarshal(headerBytes, &header); err != nil {
		return errors.New("invalid JWT header, error: " + err.Error())
	}

	payloadBytes, err := base64.RawURLEncoding.DecodeString(jwtParts[1])
	if err != nil {
		return errors.New("invalid JWT payload")
	}
	var payload map[string]any
	if err = json.Unmarshal(payloadBytes, &payload); err != nil {
		return errors.New("invalid JWT payload, error: " + err.Error())
	}

	headerJsonBytes, err := json.MarshalIndent(&header, "", "  ")
	if err == nil {
		fmt.Println(string(headerJsonBytes))
	}
	payloadJsonBytes, err := json.MarshalIndent(&payload, "", "  ")
	if err == nil {
		fmt.Println(string(payloadJsonBytes))
	}

	issuer := payload["iss"].(string)
	fmt.Printf("Issuer: %s\n", issuer)

	oidcDiscoveryUrl := issuer + "/.well-known/openid-configuration"
	fmt.Printf("Fetch OIDC discovery: %s\n", oidcDiscoveryUrl)
	ret, oidcDiscoveryJson, err := utils.GetHttp(oidcDiscoveryUrl)
	if err != nil {
		return errors.New("cannot fetch OIDC discovery, error: " + err.Error())
	}
	if ret != 200 {
		return errors.New(fmt.Sprintf("cannot fetch OIDC discovery, status: %d", ret))
	}
	var oidcDiscovery map[string]any
	if err = json.Unmarshal([]byte(oidcDiscoveryJson), &oidcDiscovery); err != nil {
		return errors.New("invalid OIDC discovery, error: " + err.Error())
	}
	jwksUri := oidcDiscovery["jwks_uri"].(string)
	fmt.Printf("Fetch JWKs: %s\n", jwksUri)
	ret, jwksJson, err := utils.GetHttp(jwksUri)
	if err != nil {
		return errors.New("cannot fetch JWKS, error: " + err.Error())
	}
	if ret != 200 {
		return errors.New(fmt.Sprintf("cannot fetch JWKS, status: %d", ret))
	}
	//fmt.Printf("JWKS json: %v\n", jwksJson)
	var jwks Jwks
	if err = json.Unmarshal([]byte(jwksJson), &jwks); err != nil {
		return errors.New("invalid JWKS, error: " + err.Error())
	}

	alg := header["alg"].(string)
	if alg != "RS256" {
		return errors.New("unsupported JWT algorithm: " + alg)
	}

	signKeyId := header["kid"].(string)
	signPublicKey, err := findRsaJwk(&jwks, signKeyId)
	if err != nil {
		return errors.New("cannot find JWK, key ID: " + signKeyId + ", error: " + err.Error())
	}
	if signPublicKey == nil {
		return errors.New("cannot find JWK, key ID: " + signKeyId)
	}

	tobeVerified := jwtParts[0] + "." + jwtParts[1]
	tobeVerifiedSha256 := sha256.Sum256([]byte(tobeVerified))

	signature, err := base64.RawURLEncoding.DecodeString(jwtParts[2])
	if err != nil {
		return errors.New("invalid JWT signature, error: " + err.Error())
	}
	err = rsa.VerifyPKCS1v15(signPublicKey, crypto.SHA256, tobeVerifiedSha256[:], signature)
	if err != nil {
		return errors.New("invalid JWT signature, error: " + err.Error())
	}

	fmt.Printf(utils.Green("[OK] Verify JWT signature success\n", true))

	printJwt(payload)
	return nil
}

func printJwt(payload map[string]any) {
	nowUnixTimestamp := time.Now().Unix()

	isJwtValid := true
	jwtFrom := int64(0)
	jwtEnd := int64(0)

	if payload["iat"] != nil {
		issueAt := payload["iat"].(float64)
		jwtFrom = int64(issueAt)
		issueAtTime := time.Unix(int64(issueAt), 0)
		issuedSecs := nowUnixTimestamp - int64(issueAt)
		fmt.Printf("Issue at        : %s %s\n", issueAtTime.Format(time.RFC3339), utils.WBlue(formatHumanTime(issuedSecs)+" ago"))
	} else {
		fmt.Printf("[WARN] iat not found in JWT payload\n")
	}

	jwtNbfFrom, isJwtNbfValid := printJwtNbf(payload, nowUnixTimestamp)
	if !isJwtNbfValid {
		isJwtValid = false
	}
	if jwtNbfFrom > 0 {
		jwtFrom = jwtNbfFrom
	}

	jwtEnd, isJwtExpValid := printJwtExp(payload, nowUnixTimestamp)
	if !isJwtExpValid {
		isJwtValid = false
	}

	if jwtFrom > 0 && jwtEnd > 0 {
		fmt.Printf("Validity period : %s\n", formatHumanTime(jwtEnd-jwtFrom))
	}

	printJwtValidation(isJwtValid)
}

func printJwtNbf(payload map[string]any, nowUnixTimestamp int64) (int64, bool) {
	isJwtNbfValid := true
	if payload["nbf"] != nil {
		notBefore := payload["nbf"].(float64)
		jwtNbfFrom := int64(notBefore)
		notBeforeTime := time.Unix(int64(notBefore), 0)
		nbfMessage := utils.WGreen("[ok]")
		if int64(notBefore) > nowUnixTimestamp {
			isJwtNbfValid = false
			nbfMessage = utils.WRed("[not yet valid]")
		}
		fmt.Printf("Not before      : %s %s\n", notBeforeTime.Format(time.RFC3339), nbfMessage)
		return jwtNbfFrom, isJwtNbfValid
	}
	return 0, isJwtNbfValid
}

func printJwtExp(payload map[string]any, nowUnixTimestamp int64) (int64, bool) {
	if payload["exp"] != nil {
		expireAt := payload["exp"].(float64)
		jwtEnd := int64(expireAt)
		isJwtExpValid := true
		expireAtTime := time.Unix(int64(expireAt), 0)
		expMessage := utils.WGreen("[ok]")
		if int64(expireAt) <= nowUnixTimestamp {
			isJwtExpValid = false
			expiredSecs := nowUnixTimestamp - int64(expireAt)
			if expiredSecs <= 1 {
				expMessage = utils.WRed("[expired] " + utils.WBold("just now"))
			} else {
				expMessage = utils.WRed("[expired]" + fmt.Sprintf(" %s ago", formatHumanTime(expiredSecs)))
			}
		} else {
			expiringSecs := int64(expireAt) - nowUnixTimestamp
			leftHumanTime := fmt.Sprintf(" left %s", formatHumanTime(expiringSecs))
			if expiringSecs < 60 {
				expMessage = utils.WYellow("[expiring]" + leftHumanTime)
			} else if expiringSecs < 300 {
				expMessage = utils.WBlue("[expiring]" + leftHumanTime)
			} else {
				expMessage = utils.WGreen("[ok]" + leftHumanTime)
			}
		}
		fmt.Printf("Expire at       : %s %s\n", expireAtTime.Format(time.RFC3339), expMessage)
		return jwtEnd, isJwtExpValid
	}
	fmt.Printf("[ERROR] exp not found in JWT payload\n")
	return 0, false
}

func printJwtValidation(isJwtValid bool) {
	if isJwtValid {
		fmt.Printf(utils.WBold(utils.WGreen("[OK] JWT is VALID\n")))
	} else {
		fmt.Printf(utils.WBold(utils.WRed("[ERROR] JWT is INVALID\n")))
	}
}

type Jwks struct {
	Keys []*RsaJwk `json:"keys"`
}
type RsaJwk struct {
	KeyId     string `json:"kid"`
	KeyType   string `json:"kty"`
	Algorithm string `json:"alg"`
	Use       string `json:"use"`
	Exponent  string `json:"e"`
	Modulus   string `json:"n"`
}

func findRsaJwk(jwks *Jwks, keyId string) (*rsa.PublicKey, error) {
	for _, key := range jwks.Keys {
		if key.KeyId == keyId {
			nBytes, err := base64.RawURLEncoding.DecodeString(key.Modulus)
			if err != nil {
				return nil, err
			}
			eBytes, err := base64.RawURLEncoding.DecodeString(key.Exponent)
			if err != nil {
				return nil, err
			}
			n := new(big.Int).SetBytes(nBytes)
			e := new(big.Int).SetBytes(eBytes).Int64()
			return &rsa.PublicKey{
				N: n,
				E: int(e),
			}, nil
		}
	}
	return nil, nil
}

func formatHumanTime(timeInSeconds int64) string {
	if timeInSeconds == 0 {
		return "0 seconds"
	}
	isMinus := timeInSeconds < 0
	if isMinus {
		timeInSeconds = -timeInSeconds
	}
	secs := timeInSeconds % 60
	mins := timeInSeconds / 60 % 60
	hours := (timeInSeconds / 60 / 60) % 24
	days := timeInSeconds / 60 / 60 / 24

	humanTime := ""
	if days > 0 {
		humanTime += fmt.Sprintf(" %d day%s", days, iff(days == 1, "", "s"))
	}
	if hours > 0 {
		humanTime += fmt.Sprintf(" %d hour%s", hours, iff(hours == 1, "", "s"))
	}
	if mins > 0 {
		humanTime += fmt.Sprintf(" %d minute%s", mins, iff(mins == 1, "", "s"))
	}
	if secs > 0 {
		humanTime += fmt.Sprintf(" %d second%s", secs, iff(secs == 1, "", "s"))
	}
	humanTime = humanTime[1:]
	if isMinus {
		return "-" + humanTime
	}
	return humanTime
}

func iff(condition bool, trueVal, falseVal string) string {
	if condition {
		return trueVal
	}
	return falseVal
}
