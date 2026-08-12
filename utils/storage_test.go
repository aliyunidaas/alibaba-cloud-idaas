package utils

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/aliyunidaas/alibaba-cloud-idaas/constants"
	"github.com/stretchr/testify/assert"
)

// TestStringWithTime_IsExpired tests the IsExpired method
func TestStringWithTime_IsExpired(t *testing.T) {
	tests := []struct {
		name        string
		cacheTime   int64
		isExpired   bool
		description string
	}{
		{
			name:        "not expired (recent)",
			cacheTime:   time.Now().UnixMilli(),
			isExpired:   false,
			description: "should not be expired when cached recently",
		},
		{
			name:        "expired (3 days ago)",
			cacheTime:   time.Now().Add(-73 * time.Hour).UnixMilli(),
			isExpired:   true,
			description: "should be expired when cached more than 3 days ago",
		},
		{
			name:        "boundary (exactly 3 days)",
			cacheTime:   time.Now().Add(-72 * time.Hour).UnixMilli(),
			isExpired:   false,
			description: "should not be expired at exactly 3 days boundary (uses > not >=)",
		},
		{
			name:        "almost expired (2 days 23 hours)",
			cacheTime:   time.Now().Add(-71 * time.Hour).UnixMilli(),
			isExpired:   false,
			description: "should not be expired when cached less than 3 days ago",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			swt := &StringWithTime{CacheTime: tt.cacheTime}
			result := swt.IsExpired()
			if result != tt.isExpired {
				t.Errorf("%s: got %v, want %v", tt.description, result, tt.isExpired)
			}
		})
	}
}

// TestStringWithTime_IsExpiringOrExpired tests the IsExpiringOrExpired method
func TestStringWithTime_IsExpiringOrExpired(t *testing.T) {
	tests := []struct {
		name              string
		cacheTime         int64
		isExpiringOrExpired bool
		description       string
	}{
		{
			name:                "not expiring (recent)",
			cacheTime:           time.Now().UnixMilli(),
			isExpiringOrExpired: false,
			description:         "should not be expiring when cached recently",
		},
		{
			name:                "expiring (1 hour ago)",
			cacheTime:           time.Now().Add(-61 * time.Minute).UnixMilli(),
			isExpiringOrExpired: true,
			description:         "should be expiring when cached more than 1 hour ago",
		},
		{
			name:                "boundary (exactly 1 hour)",
			cacheTime:           time.Now().Add(-60 * time.Minute).UnixMilli(),
			isExpiringOrExpired: false,
			description:         "should not be expiring at exactly 1 hour boundary (uses > not >=)",
		},
		{
			name:                "almost expiring (59 minutes)",
			cacheTime:           time.Now().Add(-59 * time.Minute).UnixMilli(),
			isExpiringOrExpired: false,
			description:         "should not be expiring when cached less than 1 hour ago",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			swt := &StringWithTime{CacheTime: tt.cacheTime}
			result := swt.IsExpiringOrExpired()
			if result != tt.isExpiringOrExpired {
				t.Errorf("%s: got %v, want %v", tt.description, result, tt.isExpiringOrExpired)
			}
		})
	}
}

// TestStringWithTime_IsExpiredWithCustomFunc tests the IsExpiredWithCustomFunc method
func TestStringWithTime_IsExpiredWithCustomFunc(t *testing.T) {
	tests := []struct {
		name        string
		swt         *StringWithTime
		customFunc  func(*StringWithTime) bool
		wantExpired bool
		description string
	}{
		{
			name:        "nil pointer",
			swt:         nil,
			customFunc:  nil,
			wantExpired: true,
			description: "nil should be treated as expired",
		},
		{
			name:        "custom function returns true",
			swt:         &StringWithTime{CacheTime: time.Now().UnixMilli()},
			customFunc:  func(*StringWithTime) bool { return true },
			wantExpired: true,
			description: "custom function should override default logic",
		},
		{
			name:        "custom function returns false",
			swt:         &StringWithTime{CacheTime: time.Now().Add(-73 * time.Hour).UnixMilli()},
			customFunc:  func(*StringWithTime) bool { return false },
			wantExpired: false,
			description: "custom function should override default logic",
		},
		{
			name:        "nil custom function uses default",
			swt:         &StringWithTime{CacheTime: time.Now().UnixMilli()},
			customFunc:  nil,
			wantExpired: false,
			description: "nil custom function should use default IsExpired",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.swt.IsExpiredWithCustomFunc(tt.customFunc)
			if result != tt.wantExpired {
				t.Errorf("%s: got %v, want %v", tt.description, result, tt.wantExpired)
			}
		})
	}
}

