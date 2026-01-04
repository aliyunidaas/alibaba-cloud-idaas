package common

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/aliyunidaas/alibaba-cloud-idaas/cloud"
	"github.com/aliyunidaas/alibaba-cloud-idaas/cloud/alibaba_cloud"
	"github.com/aliyunidaas/alibaba-cloud-idaas/cloud/aws"
	"github.com/aliyunidaas/alibaba-cloud-idaas/cloud/cloud_account"
	"github.com/aliyunidaas/alibaba-cloud-idaas/cloud/oidc"
	"github.com/aliyunidaas/alibaba-cloud-idaas/utils"
)

func ShowToken(sts any, oidcTokenType oidc.FetchOidcTokenType, stdout, color bool) error {
	alibabaCloudSts, ok := sts.(*alibaba_cloud.StsToken)
	if ok {
		return showStsToken(alibabaCloudSts, stdout, color)
	}
	awsStsToken, ok := sts.(*aws.AwsStsToken)
	if ok {
		return showAwsStsToken(awsStsToken, stdout, color)
	}
	oidcToken, ok := sts.(*oidc.OidcToken)
	if ok {
		return showOidcToken(oidcToken, oidcTokenType, stdout, color)
	}
	cloudAccountToken, ok := sts.(*cloud_account.CloudAccountToken)
	if ok {
		return showCloudAccountToken(cloudAccountToken, stdout, color)
	}

	return fmt.Errorf("unknown cloud STS token type")
}

func showStsToken(alibabaCloudSts *alibaba_cloud.StsToken, stdout, color bool) error {
	printRow("Access Key ID", alibabaCloudSts.AccessKeyId, stdout, color)
	printRow("Access Key Secret", alibabaCloudSts.AccessKeySecret, stdout, color)
	printRow("Security Token", alibabaCloudSts.StsToken, stdout, color)
	expiration, err := time.Parse(time.RFC3339Nano, alibabaCloudSts.Expiration)
	if err == nil {
		printRowExpiration(&expiration, stdout, color)
	} else {
		printRow("Expiration", alibabaCloudSts.Expiration, stdout, color)
	}
	return nil
}

func showAwsStsToken(awsStsToken *aws.AwsStsToken, stdout, color bool) error {
	printRow("Access Key ID", awsStsToken.AccessKeyId, stdout, color)
	printRow("Secret Access Key", awsStsToken.SecretAccessKey, stdout, color)
	printRow("Session Token", awsStsToken.SessionToken, stdout, color)
	printRowExpiration(&awsStsToken.Expiration, stdout, color)
	return nil
}

func showOidcToken(oidcToken *oidc.OidcToken, oidcTokenType oidc.FetchOidcTokenType, stdout, color bool) error {
	printIdToken := oidcToken.IdToken != "" && oidcTokenType.IsFetchIdToken()
	printAccessToken := oidcToken.AccessToken != "" && oidcTokenType.IsFetchAccessToken()

	if printIdToken {
		printRow("ID Token", oidcToken.IdToken, stdout, color)
		idTokenPayload, err := oidc.ParseIdTokenPayload(oidcToken.IdToken)
		if err == nil {
			expiresAt := time.Unix(idTokenPayload.Exp, 0)
			printRowExpiration(&expiresAt, stdout, color)
		}
	}
	if printIdToken && printAccessToken {
		printStdio("\n", stdout)
	}
	if printAccessToken {
		printRow("Access Token Type", oidcToken.TokenType, stdout, color)
		printRow("Access Token", oidcToken.AccessToken, stdout, color)
		if oidcToken.ExpiresAt > 0 {
			expiresAt := time.Unix(oidcToken.ExpiresAt, 0)
			printRowExpiration(&expiresAt, stdout, color)
		}
	}
	if oidcToken.RefreshToken != "" {
		printRow("Refresh Token", oidcToken.RefreshToken, stdout, color)
	}
	return nil
}

