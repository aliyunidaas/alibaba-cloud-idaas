package config

import (
	"encoding/base64"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

// TestTryParseProfileFromInput tests the TryParseProfileFromInput function
func TestTryParseProfileFromInput(t *testing.T) {
	tests := []struct {
		name     string
		profile  string
		wantTemp bool
		wantNil  bool
	}{
		{
			name:    "empty profile",
			profile: "",
			wantNil: true,
		},
		{
			name:    "invalid json string",
			profile: "invalid json",
			wantNil: true,
		},
		{
			name:     "valid json string",
			profile:  `{"alibaba_cloud_sts":{"region":"cn-hangzhou","sts_endpoint":"sts.cn-hangzhou.aliyuncs.com","oidc_provider_arn":"acs:ram::123456:oidc-provider/test","role_arn":"acs:ram::123456:role/test","oidc_token_provider":{"token_type":"id_token","client_credentials":{"token_endpoint":"https://example.com/token","client_id":"test-client"}}}}`,
			wantTemp: true,
			wantNil:  false,
		},
		{
			name:     "valid base64 encoded json",
			profile:  base64.StdEncoding.EncodeToString([]byte(`{"aws_sts":{"region":"us-east-1","role_arn":"arn:aws:iam::123456:role/test","oidc_token_provider":{"token_type":"id_token","client_credentials":{"token_endpoint":"https://example.com/token","client_id":"test-client"}}}}`)),
			wantTemp: true,
			wantNil:  false,
		},
		{
			name:     "invalid base64 string",
			profile:  "not-base64-$$$",
			wantNil:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tempProfile, cloudStsConfig := TryParseProfileFromInput(tt.profile)
			if tt.wantNil {
				if cloudStsConfig != nil {
					t.Errorf("TryParseProfileFromInput(%q) = %v, want nil", tt.profile, cloudStsConfig)
				}
				if tt.wantTemp && tempProfile == tt.profile {
					t.Errorf("TryParseProfileFromInput(%q) = tempProfile %q, want temp profile", tt.profile, tempProfile)
				}
			} else {
				if cloudStsConfig == nil {
					t.Errorf("TryParseProfileFromInput(%q) = nil, want non-nil", tt.profile)
				}
				if !tt.wantTemp && tempProfile != tt.profile {
					t.Errorf("TryParseProfileFromInput(%q) = tempProfile %q, want %q", tt.profile, tempProfile, tt.profile)
				}
			}
		})
	}
}

// TestCloudCredentialConfig_FindProfile tests the FindProfile method
func TestCloudCredentialConfig_FindProfile(t *testing.T) {
	tests := []struct {
		name           string
		config         *CloudCredentialConfig
		profile        string
		wantProfile    string
		wantFound      bool
	}{
		{
			name:           "nil config",
			config:         nil,
			profile:        "test",
			wantProfile:    "",
			wantFound:      false,
		},
		{
			name: "empty profile uses current profile",
			config: &CloudCredentialConfig{
				CurrentProfile: "current",
				Profile: map[string]*CloudStsConfig{
					"current": {Comment: "current profile"},
				},
			},
			profile:     "",
			wantProfile: "current",
			wantFound:   true,
		},
		{
			name: "empty profile uses default",
			config: &CloudCredentialConfig{
				Profile: map[string]*CloudStsConfig{
					"default": {Comment: "default profile"},
				},
			},
			profile:     "",
			wantProfile: "default",
			wantFound:   true,
		},
		{
			name: "profile found",
			config: &CloudCredentialConfig{
				Profile: map[string]*CloudStsConfig{
					"test": {Comment: "test profile"},
				},
			},
			profile:     "test",
			wantProfile: "test",
			wantFound:   true,
		},
		{
			name: "profile not found",
			config: &CloudCredentialConfig{
				Profile: map[string]*CloudStsConfig{
					"other": {Comment: "other profile"},
				},
			},
			profile:     "test",
			wantProfile: "test",
			wantFound:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotProfile, gotConfig := tt.config.FindProfile(tt.profile)
			if gotProfile != tt.wantProfile {
				t.Errorf("FindProfile(%q) = profile %q, want %q", tt.profile, gotProfile, tt.wantProfile)
			}
			if (gotConfig != nil) != tt.wantFound {
				t.Errorf("FindProfile(%q) = config %v, want found=%v", tt.profile, gotConfig, tt.wantFound)
			}
		})
	}
}

// TestCloudAccountTokenConfig_GetInstanceId tests GetInstanceId method
func TestCloudAccountTokenConfig_GetInstanceId(t *testing.T) {
	tests := []struct {
		name           string
		config         *CloudAccountTokenConfig
		wantInstanceId string
	}{
		{
			name:           "nil config",
			config:         nil,
			wantInstanceId: "",
		},
		{
			name:           "InstanceId takes precedence",
			config:         &CloudAccountTokenConfig{InstanceId: "instance-1", CloudAccountInstanceId: "old-instance"},
			wantInstanceId: "instance-1",
		},
		{
			name:           "fallback to CloudAccountInstanceId",
			config:         &CloudAccountTokenConfig{CloudAccountInstanceId: "old-instance"},
			wantInstanceId: "old-instance",
		},
		{
			name:           "both empty",
			config:         &CloudAccountTokenConfig{},
			wantInstanceId: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.config.GetInstanceId()
			if got != tt.wantInstanceId {
				t.Errorf("GetInstanceId() = %q, want %q", got, tt.wantInstanceId)
			}
		})
	}
}

