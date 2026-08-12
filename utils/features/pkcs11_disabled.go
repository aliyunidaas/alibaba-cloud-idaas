//go:build disable_pkcs11
// +build disable_pkcs11

package features

func IsPkcs11Enabled() bool {
	return false
}