// TestStringWithTime_IsExpiringOrExpiredWithCustomFunc tests the IsExpiringOrExpiredWithCustomFunc method
func TestStringWithTime_IsExpiringOrExpiredWithCustomFunc(t *testing.T) {
	tests := []struct {
		name              string
		swt               *StringWithTime
		customFunc        func(*StringWithTime) bool
		wantExpiring      bool
		description       string
	}{
		{
			name:         "nil pointer",
			swt:          nil,
			customFunc:   nil,
			wantExpiring: true,
			description:  "nil should be treated as expiring",
		},
		{
			name:         "custom function returns true",
			swt:          &StringWithTime{CacheTime: time.Now().UnixMilli()},
			customFunc:   func(*StringWithTime) bool { return true },
			wantExpiring: true,
			description:  "custom function should override default logic",
		},
		{
			name:         "custom function returns false",
			swt:          &StringWithTime{CacheTime: time.Now().Add(-61 * time.Minute).UnixMilli()},
			customFunc:   func(*StringWithTime) bool { return false },
			wantExpiring: false,
			description:  "custom function should override default logic",
		},
		{
			name:         "nil custom function uses default",
			swt:          &StringWithTime{CacheTime: time.Now().UnixMilli()},
			customFunc:   nil,
			wantExpiring: false,
			description:  "nil custom function should use default IsExpiringOrExpired",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.swt.IsExpiringOrExpiredWithCustomFunc(tt.customFunc)
			if result != tt.wantExpiring {
				t.Errorf("%s: got %v, want %v", tt.description, result, tt.wantExpiring)
			}
		})
	}
}

// TestStringWithTime_Marshal tests the Marshal method
func TestStringWithTime_Marshal(t *testing.T) {
	tests := []struct {
		name        string
		swt         *StringWithTime
		wantContent string
		wantError   bool
		description string
	}{
		{
			name: "valid StringWithTime",
			swt: &StringWithTime{
				CacheTime: 1234567890,
				Context:   map[string]interface{}{"key": "value"},
				Content:   "test content",
			},
			wantContent: `{"cache_time":1234567890,"context":{"key":"value"},"content":"test content"}`,
			wantError:   false,
			description: "should marshal successfully",
		},
		{
			name: "empty context",
			swt: &StringWithTime{
				CacheTime: 1234567890,
				Context:   nil,
				Content:   "test content",
			},
			wantContent: `{"cache_time":1234567890,"context":null,"content":"test content"}`,
			wantError:   false,
			description: "should handle nil context",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := tt.swt.Marshal()
			if (err != nil) != tt.wantError {
				t.Errorf("%s: error = %v, wantError %v", tt.description, err, tt.wantError)
			}
			if !tt.wantError {
				// Parse and compare to avoid ordering issues
				var got, want map[string]interface{}
				json.Unmarshal([]byte(result), &got)
				json.Unmarshal([]byte(tt.wantContent), &want)
				if !assert.Equal(t, want, got) {
					t.Errorf("%s: got %v, want %v", tt.description, result, tt.wantContent)
				}
			}
		})
	}
}

