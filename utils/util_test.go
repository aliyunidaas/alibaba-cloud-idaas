package utils

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"encoding/asn1"
	"encoding/hex"
	"math/big"
	"testing"
)

// TestSha256ToHex tests the Sha256ToHex function with various inputs
func TestSha256ToHex(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "empty string",
			input:    "",
			expected: "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855",
		},
		{
			name:     "simple string",
			input:    "hello",
			expected: "2cf24dba5fb0a30e26e83b2ac5b9e29e1b161e5c1fa7425e73043362938b9824",
		},
		{
			name:     "string with spaces",
			input:    "hello world",
			expected: "b94d27b9934d3e08a52e52d7da7dabfac484efe37a5380ee9088f7ace2efcde9",
		},
		{
			name:     "numeric string",
			input:    "12345",
			expected: "5994471abb01112afcc18159f6cc74b4f511b99806da59b3caf5a9c173cacfc5",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := Sha256ToHex(tt.input)
			if result != tt.expected {
				t.Errorf("Sha256ToHex(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

// TestSha256ToHex_Consistency verifies that the same input always produces the same output
func TestSha256ToHex_Consistency(t *testing.T) {
	input := "test consistency"
	result1 := Sha256ToHex(input)
	result2 := Sha256ToHex(input)

	if result1 != result2 {
		t.Errorf("Sha256ToHex is not consistent: got %q and %q for the same input", result1, result2)
	}
}

// TestSha256ToHex_OutputLength verifies the output is always 64 characters (256 bits in hex)
func TestSha256ToHex_OutputLength(t *testing.T) {
	inputs := []string{"", "a", "test", "longer string for testing"}

	for _, input := range inputs {
		result := Sha256ToHex(input)
		if len(result) != 64 {
			t.Errorf("Sha256ToHex(%q) returned %d characters, expected 64", input, len(result))
		}
	}
}

// TestAlignEcCoord tests the alignEcCoord function with various coordinate lengths
func TestAlignEcCoord(t *testing.T) {
	tests := []struct {
		name           string
		inputLength    int
		expectedLength int
		description    string
	}{
		{
			name:           "P256 standard length",
			inputLength:    32,
			expectedLength: 32,
			description:    "32 bytes should remain 32 bytes",
		},
		{
			name:           "P384 standard length",
			inputLength:    48,
			expectedLength: 48,
			description:    "48 bytes should remain 48 bytes",
		},
		{
			name:           "P521 standard length",
			inputLength:    66,
			expectedLength: 66,
			description:    "66 bytes should remain 66 bytes",
		},
		{
			name:           "31 bytes - pad to 32",
			inputLength:    31,
			expectedLength: 32,
			description:    "31 bytes should be padded to 32 bytes",
		},
		{
			name:           "30 bytes - pad to 32",
			inputLength:    30,
			expectedLength: 32,
			description:    "30 bytes should be padded to 32 bytes",
		},
		{
			name:           "33 bytes - trim to 32",
			inputLength:    33,
			expectedLength: 32,
			description:    "33 bytes should be trimmed to 32 bytes",
		},
		{
			name:           "47 bytes - pad to 48",
			inputLength:    47,
			expectedLength: 48,
			description:    "47 bytes should be padded to 48 bytes",
		},
		{
			name:           "46 bytes - pad to 48",
			inputLength:    46,
			expectedLength: 48,
			description:    "46 bytes should be padded to 48 bytes",
		},
		{
			name:           "49 bytes - trim to 48",
			inputLength:    49,
			expectedLength: 48,
			description:    "49 bytes should be trimmed to 48 bytes",
		},
		{
			name:           "65 bytes - pad to 66",
			inputLength:    65,
			expectedLength: 66,
			description:    "65 bytes should be padded to 66 bytes",
		},
		{
			name:           "64 bytes - pad to 66",
			inputLength:    64,
			expectedLength: 66,
			description:    "64 bytes should be padded to 66 bytes",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			input := make([]byte, tt.inputLength)
			// Fill with non-zero data to make it easier to debug
			for i := range input {
				input[i] = byte(i % 256)
			}

			result := alignEcCoord(input)
			if len(result) != tt.expectedLength {
				t.Errorf("%s: got length %d, want %d", tt.description, len(result), tt.expectedLength)
			}
		})
	}
}

// TestAlignEcCoord_PreservesData verifies that alignment doesn't corrupt the data
func TestAlignEcCoord_PreservesData(t *testing.T) {
	// Test with 31 bytes (should pad to 32)
	input := make([]byte, 31)
	for i := range input {
		input[i] = byte(i + 1)
	}

	result := alignEcCoord(input)

	if len(result) != 32 {
		t.Fatalf("Expected length 32, got %d", len(result))
	}

	// The original data should be preserved (with leading zero)
	if result[0] != 0 {
		t.Errorf("Expected leading zero, got %d", result[0])
	}

	for i := 1; i < len(result); i++ {
		if result[i] != input[i-1] {
			t.Errorf("Data corrupted at position %d: got %d, want %d", i, result[i], input[i-1])
		}
	}
}

// TestParseECDSASignatureToRs tests parsing of ECDSA signatures
func TestParseECDSASignatureToRs(t *testing.T) {
	tests := []struct {
		name        string
		setupFunc   func() []byte
		expectError bool
		description string
	}{
		{
			name: "valid P256 signature",
			setupFunc: func() []byte {
				// Create a valid ECDSA signature structure for P256
				// Use properly sized byte arrays for P256 (32 bytes each)
				rBytes := make([]byte, 32)
				sBytes := make([]byte, 32)
				// Fill with some non-zero data
				for i := range rBytes {
					rBytes[i] = byte(i + 1)
					sBytes[i] = byte(i + 33)
				}
				r := new(big.Int).SetBytes(rBytes)
				s := new(big.Int).SetBytes(sBytes)
				sig := ECDSASignature{R: r, S: s}
				der, _ := asn1.Marshal(sig)
				return der
			},
			expectError: false,
			description: "should successfully parse valid P256 signature",
		},
		{
			name: "valid P384 signature",
			setupFunc: func() []byte {
				// Create a valid ECDSA signature structure for P384
				// Use properly sized byte arrays for P384 (48 bytes each)
				rBytes := make([]byte, 48)
				sBytes := make([]byte, 48)
				// Fill with some non-zero data
				for i := range rBytes {
					rBytes[i] = byte((i + 1) % 256)
					sBytes[i] = byte((i + 49) % 256)
				}
				r := new(big.Int).SetBytes(rBytes)
				s := new(big.Int).SetBytes(sBytes)
				sig := ECDSASignature{R: r, S: s}
				der, _ := asn1.Marshal(sig)
				return der
			},
			expectError: false,
			description: "should successfully parse valid P384 signature",
		},
		{
			name: "invalid DER encoding",
			setupFunc: func() []byte {
				return []byte{0x01, 0x02, 0x03} // Invalid DER
			},
			expectError: true,
			description: "should return error for invalid DER encoding",
		},
		{
			name: "empty signature",
			setupFunc: func() []byte {
				return []byte{}
			},
			expectError: true,
			description: "should return error for empty signature",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			signature := tt.setupFunc()
			result, err := ParseECDSASignatureToRs(signature)

			if tt.expectError {
				if err == nil {
					t.Errorf("%s: expected error but got none", tt.description)
				}
			} else {
				if err != nil {
					t.Errorf("%s: unexpected error: %v", tt.description, err)
				}
				if result == nil {
					t.Errorf("%s: result is nil", tt.description)
				}
			}
		})
	}
}

// TestParseECDSASignatureToRs_RealSignature tests with actual ECDSA signature
func TestParseECDSASignatureToRs_RealSignature(t *testing.T) {
	// Generate a real ECDSA key pair
	privateKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatalf("Failed to generate key: %v", err)
	}

	// Sign a message
	message := []byte("test message")
	hashHex := Sha256ToHex(string(message))
	hashBytes, err := hex.DecodeString(hashHex)
	if err != nil {
		t.Fatalf("Failed to decode hash: %v", err)
	}

	r, s, err := ecdsa.Sign(rand.Reader, privateKey, hashBytes)
	if err != nil {
		t.Fatalf("Failed to sign: %v", err)
	}

	// Create DER encoded signature
	sig := ECDSASignature{R: r, S: s}
	derSig, err := asn1.Marshal(sig)
	if err != nil {
		t.Fatalf("Failed to marshal signature: %v", err)
	}

	// Parse the signature
	result, err := ParseECDSASignatureToRs(derSig)
	if err != nil {
		t.Fatalf("ParseECDSASignatureToRs failed: %v", err)
	}

	// Verify the length (should be 64 for P256: 32 bytes R + 32 bytes S)
	if len(result) != 64 {
		t.Errorf("Expected result length 64, got %d", len(result))
	}
}

