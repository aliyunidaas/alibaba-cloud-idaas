package idaaslog

import "strings"

func IsOn(val string) bool {
	lowerVal := strings.ToLower(val)
	return lowerVal == "1" || lowerVal == "true" || lowerVal == "yes" || lowerVal == "y" || lowerVal == "on"
}