// TestUnmarshalStringWithTime tests the UnmarshalStringWithTime function
func TestUnmarshalStringWithTime(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		wantError   bool
		description string
	}{
		{
			name:        "valid JSON",
			input:       `{"cache_time":1234567890,"context":{"key":"value"},"content":"test content"}`,
			wantError:   false,
			description: "should unmarshal successfully",
		},
		{
			name:        "invalid JSON",
			input:       `invalid json`,
			wantError:   true,
			description: "should return error for invalid JSON",
		},
		{
			name:        "empty string",
			input:       ``,
			wantError:   true,
			description: "should return error for empty string",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := UnmarshalStringWithTime(tt.input)
			if (err != nil) != tt.wantError {
				t.Errorf("%s: error = %v, wantError %v", tt.description, err, tt.wantError)
			}
			if !tt.wantError && result == nil {
				t.Errorf("%s: result is nil", tt.description)
			}
		})
	}
}

// TestEncryptText_DecryptText tests the encryption and decryption round-trip
func TestEncryptText_DecryptText(t *testing.T) {
	plaintext := "This is a secret message that needs to be encrypted"
	additionalData := []byte("test-category-test-key")

	// Test encryption
	ciphertext, err := EncryptText(plaintext, additionalData)
	if err != nil {
		t.Fatalf("EncryptText failed: %v", err)
	}

	// Test decryption
	decrypted, err := DecryptText(ciphertext, additionalData)
	if err != nil {
		t.Fatalf("DecryptText failed: %v", err)
	}

	// Verify the decrypted text matches the original
	if decrypted != plaintext {
		t.Errorf("Decrypted text does not match original: got %q, want %q", decrypted, plaintext)
	}
}

// TestEncryptText_InvalidCiphertext tests error handling for invalid ciphertext
func TestEncryptText_InvalidCiphertext(t *testing.T) {
	tests := []struct {
		name          string
		ciphertext    string
		additionalData []byte
		wantError     bool
		description   string
	}{
		{
			name:          "invalid format (not 3 parts)",
			ciphertext:    "encrypted:nonce",
			additionalData: []byte("test"),
			wantError:     true,
			description:   "should fail for invalid format",
		},
		{
			name:          "wrong prefix",
			ciphertext:    "wrong:nonce:ciphertext",
			additionalData: []byte("test"),
			wantError:     true,
			description:   "should fail for wrong prefix",
		},
		{
			name:          "invalid nonce encoding",
			ciphertext:    "encrypted:$$$invalid$$$:" + base64.RawURLEncoding.EncodeToString([]byte("ciphertext")),
			additionalData: []byte("test"),
			wantError:     true,
			description:   "should fail for invalid nonce encoding",
		},
		{
			name:          "invalid ciphertext encoding",
			ciphertext:    "encrypted:" + base64.RawURLEncoding.EncodeToString([]byte("nonce")) + ":$$$invalid$$$",
			additionalData: []byte("test"),
			wantError:     true,
			description:   "should fail for invalid ciphertext encoding",
		},
		{
			name:          "wrong additional data",
			ciphertext:    "encrypted:" + base64.RawURLEncoding.EncodeToString(make([]byte, 12)) + ":" + base64.RawURLEncoding.EncodeToString([]byte("ciphertext")),
			additionalData: []byte("wrong-data"),
			wantError:     true,
			description:   "should fail for wrong additional data (authentication failure)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := DecryptText(tt.ciphertext, tt.additionalData)
			if (err != nil) != tt.wantError {
				t.Errorf("%s: error = %v, wantError %v", tt.description, err, tt.wantError)
			}
		})
	}
}

