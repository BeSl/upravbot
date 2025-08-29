package config

import (
	"os"
	"reflect"
	"testing"
)

func TestLoad(t *testing.T) {
	tests := []struct {
		name           string
		configContent  string
		envVars        map[string]string
		expectedConfig *Config
		expectError    bool
	}{
		{
			name: "Valid YAML config",
			configContent: `bot:
  token: "test_token"
  debug: true
database:
  path: "test.db"
users:
  admin_user_ids: [123456789]
  allowed_users: [987654321]`,
			expectedConfig: &Config{
				Bot: BotConfig{
					Token: "test_token",
					Debug: true,
				},
				Database: DatabaseConfig{
					Path: "test.db",
				},
				Users: UsersConfig{
					AdminUserIDs: []int64{123456789},
					AllowedUsers: []int64{987654321},
				},
				FileManager: FileManagerConfig{
					AllowedDrives:  []string{"C:", "D:"},
					MaxFileSize:    10485760,
					AllowedActions: []string{"list", "download"},
					DownloadPath:   "./downloads",
					UploadPath:     "./uploads",
				},
				Screenshot: ScreenshotConfig{
					Enabled:     false,
					Quality:     80,
					Format:      "png",
					MaxWidth:    1920,
					MaxHeight:   1080,
					StoragePath: "./screenshots",
				},
				Events: EventsConfig{
					Enabled:         false,
					NotifyUsers:     []int64{},
					WatchEvents:     []string{"login", "logout", "error"},
					PollingInterval: 30,
				},
			},
			expectError: false,
		},
		{
			name:          "Missing config file with environment variables",
			configContent: "",
			envVars: map[string]string{
				"BOT_TOKEN":        "env_token",
				"BOT_DEBUG":        "true",
				"DB_PATH":          "env.db",
				"ADMIN_USER_IDS":   "111,222",
				"ALLOWED_USER_IDS": "333,444",
			},
			expectedConfig: &Config{
				Bot: BotConfig{
					Token: "env_token",
					Debug: true,
				},
				Database: DatabaseConfig{
					Path: "env.db",
				},
				Users: UsersConfig{
					AdminUserIDs: []int64{111, 222},
					AllowedUsers: []int64{333, 444},
				},
				FileManager: FileManagerConfig{
					AllowedDrives:  []string{"C:", "D:"},
					MaxFileSize:    10485760,
					AllowedActions: []string{"list", "download"},
					DownloadPath:   "./downloads",
					UploadPath:     "./uploads",
				},
				Screenshot: ScreenshotConfig{
					Enabled:     false,
					Quality:     80,
					Format:      "png",
					MaxWidth:    1920,
					MaxHeight:   1080,
					StoragePath: "./screenshots",
				},
				Events: EventsConfig{
					Enabled:         false,
					NotifyUsers:     []int64{},
					WatchEvents:     []string{"login", "logout", "error"},
					PollingInterval: 30,
				},
			},
			expectError: false,
		},
		{
			name: "Config with environment override",
			configContent: `bot:
  token: "config_token"
  debug: false
database:
  path: "config.db"`,
			envVars: map[string]string{
				"BOT_TOKEN": "env_override_token",
				"DB_PATH":   "env_override.db",
			},
			expectedConfig: &Config{
				Bot: BotConfig{
					Token: "env_override_token",
					Debug: false,
				},
				Database: DatabaseConfig{
					Path: "env_override.db",
				},
				Users: UsersConfig{
					AdminUserIDs: []int64{},
					AllowedUsers: []int64{},
				},
				FileManager: FileManagerConfig{
					AllowedDrives:  []string{"C:", "D:"},
					MaxFileSize:    10485760,
					AllowedActions: []string{"list", "download"},
					DownloadPath:   "./downloads",
					UploadPath:     "./uploads",
				},
				Screenshot: ScreenshotConfig{
					Enabled:     false,
					Quality:     80,
					Format:      "png",
					MaxWidth:    1920,
					MaxHeight:   1080,
					StoragePath: "./screenshots",
				},
				Events: EventsConfig{
					Enabled:         false,
					NotifyUsers:     []int64{},
					WatchEvents:     []string{"login", "logout", "error"},
					PollingInterval: 30,
				},
			},
			expectError: false,
		},
		{
			name:          "Default config when no file or env vars",
			configContent: "",
			expectedConfig: &Config{
				Bot: BotConfig{
					Token: "",
					Debug: false,
				},
				Database: DatabaseConfig{
					Path: "cupbot.db",
				},
				Users: UsersConfig{
					AdminUserIDs: []int64{},
					AllowedUsers: []int64{},
				},
				FileManager: FileManagerConfig{
					AllowedDrives:  []string{"C:", "D:"},
					MaxFileSize:    10485760,
					AllowedActions: []string{"list", "download"},
					DownloadPath:   "./downloads",
					UploadPath:     "./uploads",
				},
				Screenshot: ScreenshotConfig{
					Enabled:     false,
					Quality:     80,
					Format:      "png",
					MaxWidth:    1920,
					MaxHeight:   1080,
					StoragePath: "./screenshots",
				},
				Events: EventsConfig{
					Enabled:         false,
					NotifyUsers:     []int64{},
					WatchEvents:     []string{"login", "logout", "error"},
					PollingInterval: 30,
				},
			},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup environment variables
			originalEnv := make(map[string]string)
			for key, value := range tt.envVars {
				originalEnv[key] = os.Getenv(key)
				os.Setenv(key, value)
			}
			defer func() {
				for key, originalValue := range originalEnv {
					if originalValue == "" {
						os.Unsetenv(key)
					} else {
						os.Setenv(key, originalValue)
					}
				}
			}()

			// Create temporary config file if content provided
			var configPath string
			if tt.configContent != "" {
				tmpFile, err := os.CreateTemp("", "config_test_*.yaml")
				if err != nil {
					t.Fatal(err)
				}
				defer os.Remove(tmpFile.Name())

				_, err = tmpFile.WriteString(tt.configContent)
				if err != nil {
					t.Fatal(err)
				}
				tmpFile.Close()
				configPath = tmpFile.Name()
			} else {
				configPath = "nonexistent_config.yaml"
			}

			// Test Load function
			config, err := Load(configPath)

			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error but got none")
				}
				return
			}

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}

			if !reflect.DeepEqual(config, tt.expectedConfig) {
				t.Errorf("Config mismatch.\nExpected: %+v\nGot: %+v", tt.expectedConfig, config)
			}
		})
	}
}