// TestParseECDSASignatureToRs_InvalidLength tests error handling for invalid signature lengths
func TestParseECDSASignatureToRs_InvalidLength(t *testing.T) {
	// Create a signature with invalid R/S length (not 32, 48, or 66 bytes total)
	r := new(big.Int).SetInt64(12345)
	s := new(big.Int).SetInt64(67890)
	sig := ECDSASignature{R: r, S: s}
	derSig, err := asn1.Marshal(sig)
	if err != nil {
		t.Fatalf("Failed to create test signature: %v", err)
	}

	_, err = ParseECDSASignatureToRs(derSig)
	if err == nil {
		t.Error("Expected error for invalid signature length, but got none")
	}
}

// TestAlignEcCoord_EdgeCases tests edge cases for coordinate alignment
func TestAlignEcCoord_EdgeCases(t *testing.T) {
	// Test that standard lengths pass through unchanged
	standardLengths := []int{32, 48, 66}
	for _, length := range standardLengths {
		input := make([]byte, length)
		for i := range input {
			input[i] = byte(i)
		}
		result := alignEcCoord(input)

		if len(result) != length {
			t.Errorf("Standard length %d changed to %d", length, len(result))
		}

		// Verify data is unchanged
		for i := range result {
			if result[i] != input[i] {
				t.Errorf("Data changed at position %d for length %d", i, length)
			}
		}
	}
}
