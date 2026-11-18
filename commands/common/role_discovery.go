package common

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"unicode"

	"github.com/aliyunidaas/alibaba-cloud-idaas/utils"
)

// AssumableRole describes the role itself, not the source of its authorization.
type AssumableRole struct {
	CloudAccountName                   string `json:"cloudAccountName"`
	CloudAccountDescription            string `json:"cloudAccountDescription"`
	CloudPermissionTemplateId          string `json:"cloudPermissionTemplateId"`
	CloudPermissionTemplateName        string `json:"cloudPermissionTemplateName"`
	CloudPermissionTemplateDescription string `json:"cloudPermissionTemplateDescription"`
	CloudAccountRoleDescription        string `json:"cloudAccountRoleDescription"`
	CloudAccountId                     string `json:"cloudAccountId"`
	CloudAccountRoleId                 string `json:"cloudAccountRoleId"`
	CloudAccountRoleName               string `json:"cloudAccountRoleName"`
	CloudAccountRoleExternalId         string `json:"cloudAccountRoleExternalId"`
	CloudAccountVendorType             string `json:"cloudAccountVendorType"`
	Status                             string `json:"status"`
	CreatedByPermissionTemplate        *bool  `json:"createdByPermissionTemplate"`
}

// DisplayDescription prefers the template description without changing raw data.
func (r AssumableRole) DisplayDescription() string {
	if strings.TrimSpace(r.CloudPermissionTemplateDescription) != "" {
		return r.CloudPermissionTemplateDescription
	}
	return r.CloudAccountRoleDescription
}

// RoleDisplayCell keeps full text on one terminal line and marks missing values.
func RoleDisplayCell(value string) string {
	if strings.TrimSpace(value) == "" {
		return "-"
	}
	var out strings.Builder
	for _, r := range value {
		if unicode.IsControl(r) || r == '\u2028' || r == '\u2029' {
			quoted := strconv.QuoteRune(r)
			out.WriteString(quoted[1 : len(quoted)-1])
		} else {
			out.WriteRune(r)
		}
	}
	return out.String()
}

// FormatRoleTable provides the same complete role metadata for show and onboard.
func FormatRoleTable(roles []AssumableRole) string {
	var out strings.Builder
	indexWidth := len(fmt.Sprint(len(roles)))
	fmt.Fprintf(&out, "%-*s  CLOUD ACCOUNT NAME\tCLOUD ACCOUNT DESCRIPTION\tPERMISSION TEMPLATE NAME\tROLE NAME\tDESCRIPTION\n", indexWidth, "#")
	for i, r := range roles {
		fmt.Fprintf(&out, "%-*d  %s\t%s\t%s\t%s\t%s\n", indexWidth, i+1,
			RoleDisplayCell(r.CloudAccountName), RoleDisplayCell(r.CloudAccountDescription),
			RoleDisplayCell(r.CloudPermissionTemplateName), RoleDisplayCell(r.CloudAccountRoleName),
			RoleDisplayCell(r.DisplayDescription()))
	}
	return out.String()
}

// AssumableRoleList is the complete result, not a single server page.
type AssumableRoleList struct {
	Entities   []AssumableRole `json:"entities"`
	TotalCount int             `json:"totalCount"`
}

func TemplateCreationLabel(value *bool) string {
	if value == nil {
		return "unknown"
	}
	if *value {
		return "yes"
	}
	return "no"
}

// DiscoverAssumableRoles fixes the discovery mode for every page. It never
// returns partial results; callers must not write profiles before it succeeds.
func DiscoverAssumableRoles(client *http.Client, endpoint, instanceID, accessToken string) (*AssumableRoleList, error) {
	u, err := url.Parse(strings.TrimRight(endpoint, "/") + "/v2/" + url.PathEscape(instanceID) + "/cloudAccountRoles/_/actions/listAssumableCloudAccountRoles")
	if err != nil {
		return nil, fmt.Errorf("invalid role discovery endpoint")
	}
	result := &AssumableRoleList{Entities: []AssumableRole{}}
	seen := map[string]bool{}
	next := ""
	// A client safety limit, independent of server quotas and page sizes.
	const maxPages = 1000
	for page := 0; page < maxPages; page++ {
		query := u.Query()
		query.Set("includePermissionTemplateRoles", "true")
		query.Set("maxResults", "100")
		query.Del("nextToken")
		if next != "" {
			query.Set("nextToken", next)
		}
		u.RawQuery = query.Encode()
		req, err := http.NewRequest(http.MethodGet, u.String(), nil)
		if err != nil {
			return nil, fmt.Errorf("build role discovery request")
		}
		req.Header.Set("Authorization", "Bearer "+accessToken)
		req.Header.Set("User-Agent", utils.UserAgent)
		resp, err := client.Do(req)
		if err != nil {
			return nil, fmt.Errorf("role discovery page %d request failed", page+1)
		}
		const maxBody = 8 << 20
		body, readErr := io.ReadAll(io.LimitReader(resp.Body, maxBody+1))
		resp.Body.Close()
		if readErr != nil || len(body) > maxBody {
			return nil, fmt.Errorf("role discovery page %d response unreadable or too large", page+1)
		}
		var wire struct {
			Entities  *[]AssumableRole `json:"entities"`
			NextToken string           `json:"nextToken"`
			Error     string           `json:"error"`
			Code      json.RawMessage  `json:"code"`
			RequestID string           `json:"request_id"`
		}
		decodeErr := json.Unmarshal(body, &wire)
		code := strings.Trim(string(wire.Code), "\"")
		if resp.StatusCode != http.StatusOK || wire.Error != "" || (code != "" && code != "null" && code != "200" && code != "0") {
			hint := "check authorization and service availability"
			if strings.Contains(wire.Error, "NextToken") {
				hint = "restart discovery from the first page without changing the query mode"
			}
			if strings.Contains(wire.Error, "BatchSizeExceed") {
				hint = "contact the administrator to review authorization volume; reducing page size does not bypass the limit"
			}
			return nil, fmt.Errorf("role discovery page %d failed (HTTP %d, error=%s, code=%s, request_id=%s): %s", page+1, resp.StatusCode, wire.Error, code, wire.RequestID, hint)
		}
		if decodeErr != nil || wire.Entities == nil {
			return nil, fmt.Errorf("role discovery page %d returned an invalid response", page+1)
		}
		result.Entities = append(result.Entities, (*wire.Entities)...)
		if wire.NextToken == "" {
			result.TotalCount = len(result.Entities)
			return result, nil
		}
		if seen[wire.NextToken] {
			return nil, fmt.Errorf("role discovery returned a repeated cursor; restart discovery")
		}
		seen[wire.NextToken] = true
		next = wire.NextToken
	}
	return nil, fmt.Errorf("role discovery exceeded the client safety limit of %d pages", maxPages)
}