func TestParseUserIDs(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []int64
	}{
		{
			name:     "Single ID",
			input:    "123456789",
			expected: []int64{123456789},
		},
		{
			name:     "Multiple IDs",
			input:    "111,222,333",
			expected: []int64{111, 222, 333},
		},
		{
			name:     "IDs with spaces",
			input:    "111, 222 , 333 ",
			expected: []int64{111, 222, 333},
		},
		{
			name:     "Empty string",
			input:    "",
			expected: make([]int64, 0),
		},
		{
			name:     "Invalid ID mixed with valid",
			input:    "111,invalid,333",
			expected: []int64{111, 333},
		},
		{
			name:     "Only invalid IDs",
			input:    "invalid,abc,xyz",
			expected: make([]int64, 0),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parseUserIDs(tt.input)
			if !reflect.DeepEqual(result, tt.expected) {
				t.Errorf("parseUserIDs(%q) = %v, want %v", tt.input, result, tt.expected)
			}
		})
	}
}

func TestConfigIsAdmin(t *testing.T) {
	config := &Config{
		Users: UsersConfig{
			AdminUserIDs: []int64{123456789, 987654321},
			AllowedUsers: []int64{111111111},
		},
	}

	tests := []struct {
		name     string
		userID   int64
		expected bool
	}{
		{
			name:     "Admin user",
			userID:   123456789,
			expected: true,
		},
		{
			name:     "Another admin user",
			userID:   987654321,
			expected: true,
		},
		{
			name:     "Non-admin user",
			userID:   111111111,
			expected: false,
		},
		{
			name:     "Unknown user",
			userID:   999999999,
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := config.IsAdmin(tt.userID)
			if result != tt.expected {
				t.Errorf("IsAdmin(%d) = %v, want %v", tt.userID, result, tt.expected)
			}
		})
	}
}

func TestConfigIsAllowed(t *testing.T) {
	config := &Config{
		Users: UsersConfig{
			AdminUserIDs: []int64{123456789},
			AllowedUsers: []int64{111111111, 222222222},
		},
	}

	tests := []struct {
		name     string
		userID   int64
		expected bool
	}{
		{
			name:     "Admin user (always allowed)",
			userID:   123456789,
			expected: true,
		},
		{
			name:     "Allowed user",
			userID:   111111111,
			expected: true,
		},
		{
			name:     "Another allowed user",
			userID:   222222222,
			expected: true,
		},
		{
			name:     "Unknown user",
			userID:   999999999,
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := config.IsAllowed(tt.userID)
			if result != tt.expected {
				t.Errorf("IsAllowed(%d) = %v, want %v", tt.userID, result, tt.expected)
			}
		})
	}
}

func TestConfigIsAllowedEmptyList(t *testing.T) {
	// Test case where allowed_users is empty - only admins should be allowed
	config := &Config{
		Users: UsersConfig{
			AdminUserIDs: []int64{123456789},
			AllowedUsers: []int64{}, // Empty list
		},
	}

	tests := []struct {
		name     string
		userID   int64
		expected bool
	}{
		{
			name:     "Admin user (allowed even with empty list)",
			userID:   123456789,
			expected: true,
		},
		{
			name:     "Non-admin user (not allowed with empty list)",
			userID:   111111111,
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := config.IsAllowed(tt.userID)
			if result != tt.expected {
				t.Errorf("IsAllowed(%d) with empty allowed list = %v, want %v", tt.userID, result, tt.expected)
			}
		})
	}
}

func TestLoadInvalidYAML(t *testing.T) {
	// Test invalid YAML content
	invalidYAML := `bot:
  token: "test_token"
  debug: true
invalid_yaml_syntax: [unclosed_bracket`

	tmpFile, err := os.CreateTemp("", "invalid_config_*.yaml")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tmpFile.Name())

	_, err = tmpFile.WriteString(invalidYAML)
	if err != nil {
		t.Fatal(err)
	}
	tmpFile.Close()

	_, err = Load(tmpFile.Name())
	if err == nil {
		t.Error("Expected error for invalid YAML, but got none")
	}
}