func showCloudAccountToken(cloudAccountToken *cloud_account.CloudAccountToken, stdout, color bool) error {
	printRowWidth2("Cloud Account ID", cloudAccountToken.CloudAccountId, stdout, color)
	printRowWidth2("Cloud Account Role ID", cloudAccountToken.CloudAccountRoleId, stdout, color)
	printRowWidth2("Cloud Account Role Name", cloudAccountToken.CloudAccountRoleName, stdout, color)
	printRowWidth2("Cloud Account Role External ID", cloudAccountToken.CloudAccountRoleExternalId, stdout, color)
	printRowWidth2("Cloud Account Vendor Type", cloudAccountToken.CloudAccountVendorType, stdout, color)
	if cloudAccountToken.CloudAccountRoleAccessCredential != nil {
		credential := cloudAccountToken.CloudAccountRoleAccessCredential
		printRowWidth2("Cloud Account Token Expires At", strconv.FormatInt(credential.AccessCredentialExpiresAt, 10), stdout, color)
		printStdio("", stdout)
		if cloudAccountToken.IsAlibabaCloudToken() {
			stsToken := cloud.ConvertCloudAccountTokenAlibabaCloudStsTokenToAlibabaStsToken(credential.AlibabaCloudStsToken)
			if stsToken != nil {
				_ = showStsToken(stsToken, stdout, color)
			}
		}
	}
	return nil
}

func printRowExpiration(expiration *time.Time, stdout, color bool) {
	nowUnix := time.Now().Unix()
	expiredStatus := ""
	termColor := utils.TermGreen
	if nowUnix >= expiration.Unix() {
		termColor = utils.TermRed
		expiredStatus = "Expired"
	} else {
		leftSeconds := expiration.Unix() - nowUnix
		termColor = getExpirationColor(leftSeconds)
		expiredStatus = fmt.Sprintf("Expires in %d minute(s)", leftSeconds/60)
	}
	if expiredStatus != "" {
		expiredStatus = fmt.Sprintf("   [%s]", expiredStatus)
	}
	printRowWithColor("Expiration", fmt.Sprintf("%s%s", expiration.Local(), expiredStatus), termColor, stdout, color)
}

func getExpirationColor(leftSeconds int64) string {
	termColor := utils.TermGreen
	if leftSeconds < 20*60 {
		termColor = utils.TermRed
	} else if leftSeconds < 30*60 {
		termColor = utils.TermYellow
	}
	return termColor
}

func printRow(header, value string, stdout, color bool) {
	printRowWithWidth(header, value, stdout, color, 18)
}

func printRowWidth2(header, value string, stdout, color bool) {
	printRowWithWidth(header, value, stdout, color, 31)
}

func printRowWithWidth(header, value string, stdout, color bool, width int) {
	var sb strings.Builder
	sb.WriteString(utils.Blue(utils.Bold(
		fmt.Sprintf("%s%s: ", header, stringsRepeat(" ", width-len(header))), color), color))
	sb.WriteString(utils.Green(value, color))
	printStdio(sb.String(), stdout)
}

func printRowWithColor(header, value, termColor string, stdout, color bool) {
	printRowWithColorWithWidth(header, value, termColor, stdout, color, 18)
}

func printRowWithColorWithWidth(header, value, termColor string, stdout, color bool, width int) {
	var sb strings.Builder
	sb.WriteString(utils.Blue(utils.Bold(
		fmt.Sprintf("%s%s: ", header, stringsRepeat(" ", width-len(header))), color), color))
	sb.WriteString(utils.WithColor(value, termColor, color))
	printStdio(sb.String(), stdout)
}

func stringsRepeat(s string, count int) string {
	if count <= 0 {
		return ""
	}
	return strings.Repeat(" ", count)
}

func printStdio(str string, stdout bool) {
	if stdout {
		utils.Stdout.Println(str)
	} else {
		utils.Stderr.Println(str)
	}
}
