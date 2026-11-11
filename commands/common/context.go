package common

import (
	"fmt"
	"net/url"
	"sort"
	"strings"

	"github.com/aliyunidaas/alibaba-cloud-idaas/config"
)

// RegistrationContext is configuration metadata only; resolving it never logs in.
type RegistrationContext struct {
	Instance       string `json:"instance"`
	ClientID       string `json:"client_id"`
	ClientIDSource string `json:"client_id_source"`
	Profile        string `json:"profile,omitempty"`
}

func NormalizeInstance(value string) (string, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return "", nil
	}
	if !strings.Contains(value, "://") {
		value = "https://" + value
	}
	u, err := url.Parse(value)
	if err != nil || u.Hostname() == "" || u.User != nil || (u.Scheme != "http" && u.Scheme != "https") || strings.Trim(u.Path, "/") != "" || u.RawQuery != "" || u.Fragment != "" {
		return "", fmt.Errorf("invalid instance domain; supply a host without credentials, path or query")
	}
	return strings.ToLower(u.Host), nil
}

func ResolveRegistrationContext(filename, instance, client string) (*RegistrationContext, error) {
	domain, err := NormalizeInstance(instance)
	if err != nil {
		return nil, err
	}
	result := &RegistrationContext{Instance: domain, ClientID: client, ClientIDSource: "explicit"}
	if domain != "" && client != "" {
		return result, nil
	}
	if filename == "" {
		filename, err = config.GetDefaultCloudCredentialConfigFile()
		if err != nil {
			return nil, err
		}
	}
	cfg, err := config.ReadCloudCredentialConfig(filename)
	if err != nil {
		return nil, err
	}
	candidates := map[string]*RegistrationContext{}
	if cfg != nil {
		for name, sts := range cfg.Profile {
			if sts == nil || sts.CloudAccount == nil || sts.CloudAccount.AccessTokenProvider == nil {
				continue
			}
			dc := sts.CloudAccount.AccessTokenProvider.OidcTokenProviderDeviceCode
			if dc == nil || dc.Issuer == "" {
				continue
			}
			d, e := NormalizeInstance(extractDomainFromIssuer(dc.Issuer))
			if e != nil {
				continue
			}
			c := dc.ClientId
			if c == "" {
				c = DefaultClientId
			}
			if domain != "" && d != domain {
				continue
			}
			candidates[name] = &RegistrationContext{Instance: d, ClientID: c, ClientIDSource: "profile:" + name, Profile: name}
		}
		selected := ResolveProfile("")
		explicitSelection := selected != ""
		if selected == "" {
			selected = cfg.CurrentProfile
		}
		if candidate := candidates[selected]; candidate != nil {
			if client != "" {
				candidate.ClientID = client
				candidate.ClientIDSource = "explicit"
			}
			return candidate, nil
		}
		if domain == "" && selected != "" {
			selectedProfile, exists := cfg.Profile[selected]
			if explicitSelection || !exists || selectedProfile == nil || selectedProfile.CloudAccount != nil {
				return nil, fmt.Errorf("selected profile %s cannot determine this registration context; specify --instance", selected)
			}
			// An implicit non-cloud default belongs to another CLI workflow.
			// Allow only the unambiguous cloud-candidate resolution below; an
			// explicit selection or an unusable cloud identity must never be replaced.
		}
	}
	names := make([]string, 0, len(candidates))
	for name := range candidates {
		names = append(names, name)
	}
	sort.Strings(names)
	var chosen *RegistrationContext
	for _, name := range names {
		candidate := candidates[name]
		if client != "" && candidate.ClientID != client {
			continue
		}
		if chosen != nil && (chosen.Instance != candidate.Instance || chosen.ClientID != candidate.ClientID) {
			return nil, fmt.Errorf("ambiguous instance/client-id in profiles %s; specify --instance and --client-id or select IDAAS_PROFILE", strings.Join(names, ", "))
		}
		if chosen == nil {
			chosen = candidate
		}
	}
	if chosen != nil {
		if client != "" {
			chosen.ClientIDSource = "explicit"
		}
		return chosen, nil
	}
	if domain == "" {
		return nil, fmt.Errorf("instance domain is required; specify --instance <domain>")
	}
	if client == "" {
		result.ClientID = DefaultClientId
		result.ClientIDSource = "built-in default"
	}
	return result, nil
}