// TestReadCacheFileWithEncryption_WriteCacheFileWithEncryption tests file encryption round-trip
func TestReadCacheFileWithEncryption_WriteCacheFileWithEncryption(t *testing.T) {
	tmpDir := t.TempDir()
	originalHomeDir := os.Getenv("HOME")
	os.Setenv("HOME", tmpDir)
	defer os.Setenv("HOME", originalHomeDir)

	category := "test-category"
	key := "test-key"
	content := "sensitive cached data"

	// Write encrypted cache file
	err := WriteCacheFileWithEncryption(category, key, content)
	if err != nil {
		t.Fatalf("WriteCacheFileWithEncryption failed: %v", err)
	}

	// Read encrypted cache file
	readContent, err := ReadCacheFileWithEncryption(category, key)
	if err != nil {
		t.Fatalf("ReadCacheFileWithEncryption failed: %v", err)
	}

	// Verify content matches
	if readContent != content {
		t.Errorf("Read content does not match: got %q, want %q", readContent, content)
	}
}

// TestReadCacheFileWithEncryption_NonExistentFile tests reading non-existent file
func TestReadCacheFileWithEncryption_NonExistentFile(t *testing.T) {
	tmpDir := t.TempDir()
	originalHomeDir := os.Getenv("HOME")
	os.Setenv("HOME", tmpDir)
	defer os.Setenv("HOME", originalHomeDir)

	content, err := ReadCacheFileWithEncryption("non-existent", "non-existent")
	if err != nil {
		t.Errorf("ReadCacheFileWithEncryption should not return error for non-existent file, got: %v", err)
	}
	if content != "" {
		t.Errorf("ReadCacheFileWithEncryption should return empty string for non-existent file, got: %q", content)
	}
}

// TestRemoveCacheFile tests removing cache file
func TestRemoveCacheFile(t *testing.T) {
	tmpDir := t.TempDir()
	originalHomeDir := os.Getenv("HOME")
	os.Setenv("HOME", tmpDir)
	defer os.Setenv("HOME", originalHomeDir)

	category := "test-category"
	key := "test-key"
	content := "test content"

	// Write a cache file
	err := WriteCacheFileWithEncryption(category, key, content)
	if err != nil {
		t.Fatalf("WriteCacheFileWithEncryption failed: %v", err)
	}

	// Verify file exists
	cacheFile, _ := getCacheFile(category, key)
	if _, err := os.Stat(cacheFile); os.IsNotExist(err) {
		t.Fatal("Cache file should exist after writing")
	}

	// Remove the cache file
	err = RemoveCacheFile(category, key)
	if err != nil {
		t.Fatalf("RemoveCacheFile failed: %v", err)
	}

	// Verify file is removed
	if _, err := os.Stat(cacheFile); !os.IsNotExist(err) {
		t.Error("Cache file should not exist after removal")
	}
}

// TestRemoveCacheFile_NonExistentFile tests removing non-existent file
func TestRemoveCacheFile_NonExistentFile(t *testing.T) {
	tmpDir := t.TempDir()
	originalHomeDir := os.Getenv("HOME")
	os.Setenv("HOME", tmpDir)
	defer os.Setenv("HOME", originalHomeDir)

	// Removing non-existent file should not return error
	err := RemoveCacheFile("non-existent", "non-existent")
	if err != nil {
		t.Errorf("RemoveCacheFile should not return error for non-existent file, got: %v", err)
	}
}

// TestReadCacheWithEncryptionCallback_ForceNew tests ForceNew option
func TestReadCacheWithEncryptionCallback_ForceNew(t *testing.T) {
	tmpDir := t.TempDir()
	originalHomeDir := os.Getenv("HOME")
	os.Setenv("HOME", tmpDir)
	defer os.Setenv("HOME", originalHomeDir)

	category := "test-category"
	key := "test-key"
	cachedContent := "old cached content"
	newContent := "new fetched content"

	// Write cached content
	err := WriteCacheFileWithEncryption(category, key, cachedContent)
	if err != nil {
		t.Fatalf("WriteCacheFileWithEncryption failed: %v", err)
	}

	// Setup fetch function that returns new content
	fetchCalled := false
	options := &ReadCacheOptions{
		FetchContent: func() (int, string, error) {
			fetchCalled = true
			return http.StatusOK, newContent, nil
		},
		ForceNew: true,
	}

	// Read with ForceNew should fetch new content
	result, err := ReadCacheFileWithEncryptionCallback(category, key, options)
	if err != nil {
		t.Fatalf("ReadCacheFileWithEncryptionCallback failed: %v", err)
	}

	if !fetchCalled {
		t.Error("FetchContent should be called when ForceNew is true")
	}
	if result != newContent {
		t.Errorf("Result should be new content, got: %q", result)
	}
}

