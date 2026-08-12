package features

func GetEnabledFeatures() []string {
	var featureStrings []string
	if IsPkcs11Enabled() {
		featureStrings = append(featureStrings, "PKCS#11")
	}
	if IsYubikeyEnabled() {
		featureStrings = append(featureStrings, "YubiKey")
	}
	return featureStrings
}