// TestCloudAccountTokenConfig_GetEndpoint tests GetEndpoint method
func TestCloudAccountTokenConfig_GetEndpoint(t *testing.T) {
	tests := []struct {
		name         string
		config       *CloudAccountTokenConfig
		wantEndpoint string
	}{
		{
			name:         "nil config",
			config:       nil,
			wantEndpoint: "",
		},
		{
			name:         "DeveloperApiEndpoint takes precedence",
			config:       &CloudAccountTokenConfig{DeveloperApiEndpoint: "https://new.example.com", CloudAccountEndpoint: "https://old.example.com"},
			wantEndpoint: "https://new.example.com",
		},
		{
			name:         "fallback to CloudAccountEndpoint",
			config:       &CloudAccountTokenConfig{CloudAccountEndpoint: "https://old.example.com"},
			wantEndpoint: "https://old.example.com",
		},
		{
			name:         "both empty",
			config:       &CloudAccountTokenConfig{},
			wantEndpoint: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.config.GetEndpoint()
			if got != tt.wantEndpoint {
				t.Errorf("GetEndpoint() = %q, want %q", got, tt.wantEndpoint)
			}
		})
	}
}

// TestCloudAccountTokenConfig_GetCloudAccountEndpoint tests GetCloudAccountEndpoint method
func TestCloudAccountTokenConfig_GetCloudAccountEndpoint(t *testing.T) {
	tests := []struct {
		name          string
		config        *CloudAccountTokenConfig
		wantEndpoint  string
		wantError     bool
	}{
		{
			name:         "nil config",
			config:       nil,
			wantEndpoint: "",
			wantError:    true,
		},
		{
			name:          "CloudAccountEndpoint provided",
			config:        &CloudAccountTokenConfig{CloudAccountEndpoint: "https://custom.example.com"},
			wantEndpoint:  "https://custom.example.com",
			wantError:     false,
		},
		{
			name:          "DeveloperApiEndpoint with InstanceId",
			config:        &CloudAccountTokenConfig{DeveloperApiEndpoint: "https://api.example.com", InstanceId: "instance-123"},
			wantEndpoint:  "https://api.example.com/v2/instance-123/cloudAccountRoles/_/actions/obtainAccessCredential",
			wantError:     false,
		},
		{
			name:          "InstanceId takes precedence",
			config:        &CloudAccountTokenConfig{DeveloperApiEndpoint: "https://api.example.com", InstanceId: "instance-123", CloudAccountInstanceId: "old-instance"},
			wantEndpoint:  "https://api.example.com/v2/instance-123/cloudAccountRoles/_/actions/obtainAccessCredential",
			wantError:     false,
		},
		{
			name:          "missing region and instance",
			config:        &CloudAccountTokenConfig{},
			wantEndpoint:  "",
			wantError:     true,
		},
		{
			name:          "missing region",
			config:        &CloudAccountTokenConfig{InstanceId: "instance-123"},
			wantEndpoint:  "",
			wantError:     true,
		},
		{
			name:          "missing instance",
			config:        &CloudAccountTokenConfig{CloudAccountRegion: "cn-hangzhou"},
			wantEndpoint:  "",
			wantError:     true,
		},
		{
			name:          "region and instance provided",
			config:        &CloudAccountTokenConfig{CloudAccountRegion: "cn-hangzhou", InstanceId: "instance-123"},
			wantEndpoint:  "https://eiam-developerapi.cn-hangzhou.aliyuncs.com/v2/instance-123/cloudAccountRoles/_/actions/obtainAccessCredential",
			wantError:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tt.config.GetCloudAccountEndpoint()
			if (err != nil) != tt.wantError {
				t.Errorf("GetCloudAccountEndpoint() error = %v, wantError %v", err, tt.wantError)
			}
			if got != tt.wantEndpoint {
				t.Errorf("GetCloudAccountEndpoint() = %q, want %q", got, tt.wantEndpoint)
			}
		})
	}
}

// TestAgentConfig_CreateUserExclusiveCredentialEndpoint tests CreateUserExclusiveCredentialEndpoint method
func TestAgentConfig_CreateUserExclusiveCredentialEndpoint(t *testing.T) {
	tests := []struct {
		name         string
		config       *AgentConfig
		wantEndpoint string
		wantError    bool
	}{
		{
			name:         "nil config",
			config:       nil,
			wantEndpoint: "",
			wantError:    true,
		},
		{
			name:         "missing instance_id",
			config:       &AgentConfig{DeveloperApiEndpoint: "https://api.example.com"},
			wantEndpoint: "",
			wantError:    true,
		},
		{
			name:         "missing developer_api_endpoint",
			config:       &AgentConfig{InstanceId: "instance-123"},
			wantEndpoint: "/v2/instance-123/credentials/_/actions/createUserExclusive",
			wantError:    false,
		},
		{
			name:         "valid config",
			config:       &AgentConfig{InstanceId: "instance-123", DeveloperApiEndpoint: "https://api.example.com"},
			wantEndpoint: "https://api.example.com/v2/instance-123/credentials/_/actions/createUserExclusive",
			wantError:    false,
		},
		{
			name:         "instance domain endpoint",
			config:       &AgentConfig{InstanceId: "instance-123", DeveloperApiEndpoint: "https://example.aliyunidaas.com"},
			wantEndpoint: "https://example.aliyunidaas.com/api/v2/credentials/_/actions/createUserExclusive",
			wantError:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tt.config.CreateUserExclusiveCredentialEndpoint()
			if (err != nil) != tt.wantError {
				t.Errorf("CreateUserExclusiveCredentialEndpoint() error = %v, wantError %v", err, tt.wantError)
			}
			if got != tt.wantEndpoint {
				t.Errorf("CreateUserExclusiveCredentialEndpoint() = %q, want %q", got, tt.wantEndpoint)
			}
		})
	}
}

