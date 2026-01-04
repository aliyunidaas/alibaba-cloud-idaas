package cloud_common

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"strings"
	"time"

	"github.com/aliyunidaas/alibaba-cloud-idaas/idaaslog"
	"github.com/aliyunidaas/alibaba-cloud-idaas/idp"
)

// GenerateRoleSessionName
// 格式：包含英文字母、数字、半角句号（.）、at（@）、短划线（-）和下划线（_）。
// 长度：2~64 个字符。
// https://api.aliyun.com/document/Sts/2015-04-01/AssumeRoleWithOIDC
// Pattern: [\w+=,.@-]*
// Length Constraints: Minimum length of 2. Maximum length of 64.
// https://docs.aws.amazon.com/STS/latest/APIReference/API_AssumeRoleWithWebIdentity.html
func GenerateRoleSessionName(oidcToken string) string {
	prefix := ""
	suffix := fmt.Sprintf("-%d-%s", time.Now().Unix(), randomBytesToString())
	claims, err := idp.ParseJwtTokenClaim(oidcToken)
	if err != nil {
		idaaslog.Debug.PrintfLn("Parse OIDC token failed: %v\n", err)
	} else if claims.Subject != "" {
		prefix = filterRoleSessionNamePrefix(claims.Subject, 64-len(suffix))
	}
	if prefix == "" {
		prefix = "idaas-assumed-role"
	}
	roleSessionName := fmt.Sprintf("%s%s", prefix, suffix)
	idaaslog.Debug.PrintfLn("Generate role session name: %v\n", roleSessionName)
	return roleSessionName
}

func filterRoleSessionNamePrefix(subject string, len int) string {
	var sb strings.Builder
	for _, r := range subject {
		if r >= '0' && r <= '9' {
			sb.WriteRune(r)
		} else if r >= 'a' && r <= 'z' {
			sb.WriteRune(r)
		} else if r >= 'A' && r <= 'Z' {
			sb.WriteRune(r)
		} else if r == '.' || r == '-' || r == '_' || r == '@' {
			sb.WriteRune(r)
		} else {
			// other runes use _ instead
			sb.WriteRune('_')
		}
		if sb.Len() >= len {
			break
		}
	}
	return sb.String()
}

func randomBytesToString() string {
	b := make([]byte, 6)
	_, _ = rand.Read(b)
	return base64.RawURLEncoding.EncodeToString(b)
}
