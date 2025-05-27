package utils

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"github.com/aliyunidaas/alibaba-cloud-idaas/constants"
	"github.com/aliyunidaas/alibaba-cloud-idaas/idaaslog"
	"github.com/pkg/errors"
	"io"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"sync"
	"time"
)

const (
	Seed1 = ".alibaba_cloud_idaas_seed1"
	Seed2 = ".alibaba_cloud_idaas_seed2"
)

type StringWithTime struct {
	CacheTime int64                  `json:"cache_time"`
	Context   map[string]interface{} `json:"context"`
	Content   string                 `json:"content"`
}

func UnmarshalStringWithTime(str string) (*StringWithTime, error) {
	var stringWithTime StringWithTime
	err := json.Unmarshal([]byte(str), &stringWithTime)
	if err != nil {
		return nil, err
	}
	return &stringWithTime, nil
}

func (s *StringWithTime) IsExpired() bool {
	return (time.Now().UnixMilli() - s.CacheTime) > (3 * 24 * time.Hour.Milliseconds())
}

func (s *StringWithTime) IsExpiringOrExpired() bool {
	return (time.Now().UnixMilli() - s.CacheTime) > time.Hour.Milliseconds()
}

func (s *StringWithTime) Marshal() (string, error) {
	bs, err := json.Marshal(s)
	if err != nil {
		return "", err
	}
	return string(bs), nil
}

type ReadCacheFileOptions struct {
	Context                    map[string]interface{}
	FetchContent               func() (int, string, error)
	ForceNew                   bool
	AllowExpired               bool
	IsContentExpiringOrExpired func(time *StringWithTime) bool
	IsContentExpired           func(time *StringWithTime) bool
}

func ReadCacheFileWithEncryptionCallback(category, key string, options *ReadCacheFileOptions) (string, error) {
	var stringWithTime *StringWithTime
	data, err := ReadCacheFileWithEncryption(category, key)
	if err != nil {
		idaaslog.Warn.PrintfLn("Read cache file [%s, %s] with encryption failed: %v, ignore error",
			category, key, err)
	}
	if data != "" {
		stringWithTime, err = UnmarshalStringWithTime(data)
		if err != nil {
			idaaslog.Warn.PrintfLn("Parse cache file[%s, %s] with encryption failed: %v, ignore error",
				category, key, err)
		}
	}
	expiringOrExpired := true
	if stringWithTime != nil {
		if options.IsContentExpired != nil {
			expiringOrExpired = options.IsContentExpiringOrExpired(stringWithTime)
		} else {
			expiringOrExpired = stringWithTime.IsExpiringOrExpired()
		}
	}
	if options.ForceNew {
		expiringOrExpired = true
	}

	var fetchStatusCode int
	var fetchContent string
	var fetchContentErr error
	if expiringOrExpired {
		fetchStatusCode, fetchContent, fetchContentErr = options.FetchContent()
		if fetchContentErr != nil {
			idaaslog.Error.PrintfLn("Fetch content failed: %v", fetchContentErr)
		} else {
			if fetchStatusCode != http.StatusOK {
				idaaslog.Error.PrintfLn("Fetch content failed, statusCode: %d", fetchStatusCode)
			} else {
				stringWithTimeForStore := StringWithTime{
					CacheTime: time.Now().UnixMilli(),
					Context:   options.Context,
					Content:   fetchContent,
				}
				marshaledContent, err := stringWithTimeForStore.Marshal()
				if err != nil {
					idaaslog.Error.PrintfLn("Marshal content failed: %v", err)
				} else {
					err = WriteCacheFileWithEncryption(category, key, marshaledContent)
					if err != nil {
						idaaslog.Error.PrintfLn("Write content failed: %v", err)
					}
					return fetchContent, nil
				}
			}
		}
	}

	if options.ForceNew {
		return "", errors.New("fetch content failed, with ForceNew option")
	}

	if fetchContentErr != nil && strings.Contains(fetchContentErr.Error(), constants.ErrStopFallback) {
		return "", errors.New("user denied, stop fallback to local cached credentials")
	}

	if stringWithTime != nil {
		var expired bool
		if options.IsContentExpired != nil {
			expired = options.IsContentExpired(stringWithTime)
		} else {
			expired = stringWithTime.IsExpired()
		}
		if !expired {
			idaaslog.Warn.PrintfLn("Expired cache file [%s, %s], not expired", category, key)
			return stringWithTime.Content, nil
		}
		if options.AllowExpired {
			idaaslog.Error.PrintfLn("Expired cache file [%s, %s], allow expired", category, key)
			return stringWithTime.Content, nil
		}
	}
	return "", errors.Wrapf(fetchContentErr, "read cache file [%s, %s], context: %+v", category, key, options.Context)
}

func ReadCacheFileWithEncryption(category, key string) (string, error) {
	ciphertext, err := readCacheFile(category, key)
	if err != nil {
		return "", errors.Wrap(err, "read cache file failed")
	}
	if ciphertext == nil {
		return "", nil
	}
	additionalData := category + "-" + key
	plaintext, err := DecryptText(string(ciphertext), []byte(additionalData))
	if err != nil {
		return "", errors.Wrap(err, "decrypt ciphertext failed")
	}
	return plaintext, nil
}

func WriteCacheFileWithEncryption(category, key string, plaintext string) error {
	additionalData := category + "-" + key
	ciphertext, err := EncryptText(plaintext, []byte(additionalData))
	if err != nil {
		return errors.Wrapf(err, "encrypt plaintext failed")
	}
	return writeCacheFile(category, key, []byte(ciphertext))
}