// TestReadCacheWithEncryptionCallback_AllowExpired tests AllowExpired option
func TestReadCacheWithEncryptionCallback_AllowExpired(t *testing.T) {
	tmpDir := t.TempDir()
	originalHomeDir := os.Getenv("HOME")
	os.Setenv("HOME", tmpDir)
	defer os.Setenv("HOME", originalHomeDir)

	category := "test-category"
	key := "test-key"
	cachedContent := "old cached content"

	// Write cached content with old timestamp (expired)
	oldTime := time.Now().Add(-73 * time.Hour).UnixMilli()
	swt := StringWithTime{
		CacheTime: oldTime,
		Content:   cachedContent,
	}
	marshaled, _ := swt.Marshal()
	_ = WriteCacheFileWithEncryption(category, key, marshaled)

	// Setup fetch function that fails
	options := &ReadCacheOptions{
		FetchContent: func() (int, string, error) {
			return http.StatusInternalServerError, "", fmt.Errorf("fetch failed")
		},
		AllowExpired: true,
	}

	// Read with AllowExpired should return expired cached content
	result, err := ReadCacheFileWithEncryptionCallback(category, key, options)
	if err != nil {
		t.Fatalf("ReadCacheFileWithEncryptionCallback failed: %v", err)
	}

	if result != cachedContent {
		t.Errorf("Result should be expired cached content, got: %q", result)
	}
}

// TestReadCacheWithEncryptionCallback_FetchSuccess tests successful fetch
func TestReadCacheWithEncryptionCallback_FetchSuccess(t *testing.T) {
	tmpDir := t.TempDir()
	originalHomeDir := os.Getenv("HOME")
	os.Setenv("HOME", tmpDir)
	defer os.Setenv("HOME", originalHomeDir)

	category := "test-category"
	key := "test-key"
	fetchedContent := "freshly fetched content"

	// Setup fetch function
	fetchCalled := false
	options := &ReadCacheOptions{
		FetchContent: func() (int, string, error) {
			fetchCalled = true
			return http.StatusOK, fetchedContent, nil
		},
	}

	// Read should fetch content (no cache exists)
	result, err := ReadCacheFileWithEncryptionCallback(category, key, options)
	if err != nil {
		t.Fatalf("ReadCacheFileWithEncryptionCallback failed: %v", err)
	}

	if !fetchCalled {
		t.Error("FetchContent should be called when cache doesn't exist")
	}
	if result != fetchedContent {
		t.Errorf("Result should be fetched content, got: %q", result)
	}

	// Verify cache was written
	readBack, err := ReadCacheFileWithEncryption(category, key)
	if err != nil {
		t.Fatalf("ReadCacheFileWithEncryption failed: %v", err)
	}
	var swt StringWithTime
	json.Unmarshal([]byte(readBack), &swt)
	if swt.Content != fetchedContent {
		t.Errorf("Cached content should match fetched content, got: %q", swt.Content)
	}
}

// TestReadCacheWithEncryptionCallback_FetchFailure tests fetch failure without ForceNew
func TestReadCacheWithEncryptionCallback_FetchFailure(t *testing.T) {
	tmpDir := t.TempDir()
	originalHomeDir := os.Getenv("HOME")
	os.Setenv("HOME", tmpDir)
	defer os.Setenv("HOME", originalHomeDir)

	category := "test-category"
	key := "test-key"

	// Setup fetch function that fails
	options := &ReadCacheOptions{
		FetchContent: func() (int, string, error) {
			return http.StatusInternalServerError, "", fmt.Errorf("fetch failed")
		},
	}

	// Read should fail when no cache exists and fetch fails
	_, err := ReadCacheFileWithEncryptionCallback(category, key, options)
	if err == nil {
		t.Error("ReadCacheFileWithEncryptionCallback should fail when no cache exists and fetch fails")
	}
}