// TestAgentConfig_GetCredentialEndpoint tests GetCredentialEndpoint method
func TestAgentConfig_GetCredentialEndpoint(t *testing.T) {
	tests := []struct {
		name         string
		config       *AgentConfig
		wantEndpoint string
		wantError    bool
	}{
		{
			name:         "nil config",
			config:       nil,
			wantEndpoint: "",
			wantError:    true,
		},
		{
			name:         "missing instance_id",
			config:       &AgentConfig{DeveloperApiEndpoint: "https://api.example.com"},
			wantEndpoint: "",
			wantError:    true,
		},
		{
			name:         "missing developer_api_endpoint",
			config:       &AgentConfig{InstanceId: "instance-123"},
			wantEndpoint: "/v2/instance-123/credentials/_/actions/obtain",
			wantError:    false,
		},
		{
			name:         "valid config",
			config:       &AgentConfig{InstanceId: "instance-123", DeveloperApiEndpoint: "https://api.example.com"},
			wantEndpoint: "https://api.example.com/v2/instance-123/credentials/_/actions/obtain",
			wantError:    false,
		},
		{
			name:         "instance domain endpoint",
			config:       &AgentConfig{InstanceId: "instance-123", DeveloperApiEndpoint: "https://example.aliyunidaas.com"},
			wantEndpoint: "https://example.aliyunidaas.com/api/v2/credentials/_/actions/obtain",
			wantError:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tt.config.GetCredentialEndpoint()
			if (err != nil) != tt.wantError {
				t.Errorf("GetCredentialEndpoint() error = %v, wantError %v", err, tt.wantError)
			}
			if got != tt.wantEndpoint {
				t.Errorf("GetCredentialEndpoint() = %q, want %q", got, tt.wantEndpoint)
			}
		})
	}
}

// TestAgentConfig_Clone tests Clone method
func TestAgentConfig_Clone(t *testing.T) {
	tests := []struct {
		name    string
		config  *AgentConfig
		wantNil bool
	}{
		{
			name:    "nil config",
			config:  nil,
			wantNil: true,
		},
		{
			name: "valid config",
			config: &AgentConfig{
				InstanceId:           "instance-123",
				DeveloperApiEndpoint: "https://api.example.com",
				AccessTokenProvider: &OidcTokenProviderConfig{
					TokenType: "id_token",
					OidcTokenProviderClientCredentials: &OidcTokenProviderClientCredentialsConfig{
						ClientId: "test-client",
					},
				},
			},
			wantNil: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cloned, err := tt.config.Clone()
			if err != nil {
				t.Errorf("Clone() error = %v", err)
			}
			if tt.wantNil {
				if cloned != nil {
					t.Errorf("Clone() = %v, want nil", cloned)
				}
			} else {
				if cloned == nil {
					t.Errorf("Clone() = nil, want non-nil")
				} else {
					if cloned.InstanceId != tt.config.InstanceId {
						t.Errorf("Clone() InstanceId = %q, want %q", cloned.InstanceId, tt.config.InstanceId)
					}
					if cloned.DeveloperApiEndpoint != tt.config.DeveloperApiEndpoint {
						t.Errorf("Clone() DeveloperApiEndpoint = %q, want %q", cloned.DeveloperApiEndpoint, tt.config.DeveloperApiEndpoint)
					}
					if cloned.AccessTokenProvider.GetId() != tt.config.AccessTokenProvider.GetId() {
						t.Errorf("Clone() AccessTokenProvider.GetId() = %q, want %q", cloned.AccessTokenProvider.GetId(), tt.config.AccessTokenProvider.GetId())
					}
				}
			}
		})
	}
}

// TestOidcTokenProviderConfig_GetCacheKey tests GetCacheKey method
func TestOidcTokenProviderConfig_GetCacheKey(t *testing.T) {
	tests := []struct {
		name    string
		config  *OidcTokenProviderConfig
		wantKey string
	}{
		{
			name: "client credentials",
			config: &OidcTokenProviderConfig{
				OidcTokenProviderClientCredentials: &OidcTokenProviderClientCredentialsConfig{
					ClientId: "test-client",
				},
			},
			wantKey: "test-client_",
		},
		{
			name: "device code",
			config: &OidcTokenProviderConfig{
				OidcTokenProviderDeviceCode: &OidcTokenProviderDeviceCodeConfig{
					ClientId: "device-client",
				},
			},
			wantKey: "device-client_",
		},
		{
			name:    "unknown",
			config:  &OidcTokenProviderConfig{},
			wantKey: "unknown_oidc_",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.config.GetCacheKey()
			if len(got) < len(tt.wantKey) {
				t.Errorf("GetCacheKey() returned %q, want at least %d chars", got, len(tt.wantKey))
			}
			if got[:len(tt.wantKey)] != tt.wantKey {
				t.Errorf("GetCacheKey() = %q, want prefix %q", got[:len(tt.wantKey)], tt.wantKey)
			}
		})
	}
}