func EncryptText(plaintext string, additionalData []byte) (string, error) {
	key := getEncryptionKey()
	block, err := aes.NewCipher(key)
	if err != nil {
		return "", errors.Wrap(err, "new cipher failed")
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", errors.Wrap(err, "new GCM failed")
	}
	nonce := make([]byte, gcm.NonceSize())
	if _, err = io.ReadFull(rand.Reader, nonce); err != nil {
		return "", errors.Wrap(err, "read nonce failed")
	}
	ciphertext := gcm.Seal(nil, nonce, []byte(plaintext), additionalData)
	return "encrypted:" +
		base64.RawURLEncoding.EncodeToString(nonce) + ":" +
		base64.RawURLEncoding.EncodeToString(ciphertext), nil
}

func DecryptText(ciphertext string, additionalData []byte) (string, error) {
	key := getEncryptionKey()
	block, err := aes.NewCipher(key)
	if err != nil {
		return "", errors.Wrap(err, "new cipher failed")
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", errors.Wrap(err, "new GCM failed")
	}

	parts := strings.Split(ciphertext, ":")
	if len(parts) != 3 {
		return "", errors.Errorf("invalid ciphertext, len() == %d", len(parts))
	}
	if parts[0] != "encrypted" {
		return "", errors.New("invalid ciphertext, not starts with encrypted")
	}

	nonce, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return "", errors.Wrapf(err, "invalid nonce: %s", parts[1])
	}
	ciphertextBody, err := base64.RawURLEncoding.DecodeString(parts[2])
	if err != nil {
		return "", errors.Wrapf(err, "invalid ciphertext: %s", parts[2])
	}
	plaintext, err := gcm.Open(nil, nonce, ciphertextBody, additionalData)
	if err != nil {
		return "", errors.Wrap(err, "decrypt failed")
	}
	return string(plaintext), nil
}

var encryptionKey []byte
var encryptionKeyOnce sync.Once

// Encrypted data is stored in local disk temporary,
// we use a combined encryption key ro protect(encrypt) data
func getEncryptionKey() []byte {
	encryptionKeyOnce.Do(func() {
		seed1 := getSeed1()
		seed2 := getSeed2()
		seed3 := []byte(getMacs())
		seed4 := []byte("ALIBABA_CLOUD_IDAAS_STATIC_PART")

		hash := sha256.New()
		_, _ = io.Copy(hash, bytes.NewReader(seed1))
		_, _ = io.Copy(hash, bytes.NewReader(seed2))
		_, _ = io.Copy(hash, bytes.NewReader(seed3))
		_, _ = io.Copy(hash, bytes.NewReader(seed4))
		encryptionKey = hash.Sum(nil)
	})
	return encryptionKey
}

func getSeed1() []byte {
	return getSeed(1024, Seed1)
}

func getSeed2() []byte {
	return getSeed(2048, Seed2)
}

func getMacs() string {
	interfaces, err := net.Interfaces()
	if err != nil {
		// ignore error here
		return "unknown_mac"
	}
	var macs []string
	for _, iface := range interfaces {
		if len(iface.HardwareAddr) == 0 {
			continue
		}
		macs = append(macs, iface.HardwareAddr.String())
	}
	slices.Sort(macs)
	return strings.Join(macs, ",")
}

func writeCacheFile(category, key string, content []byte) error {
	cacheFile, err := getCacheFile(category, key)
	if err != nil {
		return err
	}
	return os.WriteFile(cacheFile, content, 0600)
}

func readCacheFile(category, key string) ([]byte, error) {
	cacheFile, err := getCacheFile(category, key)
	if err != nil {
		return nil, err
	}
	if _, err := os.Stat(cacheFile); os.IsNotExist(err) {
		return nil, nil
	}
	return os.ReadFile(cacheFile)
}

func getCacheFile(category, key string) (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", errors.Wrap(err, "get home dir failed")
	}
	cacheDir := filepath.Join(homeDir, constants.DotAliyunDir, constants.AlibabaCloudIdaasDir, category)
	if _, err := os.Stat(cacheDir); os.IsNotExist(err) {
		mkdirErr := os.MkdirAll(cacheDir, 0755)
		if mkdirErr != nil {
			return "", mkdirErr
		}
	}
	return filepath.Join(cacheDir, key), nil
}

func getSeed(seedLen int, seedFilename string) []byte {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		// ignore error here
		return nil
	}
	seedFullFilename := filepath.Join(homeDir, seedFilename)
	if _, err := os.Stat(seedFullFilename); os.IsNotExist(err) {
		seedFile, createErr := os.Create(seedFullFilename)
		if createErr != nil {
			return nil
		}
		defer seedFile.Close()

		randomBytes, randErr := generateRandomBytes(seedLen)
		if randErr != nil {
			return nil
		}
		_, writeErr := seedFile.Write(randomBytes)
		if writeErr != nil {
			return nil
		}
		return randomBytes
	}
	seedFile, openErr := os.Open(seedFullFilename)
	if openErr != nil {
		return nil
	}
	defer seedFile.Close()
	content, readErr := io.ReadAll(seedFile)
	if readErr != nil {
		return nil
	}
	return content
}

func generateRandomBytes(n int) ([]byte, error) {
	b := make([]byte, n)
	_, err := rand.Read(b)
	if err != nil {
		return nil, errors.Wrap(err, "failed to generate random bytes")
	}
	return b, nil
}
