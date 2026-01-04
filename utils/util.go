package utils

import (
	"crypto"
	"encoding/asn1"
	"encoding/hex"
	"math/big"
	"os/exec"
	"runtime"
	"time"

	"github.com/pkg/errors"
)

func OpenUrl(url string) error {
	var cmd string
	var args []string

	switch runtime.GOOS {
	case "windows":
		cmd = "cmd"
		args = []string{"/c", "start", url}
	case "darwin":
		cmd = "open"
		args = []string{url}
	case "linux":
		cmd = "xdg-open"
		args = []string{url}
	default:
		return errors.Errorf("unsupported platform")
	}
	return exec.Command(cmd, args...).Start()
}

func Sha256ToHex(message string) string {
	hash := crypto.SHA256.New()
	hash.Write([]byte(message))
	hashBytes := hash.Sum(nil)
	return hex.EncodeToString(hashBytes)
}

func SleepSeconds(interval int64) {
	sleepInterval := 2
	if interval <= 0 {
		sleepInterval = 2
	} else if interval > 5 {
		sleepInterval = 5
	}
	time.Sleep(time.Duration(sleepInterval) * time.Second)
}

type ECDSASignature struct {
	R, S *big.Int
}

// ParseECDSASignatureToRs Parse DER format to R and S
// ECDSA DER format looks like below:
// SEQUENCE {
//    INTEGER r
//    INTEGER s
// }
// Example：
// SEQUENCE {
//  INTEGER: 0083ef88158ebfe6065c8c9a6e00724685309a42366ff9cdf38931c0203229127c
//  INTEGER: 29068abc9e216f18ac5a3831b9c7bfdb7289c757fcefda441b8cf41e774a98a9
// }
func ParseECDSASignatureToRs(signature []byte) ([]byte, error) {
	var sig ECDSASignature
	_, err := asn1.Unmarshal(signature, &sig)
	if err != nil {
		return nil, errors.Errorf(" error parse signature: %v", err)
	}
	rs := append(alignEcCoord(sig.R.Bytes()), alignEcCoord(sig.S.Bytes())...)
	rsLen := len(rs)
	if rsLen != 64 && rsLen != 96 && rsLen != 128 {
		return nil, errors.Errorf("error parse ECDSA signature: invalid length of R or S %s", hex.EncodeToString(signature))
	}
	return rs, nil
}

func alignEcCoord(x []byte) []byte {
	xLen := len(x)
	// 32 P256; 48 P384; 66 P521
	if xLen == 32 || xLen == 48 || xLen == 66 {
		return x
	}
	if xLen == 31 {
		return append([]byte{0}, x...)
	}
	if xLen == 30 {
		return append([]byte{0, 0}, x...)
	}
	if xLen == 33 {
		return x[1:]
	}
	if xLen == 47 {
		return append([]byte{0}, x...)
	}
	if xLen == 46 {
		return append([]byte{0, 0}, x...)
	}
	if xLen == 49 {
		return x[1:]
	}
	if xLen == 65 {
		return append([]byte{0}, x...)
	}
	if xLen == 64 {
		return append([]byte{0, 0}, x...)
	}
	if xLen == 67 {
		// since P521's most high bit should always be 0, this line is not reachable
		return x[1:]
	}
	// SHOULD NOT HAPPEN
	return x
}