// TestOidcTokenProviderConfig_GetId tests GetId method
func TestOidcTokenProviderConfig_GetId(t *testing.T) {
	tests := []struct {
		name    string
		config  *OidcTokenProviderConfig
		wantId  string
	}{
		{
			name: "client credentials",
			config: &OidcTokenProviderConfig{
				OidcTokenProviderClientCredentials: &OidcTokenProviderClientCredentialsConfig{
					ClientId: "test-client",
				},
			},
			wantId: "test-client",
		},
		{
			name: "device code",
			config: &OidcTokenProviderConfig{
				OidcTokenProviderDeviceCode: &OidcTokenProviderDeviceCodeConfig{
					ClientId: "device-client",
				},
			},
			wantId: "device-client",
		},
		{
			name:   "unknown",
			config: &OidcTokenProviderConfig{},
			wantId: "unknown_oidc",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.config.GetId()
			if got != tt.wantId {
				t.Errorf("GetId() = %q, want %q", got, tt.wantId)
			}
		})
	}
}

// TestOidcTokenProviderConfig_Marshal tests Marshal method
func TestOidcTokenProviderConfig_Marshal(t *testing.T) {
	tests := []struct {
		name    string
		config  *OidcTokenProviderConfig
		wantNil bool
	}{
		{
			name:    "nil config",
			config:  nil,
			wantNil: true,
		},
		{
			name: "valid config",
			config: &OidcTokenProviderConfig{
				TokenType: "id_token",
				OidcTokenProviderClientCredentials: &OidcTokenProviderClientCredentialsConfig{
					ClientId: "test-client",
				},
			},
			wantNil: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.config.Marshal()
			if tt.wantNil {
				if got != "\"null\"" {
					t.Errorf("Marshal() = %q, want \"null\"", got)
				}
			} else {
				var unmarshaled OidcTokenProviderConfig
				err := json.Unmarshal([]byte(got), &unmarshaled)
				if err != nil {
					t.Errorf("Marshal() returned invalid JSON: %v", err)
				}
			}
		})
	}
}

// TestCheckEndpointAndInstanceId tests checkEndpointAndInstanceId function
func TestCheckEndpointAndInstanceId(t *testing.T) {
	tests := []struct {
		name     string
		endpoint string
		instance string
		want     bool
	}{
		{
			name:     "instance domain endpoint",
			endpoint: "https://example.aliyunidaas.com",
			instance: "",
			want:     true,
		},
		{
			name:     "instance domain endpoint with cloud-idaas",
			endpoint: "https://example.cloud-idaas.com",
			instance: "",
			want:     true,
		},
		{
			name:     "non-instance domain with instance",
			endpoint: "https://api.example.com",
			instance: "instance-123",
			want:     true,
		},
		{
			name:     "non-instance domain without instance",
			endpoint: "https://api.example.com",
			instance: "",
			want:     false,
		},
		{
			name:     "empty endpoint and instance",
			endpoint: "",
			instance: "",
			want:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := checkEndpointAndInstanceId(tt.endpoint, tt.instance)
			if got != tt.want {
				t.Errorf("checkEndpointAndInstanceId(%q, %q) = %v, want %v", tt.endpoint, tt.instance, got, tt.want)
			}
		})
	}
}

// TestBuildDeveloperApiUrl tests buildDeveloperApiUrl function
func TestBuildDeveloperApiUrl(t *testing.T) {
	tests := []struct {
		name     string
		endpoint string
		instance string
		path     string
		want     string
	}{
		{
			name:     "instance domain",
			endpoint: "https://example.aliyunidaas.com",
			instance: "instance-123",
			path:     "/test",
			want:     "https://example.aliyunidaas.com/api/v2/test",
		},
		{
			name:     "non-instance domain",
			endpoint: "https://api.example.com",
			instance: "instance-123",
			path:     "/test",
			want:     "https://api.example.com/v2/instance-123/test",
		},
		{
			name:     "instance domain with trailing slash",
			endpoint: "https://example.aliyunidaas.com/",
			instance: "instance-123",
			path:     "/test",
			want:     "https://example.aliyunidaas.com/api/v2/test",
		},
		{
			name:     "non-instance domain with trailing slash",
			endpoint: "https://api.example.com/",
			instance: "instance-123",
			path:     "/test",
			want:     "https://api.example.com/v2/instance-123/test",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := buildDeveloperApiUrl(tt.endpoint, tt.instance, tt.path)
			if got != tt.want {
				t.Errorf("buildDeveloperApiUrl(%q, %q, %q) = %q, want %q", tt.endpoint, tt.instance, tt.path, got, tt.want)
			}
		})
	}
}

// TestIsInstanceDomain tests isInstanceDomain function
func TestIsInstanceDomain(t *testing.T) {
	tests := []struct {
		name     string
		endpoint string
		want     bool
	}{
		{
			name:     "aliyunidaas domain",
			endpoint: "https://example.aliyunidaas.com",
			want:     true,
		},
		{
			name:     "cloud-idaas domain",
			endpoint: "https://example.cloud-idaas.com",
			want:     true,
		},
		{
			name:     "aliyunidaas domain mixed case",
			endpoint: "https://Example.AliyunIdaas.COM",
			want:     true,
		},
		{
			name:     "regular domain",
			endpoint: "https://api.example.com",
			want:     false,
		},
		{
			name:     "empty string",
			endpoint: "",
			want:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isInstanceDomain(tt.endpoint)
			if got != tt.want {
				t.Errorf("isInstanceDomain(%q) = %v, want %v", tt.endpoint, got, tt.want)
			}
		})
	}
}

