package oidc

import (
	"encoding/base64"
	"encoding/json"
	"strings"
	"time"

	"github.com/aliyunidaas/alibaba-cloud-idaas/idaaslog"
	"github.com/aliyunidaas/alibaba-cloud-idaas/oidc"
	"github.com/pkg/errors"
)

type OidcToken struct {
	IdToken      string `json:"id_token,omitempty"`
	TokenType    string `json:"token_type,omitempty"`
	Scope        string `json:"scope,omitempty"`
	AccessToken  string `json:"access_token,omitempty"`
	RefreshToken string `json:"refresh_token,omitempty"`
	ExpiresIn    int64  `json:"expires_in"`
	ExpiresAt    int64  `json:"expires_at"`
}

type OidcTokenType2 struct {
	IdToken      string    `json:"id_token,omitempty"`
	TokenType    string    `json:"token_type,omitempty"`
	Scope        string    `json:"scope,omitempty"`
	AccessToken  string    `json:"access_token,omitempty"`
	RefreshToken string    `json:"refresh_token,omitempty"`
	ExpiresIn    int64     `json:"expires_in"`
	ExpiresAt    time.Time `json:"expires_at"`
}

type FetchOidcTokenType int

const (
	FetchDefault FetchOidcTokenType = iota
	FetchIdToken
	FetchAccessToken
)

func (t FetchOidcTokenType) IsFetchIdToken() bool {
	return t == FetchIdToken || t == FetchDefault
}

func (t FetchOidcTokenType) IsFetchAccessToken() bool {
	return t == FetchAccessToken || t == FetchDefault
}

func GetOidcTokenType(oidcField string) FetchOidcTokenType {
	if oidcField == oidc.TokenIdToken {
		return FetchIdToken
	} else if oidcField == oidc.TokenAccessToken {
		return FetchAccessToken
	} else {
		return FetchDefault
	}
}

func FromTokenResponse(startTime int64, response *oidc.TokenResponse) (*OidcToken, error) {
	if response == nil {
		return nil, errors.New("oidc response is nil")
	}
	oidcToken := &OidcToken{}
	oidcToken.IdToken = response.IdToken
	oidcToken.TokenType = response.TokenType
	oidcToken.Scope = response.Scope
	oidcToken.AccessToken = response.AccessToken
	oidcToken.RefreshToken = response.RefreshToken
	oidcToken.ExpiresIn = response.ExpiresIn
	if response.ExpiresAt > 0 {
		oidcToken.ExpiresAt = response.ExpiresAt
	} else if response.ExpiresIn > 0 {
		oidcToken.ExpiresAt = startTime + response.ExpiresIn
	}
	return oidcToken, nil
}

func (t *OidcToken) Marshal() (string, error) {
	if t == nil {
		return "null", nil
	}
	tokenBytes, err := json.Marshal(t)
	if err != nil {
		return "", errors.Wrap(err, "marshal OIDC token failed")
	}
	return string(tokenBytes), nil
}

func (t *OidcTokenType2) Marshal() (string, error) {
	if t == nil {
		return "null", nil
	}
	tokenBytes, err := json.Marshal(t)
	if err != nil {
		return "", errors.Wrap(err, "marshal OIDC token failed")
	}
	return string(tokenBytes), nil
}

func UnmarshalOidcToken(token string) (*OidcToken, error) {
	var oidcToken OidcToken
	err := json.Unmarshal([]byte(token), &oidcToken)
	if err != nil {
		return nil, errors.Wrapf(err, "unmarshal OIDC token: %s failed", token)
	}
	return &oidcToken, nil
}

type IdTokenPayload struct {
	Exp int64 `json:"exp"`
}

func ParseIdTokenPayload(idToken string) (*IdTokenPayload, error) {
	idTokenParts := strings.Split(idToken, ".")
	if len(idTokenParts) != 3 {
		return nil, errors.New("invalid ID token")
	}
	idTokenPayloadStr, err := base64.RawURLEncoding.DecodeString(idTokenParts[1])
	if err != nil {
		return nil, errors.New("invalid ID token")
	}
	var idTokenPayload IdTokenPayload
	err = json.Unmarshal(idTokenPayloadStr, &idTokenPayload)
	if err != nil {
		return nil, errors.New("invalid ID token")
	}
	return &idTokenPayload, nil
}

func (t *OidcToken) ConvertToType2() *OidcTokenType2 {
	expireAtUnixTimestamp := t.ExpiresAt
	if expireAtUnixTimestamp == 0 {
		expireAtUnixTimestamp = time.Now().Unix() + t.ExpiresIn - 3
	}
	return &OidcTokenType2{
		IdToken:      t.IdToken,
		TokenType:    t.TokenType,
		Scope:        t.Scope,
		AccessToken:  t.AccessToken,
		RefreshToken: t.RefreshToken,
		ExpiresIn:    t.ExpiresAt,
		ExpiresAt:    time.Unix(expireAtUnixTimestamp, 0),
	}
}

func (t *OidcToken) IsValidAtLeastThreshold(fetchTokenType FetchOidcTokenType, thresholdDuration time.Duration) bool {
	// check ID token
	if t.IdToken != "" && fetchTokenType.IsFetchIdToken() {
		idTokenPayload, err := ParseIdTokenPayload(t.IdToken)
		if err != nil {
			return false
		}
		idaaslog.Debug.PrintfLn("Check ID token is valid, expire at: %s, threshold: %d ms",
			idTokenPayload.Exp, thresholdDuration.Milliseconds())
		if idTokenPayload.Exp == 0 {
			return false
		}
		valid := time.Until(time.Unix(idTokenPayload.Exp, 0)) > thresholdDuration
		idaaslog.Info.PrintfLn("Check ID token is valid: %s", valid)
		if !valid {
			return false
		}
	}
	// check access token
	if t.ExpiresAt > 0 && fetchTokenType.IsFetchAccessToken() {
		idaaslog.Debug.PrintfLn("Check access token is valid, expire at: %s, threshold: %d ms",
			t.ExpiresAt, thresholdDuration.Milliseconds())
		valid := time.Until(time.Unix(t.ExpiresAt, 0)) > thresholdDuration
		idaaslog.Info.PrintfLn("Check access token is valid: %s", valid)
		return valid
	}
	return true
}