// TestReadCacheWithEncryptionCallback_ForceNewFetchFailure tests ForceNew with fetch failure
func TestReadCacheWithEncryptionCallback_ForceNewFetchFailure(t *testing.T) {
	tmpDir := t.TempDir()
	originalHomeDir := os.Getenv("HOME")
	os.Setenv("HOME", tmpDir)
	defer os.Setenv("HOME", originalHomeDir)

	category := "test-category"
	key := "test-key"
	cachedContent := "cached content"

	// Write cached content
	_ = WriteCacheFileWithEncryption(category, key, cachedContent)

	// Setup fetch function that fails
	options := &ReadCacheOptions{
		FetchContent: func() (int, string, error) {
			return http.StatusInternalServerError, "", fmt.Errorf("fetch failed")
		},
		ForceNew: true,
	}

	// Read should fail when ForceNew is true and fetch fails
	_, err := ReadCacheFileWithEncryptionCallback(category, key, options)
	if err == nil {
		t.Error("ReadCacheFileWithEncryptionCallback should fail when ForceNew is true and fetch fails")
	}
}

// TestReadCacheWithEncryptionCallback_CustomExpiryFunc tests custom expiry functions
func TestReadCacheWithEncryptionCallback_CustomExpiryFunc(t *testing.T) {
	tmpDir := t.TempDir()
	originalHomeDir := os.Getenv("HOME")
	os.Setenv("HOME", tmpDir)
	defer os.Setenv("HOME", originalHomeDir)

	category := "test-category"
	key := "test-key"
	cachedContent := "cached content"
	fetchedContent := "fetched content"

	// Write cached content
	_ = WriteCacheFileWithEncryption(category, key, cachedContent)

	// Setup custom expiry function that always returns true (always expired)
	options := &ReadCacheOptions{
		FetchContent: func() (int, string, error) {
			return http.StatusOK, fetchedContent, nil
		},
		IsContentExpiringOrExpired: func(swt *StringWithTime) bool {
			return true
		},
	}

	// Read should use custom expiry function
	result, err := ReadCacheFileWithEncryptionCallback(category, key, options)
	if err != nil {
		t.Fatalf("ReadCacheFileWithEncryptionCallback failed: %v", err)
	}

	if result != fetchedContent {
		t.Errorf("Result should be fetched content, got: %q", result)
	}
}

// TestGetCacheFile tests getCacheFile function
func TestGetCacheFile(t *testing.T) {
	tmpDir := t.TempDir()
	originalHomeDir := os.Getenv("HOME")
	os.Setenv("HOME", tmpDir)
	defer os.Setenv("HOME", originalHomeDir)

	category := "test-category"
	key := "test-key"

	cacheFile, err := getCacheFile(category, key)
	if err != nil {
		t.Fatalf("getCacheFile failed: %v", err)
	}

	expectedPath := filepath.Join(tmpDir, constants.ConfigRootDir, constants.ConfigIdaasDir, category, key)
	if cacheFile != expectedPath {
		t.Errorf("getCacheFile returned %q, want %q", cacheFile, expectedPath)
	}

	// Verify directory was created
	if _, err := os.Stat(filepath.Dir(cacheFile)); os.IsNotExist(err) {
		t.Error("Cache directory should be created")
	}
}