// TestLoadCloudCredentialConfig tests LoadCloudCredentialConfig function
func TestLoadCloudCredentialConfig(t *testing.T) {
	// Create a temporary directory for test files
	tmpDir := t.TempDir()

	tests := []struct {
		name        string
		filename    string
		content     string
		wantError   bool
		wantVersion string
	}{
		{
			name:      "non-existent file",
			filename:  filepath.Join(tmpDir, "nonexistent.json"),
			wantError: true,
		},
		{
			name:     "invalid JSON",
			filename: filepath.Join(tmpDir, "invalid.json"),
			content:  "invalid json content",
			wantError: true,
		},
		{
			name:     "valid config",
			filename: filepath.Join(tmpDir, "valid.json"),
			content: `{
				"version": "1",
				"current_profile": "default",
				"profile": {
					"default": {
						"alibaba_cloud_sts": {
							"region": "cn-hangzhou",
							"sts_endpoint": "sts.cn-hangzhou.aliyuncs.com",
							"oidc_provider_arn": "acs:ram::123456:oidc-provider/test",
							"role_arn": "acs:ram::123456:role/test",
							"oidc_token_provider": {
								"token_type": "id_token",
								"client_credentials": {
									"token_endpoint": "https://example.com/token",
									"client_id": "test-client"
								}
							}
						}
					}
				}
			}`,
			wantError:   false,
			wantVersion: "1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.content != "" {
				err := os.WriteFile(tt.filename, []byte(tt.content), 0644)
				if err != nil {
					t.Fatalf("Failed to write test file: %v", err)
				}
			}

			config, err := LoadCloudCredentialConfig(tt.filename)
			if (err != nil) != tt.wantError {
				t.Errorf("LoadCloudCredentialConfig() error = %v, wantError %v", err, tt.wantError)
			}
			if !tt.wantError && config != nil {
				if config.Version != tt.wantVersion {
					t.Errorf("LoadCloudCredentialConfig() Version = %q, want %q", config.Version, tt.wantVersion)
				}
			}
		})
	}
}

// TestFindProfile tests FindProfile function
func TestFindProfile(t *testing.T) {
	tmpDir := t.TempDir()

	tests := []struct {
		name              string
		filename          string
		profile           string
		ignoreParse       bool
		content           string
		wantError         bool
		wantProfileFound  bool
	}{
		{
			name:     "profile not found",
			filename: filepath.Join(tmpDir, "config1.json"),
			profile:  "nonexistent",
			content: `{
				"version": "1",
				"profile": {
					"default": {
						"alibaba_cloud_sts": {
							"region": "cn-hangzhou",
							"sts_endpoint": "sts.cn-hangzhou.aliyuncs.com",
							"oidc_provider_arn": "acs:ram::123456:oidc-provider/test",
							"role_arn": "acs:ram::123456:role/test",
							"oidc_token_provider": {
								"token_type": "id_token",
								"client_credentials": {
									"token_endpoint": "https://example.com/token",
									"client_id": "test-client"
								}
							}
						}
					}
				}
			}`,
			wantError:        true,
			wantProfileFound: false,
		},
		{
			name:     "profile found",
			filename: filepath.Join(tmpDir, "config2.json"),
			profile:  "default",
			content: `{
				"version": "1",
				"profile": {
					"default": {
						"alibaba_cloud_sts": {
							"region": "cn-hangzhou",
							"sts_endpoint": "sts.cn-hangzhou.aliyuncs.com",
							"oidc_provider_arn": "acs:ram::123456:oidc-provider/test",
							"role_arn": "acs:ram::123456:role/test",
							"oidc_token_provider": {
								"token_type": "id_token",
								"client_credentials": {
									"token_endpoint": "https://example.com/token",
									"client_id": "test-client"
								}
							}
						}
					}
				}
			}`,
			wantError:        false,
			wantProfileFound: true,
		},
		{
			name:        "parse from input profile",
			filename:    filepath.Join(tmpDir, "config3.json"),
			profile:     `{"alibaba_cloud_sts":{"region":"cn-hangzhou","sts_endpoint":"sts.cn-hangzhou.aliyuncs.com","oidc_provider_arn":"acs:ram::123456:oidc-provider/test","role_arn":"acs:ram::123456:role/test","oidc_token_provider":{"token_type":"id_token","client_credentials":{"token_endpoint":"https://example.com/token","client_id":"test-client"}}}}`,
			ignoreParse: false,
			wantError:   false,
			wantProfileFound: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.content != "" {
				err := os.WriteFile(tt.filename, []byte(tt.content), 0644)
				if err != nil {
					t.Fatalf("Failed to write test file: %v", err)
				}
			}

			gotProfile, gotConfig, err := FindProfile(tt.filename, tt.profile, tt.ignoreParse)
			if (err != nil) != tt.wantError {
				t.Errorf("FindProfile() error = %v, wantError %v", err, tt.wantError)
			}
			if tt.wantProfileFound && gotConfig == nil {
				t.Errorf("FindProfile() got nil config, want non-nil")
			}
			if !tt.wantProfileFound && gotConfig != nil {
				t.Errorf("FindProfile() got config %v, want nil", gotConfig)
			}
			if tt.wantProfileFound && gotProfile == "" {
				t.Errorf("FindProfile() got empty profile name")
			}
		})
	}
}

