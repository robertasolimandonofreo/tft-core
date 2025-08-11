package internal

import (
	"os"
	"testing"
)

func TestGetEnvDefault(t *testing.T) {
	tests := []struct {
		name         string
		key          string
		defaultValue string
		envValue     string
		expected     string
	}{
		{
			name:         "returns environment value when set",
			key:          "TEST_KEY_1",
			defaultValue: "default",
			envValue:     "env_value",
			expected:     "env_value",
		},
		{
			name:         "returns default when env not set",
			key:          "TEST_KEY_2",
			defaultValue: "default_value",
			envValue:     "",
			expected:     "default_value",
		},
		{
			name:         "returns empty string when both empty",
			key:          "TEST_KEY_3",
			defaultValue: "",
			envValue:     "",
			expected:     "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.envValue != "" {
				os.Setenv(tt.key, tt.envValue)
				defer os.Unsetenv(tt.key)
			}

			result := getEnvDefault(tt.key, tt.defaultValue)
			if result != tt.expected {
				t.Errorf("getEnvDefault() = %v, expected %v", result, tt.expected)
			}
		})
	}
}

func TestGetBoolEnvDefault(t *testing.T) {
	tests := []struct {
		name         string
		key          string
		defaultValue bool
		envValue     string
		expected     bool
	}{
		{
			name:         "returns true when env is 'true'",
			key:          "BOOL_TEST_1",
			defaultValue: false,
			envValue:     "true",
			expected:     true,
		},
		{
			name:         "returns false when env is 'false'",
			key:          "BOOL_TEST_2",
			defaultValue: true,
			envValue:     "false",
			expected:     false,
		},
		{
			name:         "returns default when env is empty",
			key:          "BOOL_TEST_3",
			defaultValue: true,
			envValue:     "",
			expected:     true,
		},
		{
			name:         "returns false when env is not 'true'",
			key:          "BOOL_TEST_4",
			defaultValue: true,
			envValue:     "invalid",
			expected:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.envValue != "" {
				os.Setenv(tt.key, tt.envValue)
				defer os.Unsetenv(tt.key)
			} else {
				os.Unsetenv(tt.key)
			}

			result := getBoolEnvDefault(tt.key, tt.defaultValue)
			if result != tt.expected {
				t.Errorf("getBoolEnvDefault() = %v, expected %v", result, tt.expected)
			}
		})
	}
}

func TestConfig_Validate(t *testing.T) {
	tests := []struct {
		name      string
		config    Config
		expectErr bool
	}{
		{
			name: "valid config with required fields",
			config: Config{
				RiotAPIKey:      "test-key",
				RiotBaseURL:     "https://test.api.com",
				DatabaseEnabled: false,
			},
			expectErr: false,
		},
		{
			name: "missing riot api key",
			config: Config{
				RiotAPIKey:      "",
				RiotBaseURL:     "https://test.api.com",
				DatabaseEnabled: false,
			},
			expectErr: true,
		},
		{
			name: "missing riot base url",
			config: Config{
				RiotAPIKey:      "test-key",
				RiotBaseURL:     "",
				DatabaseEnabled: false,
			},
			expectErr: true,
		},
		{
			name: "database enabled but missing postgres user",
			config: Config{
				RiotAPIKey:       "test-key",
				RiotBaseURL:      "https://test.api.com",
				DatabaseEnabled:  true,
				PostgresUser:     "",
				PostgresPassword: "pass",
				PostgresDB:       "db",
			},
			expectErr: true,
		},
		{
			name: "database enabled with all postgres fields",
			config: Config{
				RiotAPIKey:       "test-key",
				RiotBaseURL:      "https://test.api.com",
				DatabaseEnabled:  true,
				PostgresUser:     "user",
				PostgresPassword: "pass",
				PostgresDB:       "db",
			},
			expectErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.validate()
			if tt.expectErr && err == nil {
				t.Error("expected error but got nil")
			}
			if !tt.expectErr && err != nil {
				t.Errorf("expected no error but got: %v", err)
			}
		})
	}
}