// TestWriteCacheFile_ReadCacheFile tests low-level file operations
func TestWriteCacheFile_ReadCacheFile(t *testing.T) {
	tmpDir := t.TempDir()
	originalHomeDir := os.Getenv("HOME")
	os.Setenv("HOME", tmpDir)
	defer os.Setenv("HOME", originalHomeDir)

	category := "test-category"
	key := "test-key"
	content := []byte("raw content")

	// Write cache file
	err := writeCacheFile(category, key, content)
	if err != nil {
		t.Fatalf("writeCacheFile failed: %v", err)
	}

	// Read cache file
	readContent, err := readCacheFile(category, key)
	if err != nil {
		t.Fatalf("readCacheFile failed: %v", err)
	}

	if string(readContent) != string(content) {
		t.Errorf("Read content does not match: got %q, want %q", string(readContent), string(content))
	}
}

// TestWriteCacheFile_ReadCacheFile_NonExistent tests reading non-existent file
func TestWriteCacheFile_ReadCacheFile_NonExistent(t *testing.T) {
	tmpDir := t.TempDir()
	originalHomeDir := os.Getenv("HOME")
	os.Setenv("HOME", tmpDir)
	defer os.Setenv("HOME", originalHomeDir)

	content, err := readCacheFile("non-existent", "non-existent")
	if err != nil {
		t.Errorf("readCacheFile should not return error for non-existent file, got: %v", err)
	}
	if content != nil {
		t.Errorf("readCacheFile should return nil for non-existent file, got: %v", content)
	}
}

// TestGenerateRandomBytes tests generateRandomBytes function
func TestGenerateRandomBytes(t *testing.T) {
	tests := []struct {
		name      string
		length    int
		wantError bool
	}{
		{
			name:      "zero length",
			length:    0,
			wantError: false,
		},
		{
			name:      "small length",
			length:    16,
			wantError: false,
		},
		{
			name:      "medium length",
			length:    1024,
			wantError: false,
		},
		{
			name:      "large length",
			length:    4096,
			wantError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := generateRandomBytes(tt.length)
			if (err != nil) != tt.wantError {
				t.Errorf("generateRandomBytes(%d) error = %v, wantError %v", tt.length, err, tt.wantError)
			}
			if !tt.wantError && len(result) != tt.length {
				t.Errorf("generateRandomBytes(%d) returned length %d, want %d", tt.length, len(result), tt.length)
			}
		})
	}
}

// TestGetSeed tests getSeed function
func TestGetSeed(t *testing.T) {
	tmpDir := t.TempDir()
	originalHomeDir := os.Getenv("HOME")
	os.Setenv("HOME", tmpDir)
	defer os.Setenv("HOME", originalHomeDir)

	tests := []struct {
		name      string
		seedLen   int
		seedFile  string
		wantLen   int
	}{
		{
			name:     "seed1",
			seedLen:  1024,
			seedFile: Seed1,
			wantLen:  1024,
		},
		{
			name:     "seed2",
			seedLen:  2048,
			seedFile: Seed2,
			wantLen:  2048,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := getSeed(tt.seedLen, tt.seedFile)
			if len(result) != tt.wantLen {
				t.Errorf("getSeed returned length %d, want %d", len(result), tt.wantLen)
			}

			// Verify file was created
			seedFilePath := filepath.Join(tmpDir, tt.seedFile)
			if _, err := os.Stat(seedFilePath); os.IsNotExist(err) {
				t.Error("Seed file should be created")
			}

			// Second call should read from file
			result2 := getSeed(tt.seedLen, tt.seedFile)
			if len(result2) != tt.wantLen {
				t.Errorf("Second getSeed call returned length %d, want %d", len(result2), tt.wantLen)
			}
			if string(result) != string(result2) {
				t.Error("Second getSeed call should return same content")
			}
		})
	}
}