// TestDigest tests Digest method on OidcTokenProviderConfig
func TestOidcTokenProviderConfig_Digest(t *testing.T) {
	tests := []struct {
		name   string
		config *OidcTokenProviderConfig
	}{
		{
			name: "client credentials",
			config: &OidcTokenProviderConfig{
				TokenType: "id_token",
				OidcTokenProviderClientCredentials: &OidcTokenProviderClientCredentialsConfig{
					ClientId:      "test-client",
					TokenEndpoint: "https://example.com/token",
				},
			},
		},
		{
			name: "device code",
			config: &OidcTokenProviderConfig{
				TokenType: "id_token",
				OidcTokenProviderDeviceCode: &OidcTokenProviderDeviceCodeConfig{
					ClientId: "device-client",
					Issuer:   "https://issuer.example.com",
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.config.Digest()
			if len(got) != 64 { // SHA256 hex is 64 characters
				t.Errorf("Digest() returned %d chars, want 64", len(got))
			}
		})
	}
}

// TestCloudStsConfig_Digest tests Digest method on CloudStsConfig
func TestCloudStsConfig_Digest(t *testing.T) {
	tests := []struct {
		name   string
		config *CloudStsConfig
	}{
		{
			name: "alibaba cloud sts",
			config: &CloudStsConfig{
				AlibabaCloud: &AlibabaCloudStsConfig{
					Region:          "cn-hangzhou",
					StsEndpoint:     "sts.cn-hangzhou.aliyuncs.com",
					OidcProviderArn: "acs:ram::123456:oidc-provider/test",
					RoleArn:         "acs:ram::123456:role/test",
				},
			},
		},
		{
			name: "aws sts",
			config: &CloudStsConfig{
				Aws: &AwsCloudStsConfig{
					Region:  "us-east-1",
					RoleArn: "arn:aws:iam::123456:role/test",
				},
			},
		},
		{
			name: "cloud account",
			config: &CloudStsConfig{
				CloudAccount: &CloudAccountTokenConfig{
					InstanceId: "instance-123",
				},
			},
		},
		{
			name: "agent",
			config: &CloudStsConfig{
				Agent: &AgentConfig{
					InstanceId: "instance-123",
				},
			},
		},
		{
			name: "nil config",
			config: &CloudStsConfig{
				OidcToken: &OidcTokenProviderConfig{},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.config.Digest()
			if len(got) != 64 {
				t.Errorf("Digest() returned %d chars, want 64", len(got))
			}
		})
	}
}

// TestAlibabaCloudStsConfig_Digest tests Digest method on AlibabaCloudStsConfig
func TestAlibabaCloudStsConfig_Digest(t *testing.T) {
	tests := []struct {
		name   string
		config *AlibabaCloudStsConfig
	}{
		{
			name: "full config",
			config: &AlibabaCloudStsConfig{
				Region:          "cn-hangzhou",
				StsEndpoint:     "sts.cn-hangzhou.aliyuncs.com",
				OidcProviderArn: "acs:ram::123456:oidc-provider/test",
				RoleArn:         "acs:ram::123456:role/test",
				DurationSeconds: 3600,
				RoleSessionName: "test-session",
			},
		},
		{
			name: "minimal config",
			config: &AlibabaCloudStsConfig{
				StsEndpoint:     "sts.cn-hangzhou.aliyuncs.com",
				OidcProviderArn: "acs:ram::123456:oidc-provider/test",
				RoleArn:         "acs:ram::123456:role/test",
			},
		},
		{
			name:   "nil config",
			config: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.config.Digest()
			if len(got) != 64 && tt.config != nil {
				t.Errorf("Digest() returned %d chars, want 64", len(got))
			}
			if tt.config == nil && got != "" {
				t.Errorf("Digest() on nil returned %q, want empty string", got)
			}
		})
	}
}

// TestAwsCloudStsConfig_Digest tests Digest method on AwsCloudStsConfig
func TestAwsCloudStsConfig_Digest(t *testing.T) {
	tests := []struct {
		name   string
		config *AwsCloudStsConfig
	}{
		{
			name: "full config",
			config: &AwsCloudStsConfig{
				Region:          "us-east-1",
				RoleArn:         "arn:aws:iam::123456:role/test",
				DurationSeconds: 3600,
				RoleSessionName: "test-session",
			},
		},
		{
			name: "minimal config",
			config: &AwsCloudStsConfig{
				Region:  "us-east-1",
				RoleArn: "arn:aws:iam::123456:role/test",
			},
		},
		{
			name:   "nil config",
			config: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.config.Digest()
			if len(got) != 64 && tt.config != nil {
				t.Errorf("Digest() returned %d chars, want 64", len(got))
			}
			if tt.config == nil && got != "" {
				t.Errorf("Digest() on nil returned %q, want empty string", got)
			}
		})
	}
}

// TestOidcTokenProviderClientCredentialsConfig_Digest tests Digest method
func TestOidcTokenProviderClientCredentialsConfig_Digest(t *testing.T) {
	tests := []struct {
		name   string
		config *OidcTokenProviderClientCredentialsConfig
	}{
		{
			name: "full config",
			config: &OidcTokenProviderClientCredentialsConfig{
				TokenEndpoint:                      "https://example.com/token",
				ClientId:                           "test-client",
				Scope:                              "openid",
				ApplicationFederatedCredentialName: "test-app",
			},
		},
		{
			name: "minimal config",
			config: &OidcTokenProviderClientCredentialsConfig{
				TokenEndpoint: "https://example.com/token",
				ClientId:      "test-client",
			},
		},
		{
			name:   "nil config",
			config: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.config.Digest()
			if len(got) != 64 && tt.config != nil {
				t.Errorf("Digest() returned %d chars, want 64", len(got))
			}
			if tt.config == nil && got != "" {
				t.Errorf("Digest() on nil returned %q, want empty string", got)
			}
		})
	}
}

