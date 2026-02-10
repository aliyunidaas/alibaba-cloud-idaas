package serve

import (
	"net/http"

	"github.com/aliyunidaas/alibaba-cloud-idaas/cloud"
	"github.com/aliyunidaas/alibaba-cloud-idaas/cloud/alibaba_cloud"
	"github.com/aliyunidaas/alibaba-cloud-idaas/cloud/aws"
	"github.com/aliyunidaas/alibaba-cloud-idaas/cloud/cloud_account"
)

func handleCloudToken(w http.ResponseWriter, r *http.Request, serveOptions *HttpServeOptions) {
	if !isRequestAllowed(w, r, serveOptions) {
		return
	}
	query := r.URL.Query()

	// TODO memory cache
	profile := query.Get("profile")
	forceNew := query.Get("force-new")
	forceNewCloudToken := query.Get("force-new-cloud-token")

	options := &cloud.FetchCloudStsOptions{
		ForceNew:               forceNew == "true",
		ForceNewCloudToken:     forceNewCloudToken == "true",
		IgnoreParseFromProfile: true,
	}
	sts, _, err := cloud.FetchCloudStsFromDefaultConfig("", profile, options)
	if err != nil {
		printResponse(w, http.StatusInternalServerError, ErrorResponse{
			Error:   "internal_error",
			Message: "Fetch cloud sts token failed.",
		})
		return
	}

	alibabaCloudSts, ok := sts.(*alibaba_cloud.StsToken)
	if ok {
		printResponse(w, http.StatusOK, alibabaCloudSts.ConvertToCredentialsUri())
		return
	}

	cloudAccountToken, ok := sts.(*cloud_account.CloudAccountToken)
	if ok {
		if cloudAccountToken.IsAlibabaCloudToken() {
			alibabaCloudSts := cloud.ConvertCloudAccountTokenAlibabaCloudStsTokenToAlibabaStsToken(cloudAccountToken.CloudAccountRoleAccessCredential.AlibabaCloudStsToken)
			printResponse(w, http.StatusOK, alibabaCloudSts)
			return
		}
		// TODO handle other token
	}

	_, ok = sts.(*aws.AwsStsToken)
	if ok {
		printResponse(w, http.StatusNotImplemented, ErrorResponse{
			Error:   "not_implemented",
			Message: "AWS sts token sts not implemented.",
		})
		return
	}

	printResponse(w, http.StatusInternalServerError, ErrorResponse{
		Error:   "bad_request",
		Message: "Unknown cloud sts token.",
	})
	return
}