// TestGetEncryptionKey tests getEncryptionKey function
func TestGetEncryptionKey(t *testing.T) {
	tmpDir := t.TempDir()
	originalHomeDir := os.Getenv("HOME")
	os.Setenv("HOME", tmpDir)
	defer os.Setenv("HOME", originalHomeDir)

	key1 := getEncryptionKey()
	key2 := getEncryptionKey()

	// Should return same key on multiple calls (cached)
	if len(key1) != 32 {
		t.Errorf("getEncryptionKey returned length %d, want 32", len(key1))
	}
	if string(key1) != string(key2) {
		t.Error("getEncryptionKey should return same key on multiple calls")
	}
}

// TestEncryptedFileCacheReadWrite tests EncryptedFileCacheReadWrite implementation
func TestEncryptedFileCacheReadWrite(t *testing.T) {
	tmpDir := t.TempDir()
	originalHomeDir := os.Getenv("HOME")
	os.Setenv("HOME", tmpDir)
	defer os.Setenv("HOME", originalHomeDir)

	cache := &EncryptedFileCacheReadWrite{}
	category := "test-category"
	key := "test-key"
	content := "test content"

	// Test Write
	err := cache.Write(category, key, content)
	if err != nil {
		t.Fatalf("Write failed: %v", err)
	}

	// Test Read
	readContent, err := cache.Read(category, key)
	if err != nil {
		t.Fatalf("Read failed: %v", err)
	}

	if readContent != content {
		t.Errorf("Read content does not match: got %q, want %q", readContent, content)
	}
}

// TestReadCacheFileWithEncryptionCallback_StopFallback tests stop fallback error handling
func TestReadCacheFileWithEncryptionCallback_StopFallback(t *testing.T) {
	tmpDir := t.TempDir()
	originalHomeDir := os.Getenv("HOME")
	os.Setenv("HOME", tmpDir)
	defer os.Setenv("HOME", originalHomeDir)

	category := "test-category"
	key := "test-key"

	options := &ReadCacheOptions{
		FetchContent: func() (int, string, error) {
			return http.StatusInternalServerError, "", fmt.Errorf(constants.ErrStopFallback)
		},
	}

	_, err := ReadCacheFileWithEncryptionCallback(category, key, options)
	if err == nil {
		t.Error("Should return error when fetch returns stop fallback error")
	}
	if err != nil && !assert.Contains(t, err.Error(), "user denied") {
		t.Errorf("Error should contain user denied message, got: %v", err)
	}
}

// BenchmarkEncryptText benchmarks the EncryptText function
func BenchmarkEncryptText(b *testing.B) {
	plaintext := "This is a benchmark test message for encryption performance"
	additionalData := []byte("benchmark-category-benchmark-key")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := EncryptText(plaintext, additionalData)
		if err != nil {
			b.Fatalf("EncryptText failed: %v", err)
		}
	}
}

// BenchmarkDecryptText benchmarks the DecryptText function
func BenchmarkDecryptText(b *testing.B) {
	plaintext := "This is a benchmark test message for decryption performance"
	additionalData := []byte("benchmark-category-benchmark-key")
	ciphertext, _ := EncryptText(plaintext, additionalData)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := DecryptText(ciphertext, additionalData)
		if err != nil {
			b.Fatalf("DecryptText failed: %v", err)
		}
	}
}

// BenchmarkReadCacheFileWithEncryptionCallback benchmarks the cache read callback
func BenchmarkReadCacheFileWithEncryptionCallback(b *testing.B) {
	tmpDir := b.TempDir()
	originalHomeDir := os.Getenv("HOME")
	os.Setenv("HOME", tmpDir)
	defer os.Setenv("HOME", originalHomeDir)

	category := "bench-category"
	key := "bench-key"
	content := "benchmark content"

	// Write cache file once
	_ = WriteCacheFileWithEncryption(category, key, content)

	options := &ReadCacheOptions{
		FetchContent: func() (int, string, error) {
			return http.StatusOK, content, nil
		},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := ReadCacheFileWithEncryptionCallback(category, key, options)
		if err != nil {
			b.Fatalf("ReadCacheFileWithEncryptionCallback failed: %v", err)
		}
	}
}