// TestOidcTokenProviderDeviceCodeConfig_Digest tests Digest method
func TestOidcTokenProviderDeviceCodeConfig_Digest(t *testing.T) {
	tests := []struct {
		name   string
		config *OidcTokenProviderDeviceCodeConfig
	}{
		{
			name: "full config",
			config: &OidcTokenProviderDeviceCodeConfig{
				Issuer:       "https://issuer.example.com",
				ClientId:     "test-client",
				Scope:        "openid",
				ClientSecret: "secret",
				AutoOpenUrl:  true,
				ShowQrCode:   true,
			},
		},
		{
			name: "minimal config",
			config: &OidcTokenProviderDeviceCodeConfig{
				Issuer:   "https://issuer.example.com",
				ClientId: "test-client",
			},
		},
		{
			name:   "nil config",
			config: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.config.Digest()
			if len(got) != 64 && tt.config != nil {
				t.Errorf("Digest() returned %d chars, want 64", len(got))
			}
			if tt.config == nil && got != "" {
				t.Errorf("Digest() on nil returned %q, want empty string", got)
			}
		})
	}
}

// TestOpenApiConfig_Digest tests Digest method
func TestOpenApiConfig_Digest(t *testing.T) {
	tests := []struct {
		name   string
		config *OpenApiConfig
	}{
		{
			name: "full config",
			config: &OpenApiConfig{
				InstanceId:      "instance-123",
				ApplicationId:   "app-456",
				ScopeValues:     []string{"openid", "profile"},
				Audience:        "test-audience",
				OpenApiEndpoint: "https://api.example.com",
			},
		},
		{
			name:   "nil config",
			config: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.config.Digest()
			if len(got) != 64 && tt.config != nil {
				t.Errorf("Digest() returned %d chars, want 64", len(got))
			}
			if tt.config == nil && got != "" {
				t.Errorf("Digest() on nil returned %q, want empty string", got)
			}
		})
	}
}

// TestPkcs7Config_Digest tests Digest method
func TestPkcs7Config_Digest(t *testing.T) {
	tests := []struct {
		name   string
		config *Pkcs7Config
	}{
		{
			name: "full config",
			config: &Pkcs7Config{
				Provider:         "alibaba_cloud",
				AlibabaCloudMode: "secure",
			},
		},
		{
			name:   "nil config",
			config: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.config.Digest()
			if len(got) != 64 && tt.config != nil {
				t.Errorf("Digest() returned %d chars, want 64", len(got))
			}
			if tt.config == nil && got != "" {
				t.Errorf("Digest() on nil returned %q, want empty string", got)
			}
		})
	}
}

// TestPrivateCaConfig_Digest tests Digest method
func TestPrivateCaConfig_Digest(t *testing.T) {
	tests := []struct {
		name   string
		config *PrivateCaConfig
	}{
		{
			name: "with certificate",
			config: &PrivateCaConfig{
				Certificate: "test-cert",
			},
		},
		{
			name:   "nil config",
			config: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.config.Digest()
			if len(got) != 64 && tt.config != nil {
				t.Errorf("Digest() returned %d chars, want 64", len(got))
			}
			if tt.config == nil && got != "" {
				t.Errorf("Digest() on nil returned %q, want empty string", got)
			}
		})
	}
}

// TestOidcTokenConfig_Digest tests Digest method
func TestOidcTokenConfig_Digest(t *testing.T) {
	tests := []struct {
		name   string
		config *OidcTokenConfig
	}{
		{
			name: "gcp provider",
			config: &OidcTokenConfig{
				Provider:            "gcp",
				GoogleVmIdentityUrl: "https://metadata.google.internal",
				GoogleVmIdentityAud: "test-audience",
			},
		},
		{
			name: "custom provider",
			config: &OidcTokenConfig{
				Provider:  "custom",
				OidcToken: "test-token",
			},
		},
		{
			name:   "nil config",
			config: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.config.Digest()
			if len(got) != 64 && tt.config != nil {
				t.Errorf("Digest() returned %d chars, want 64", len(got))
			}
			if tt.config == nil && got != "" {
				t.Errorf("Digest() on nil returned %q, want empty string", got)
			}
		})
	}
}

// TestExSignerConfig_Digest tests Digest method
func TestExSignerConfig_Digest(t *testing.T) {
	tests := []struct {
		name   string
		config *ExSignerConfig
	}{
		{
			name: "with pkcs11",
			config: &ExSignerConfig{
				KeyID:     "key-123",
				Algorithm: "RS256",
				Pkcs11: &ExSignerPkcs11Config{
					LibraryPath: "/usr/lib/libpkcs11.so",
					TokenLabel:  "token",
					KeyLabel:    "key",
				},
			},
		},
		{
			name: "with yubikey",
			config: &ExSignerConfig{
				Algorithm:  "ES256",
				YubikeyPiv: &ExSignerYubikeyPivConfig{Slot: "auth"},
			},
		},
		{
			name: "with external command",
			config: &ExSignerConfig{
				Algorithm: "RS256",
				ExternalCommand: &ExSignerExternalCommandConfig{
					Command:   "/usr/bin/sign",
					Parameter: "--sign",
				},
			},
		},
		{
			name: "with key file",
			config: &ExSignerConfig{
				Algorithm: "RS256",
				KeyFile: &ExSignerKeyFileConfig{
					File: "/path/to/key.pem",
				},
			},
		},
		{
			name:   "nil config",
			config: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.config.Digest()
			if len(got) != 64 && tt.config != nil {
				t.Errorf("Digest() returned %d chars, want 64", len(got))
			}
			if tt.config == nil && got != "" {
				t.Errorf("Digest() on nil returned %q, want empty string", got)
			}
		})
	}
}

