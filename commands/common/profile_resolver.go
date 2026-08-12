package common

import "os"

// ResolveProfile resolves the profile name with priority:
// --profile flag > IDAAS_PROFILE env > "" (let config.FindProfile use current_profile).
func ResolveProfile(flagProfile string) string {
	if flagProfile != "" {
		return flagProfile
	}
	if env := os.Getenv("IDAAS_PROFILE"); env != "" {
		return env
	}
	return ""
}
