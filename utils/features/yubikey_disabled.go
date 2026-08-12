//go:build disable_yubikey_piv
// +build disable_yubikey_piv

package features

func IsYubikeyEnabled() bool {
	return false
}