// TestExSignerPkcs11Config_Digest tests Digest method
func TestExSignerPkcs11Config_Digest(t *testing.T) {
	tests := []struct {
		name   string
		config *ExSignerPkcs11Config
	}{
		{
			name: "full config",
			config: &ExSignerPkcs11Config{
				LibraryPath: "/usr/lib/libpkcs11.so",
				TokenLabel:  "token",
				KeyLabel:    "key",
				Pin:         "1234",
			},
		},
		{
			name:   "nil config",
			config: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.config.Digest()
			if len(got) != 64 && tt.config != nil {
				t.Errorf("Digest() returned %d chars, want 64", len(got))
			}
			if tt.config == nil && got != "" {
				t.Errorf("Digest() on nil returned %q, want empty string", got)
			}
		})
	}
}

// TestExSignerYubikeyPivConfig_Digest tests Digest method
func TestExSignerYubikeyPivConfig_Digest(t *testing.T) {
	tests := []struct {
		name   string
		config *ExSignerYubikeyPivConfig
	}{
		{
			name: "full config",
			config: &ExSignerYubikeyPivConfig{
				Slot:      "auth",
				Pin:       "1234",
				PinPolicy: "always",
			},
		},
		{
			name:   "nil config",
			config: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.config.Digest()
			if len(got) != 64 && tt.config != nil {
				t.Errorf("Digest() returned %d chars, want 64", len(got))
			}
			if tt.config == nil && got != "" {
				t.Errorf("Digest() on nil returned %q, want empty string", got)
			}
		})
	}
}

// TestExSignerExternalCommandConfig_Digest tests Digest method
func TestExSignerExternalCommandConfig_Digest(t *testing.T) {
	tests := []struct {
		name   string
		config *ExSignerExternalCommandConfig
	}{
		{
			name: "full config",
			config: &ExSignerExternalCommandConfig{
				Command:   "/usr/bin/sign",
				Parameter: "--sign",
			},
		},
		{
			name:   "nil config",
			config: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.config.Digest()
			if len(got) != 64 && tt.config != nil {
				t.Errorf("Digest() returned %d chars, want 64", len(got))
			}
			if tt.config == nil && got != "" {
				t.Errorf("Digest() on nil returned %q, want empty string", got)
			}
		})
	}
}

// TestExSignerKeyFileConfig_Digest tests Digest method
func TestExSignerKeyFileConfig_Digest(t *testing.T) {
	tests := []struct {
		name   string
		config *ExSignerKeyFileConfig
	}{
		{
			name: "with key",
			config: &ExSignerKeyFileConfig{
				Key: "test-key",
			},
		},
		{
			name: "with file",
			config: &ExSignerKeyFileConfig{
				File: "/path/to/key.pem",
			},
		},
		{
			name:   "nil config",
			config: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.config.Digest()
			if len(got) != 64 && tt.config != nil {
				t.Errorf("Digest() returned %d chars, want 64", len(got))
			}
			if tt.config == nil && got != "" {
				t.Errorf("Digest() on nil returned %q, want empty string", got)
			}
		})
	}
}

// TestDigestHelper tests the digest helper function
func TestDigestHelper(t *testing.T) {
	tests := []struct {
		name     string
		args     []string
		wantLen  int
	}{
		{
			name:    "empty args",
			args:    []string{},
			wantLen: 64,
		},
		{
			name:    "single arg",
			args:    []string{"test"},
			wantLen: 64,
		},
		{
			name:    "multiple args",
			args:    []string{"test1", "test2", "test3"},
			wantLen: 64,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := digest(tt.args...)
			if len(got) != tt.wantLen {
				t.Errorf("digest() returned %d chars, want %d", len(got), tt.wantLen)
			}
		})
	}
}

// TestIntToString tests intToString helper function
func TestIntToString(t *testing.T) {
	tests := []struct {
		name     string
		input    int
		expected string
	}{
		{
			name:     "zero",
			input:    0,
			expected: "",
		},
		{
			name:     "positive",
			input:    42,
			expected: "42",
		},
		{
			name:     "negative",
			input:    -10,
			expected: "-10",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := intToString(tt.input)
			if got != tt.expected {
				t.Errorf("intToString(%d) = %q, want %q", tt.input, got, tt.expected)
			}
		})
	}
}

// TestFileModTime tests fileModTime helper function
func TestFileModTime(t *testing.T) {
	tmpDir := t.TempDir()

	tests := []struct {
		name     string
		filename string
		setup    func() string
		wantHex  bool
	}{
		{
			name:     "empty filename",
			filename: "",
			wantHex:  false,
		},
		{
			name: "existing file",
			setup: func() string {
				file := filepath.Join(tmpDir, "test.txt")
				os.WriteFile(file, []byte("test"), 0644)
				return file
			},
			wantHex: true,
		},
		{
			name:     "non-existing file",
			filename: filepath.Join(tmpDir, "nonexistent.txt"),
			wantHex:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			filename := tt.filename
			if tt.setup != nil {
				filename = tt.setup()
			}
			got := fileModTime(filename)
			if tt.wantHex {
				if len(got) == 0 {
					t.Errorf("fileModTime(%q) = %q, want non-empty hex string", filename, got)
				}
			} else {
				if len(got) != 0 {
					t.Errorf("fileModTime(%q) = %q, want empty string", filename, got)
				}
			}
		})
	}
}
