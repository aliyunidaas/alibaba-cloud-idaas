package serve

import (
	"encoding/json"
	"net/http"
)

const (
	SSRF_TOKEN_HEADER = "X-Aliyun-Parameters-Secrets-Token"
)

type HttpServeOptions struct {
	SsrfToken string
}

type ErrorResponse struct {
	Error   string `json:"error"`
	Message string `json:"message"`
}

type VersionResponse struct {
	Name    string `json:"name"`
	Version string `json:"version"`
	Startup int64  `json:"startup"`
}

func printResponse(w http.ResponseWriter, code int, response any) {
	w.WriteHeader(code)
	w.Header().Set("Content-Type", "application/json")
	responseJson, err := json.Marshal(response)
	if err == nil {
		_, _ = w.Write(responseJson)
	}
}

func isRequestAllowed(w http.ResponseWriter, r *http.Request, serveOptions *HttpServeOptions) bool {
	if serveOptions.SsrfToken != "" {
		ssrfTokenFromRequest := getSsrfToken(r)
		if ssrfTokenFromRequest == "" {
			printResponse(w, http.StatusUnauthorized, ErrorResponse{
				Error:   "request_denied",
				Message: SSRF_TOKEN_HEADER + " header or __ssrf_token parameter is required",
			})
			return false
		}
		if ssrfTokenFromRequest != serveOptions.SsrfToken {
			printResponse(w, http.StatusForbidden, ErrorResponse{
				Error:   "request_denied",
				Message: "Invalid SSRF token",
			})
			return false
		}
	}
	return true
}

func getSsrfToken(r *http.Request) string {
	ssrfTokenFromHeader := r.Header.Get(SSRF_TOKEN_HEADER)
	if ssrfTokenFromHeader != "" {
		return ssrfTokenFromHeader
	}
	query := r.URL.Query()
	return query.Get("__ssrf_token")
}
