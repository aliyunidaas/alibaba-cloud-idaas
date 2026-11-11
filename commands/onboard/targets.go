package onboard

import (
	"fmt"
	"strings"
)

func validatedTargets(value string, roles []assumableRole, checkRoles bool) ([]string, error) {
	if strings.TrimSpace(value) == "" {
		return parseTargets("", roles), nil
	}
	targets := []string{}
	seen := map[string]bool{}
	vendors := map[string]string{"aliyun-cli": vendorAlibaba, "aws-cli": vendorAws, "tencentcloud-cli": vendorTencent}
	for _, part := range strings.Split(value, ",") {
		name := strings.TrimSpace(part)
		if name != "none" && vendors[name] == "" {
			return nil, fmt.Errorf("unsupported target %q; use aliyun-cli, aws-cli, tencentcloud-cli or none", name)
		}
		if checkRoles && name != "none" {
			found := false
			for _, role := range roles {
				if strings.EqualFold(role.CloudAccountVendorType, vendors[name]) {
					found = true
					break
				}
			}
			if !found {
				return nil, fmt.Errorf("target %s has no matching cloud roles; no configuration was written", name)
			}
		}
		if !seen[name] {
			targets = append(targets, name)
			seen[name] = true
		}
	}
	if seen["none"] && len(targets) > 1 {
		return nil, fmt.Errorf("target none cannot be combined with other targets")
	}
	return targets, nil
}
