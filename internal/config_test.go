package internal

import (
	"os"
	"testing"
)

func TestLoadConfig_Success(t *testing.T) {
	// Set required environment variables
	os.Setenv("RIOT_API_KEY", "test-api-key")
	os.Setenv("RIOT_BASE_URL", "https://test.api.riot.com")
	os.Setenv("POSTGRES_USER", "test-user")
	os.Setenv("POSTGRES_PASSWORD", "test-pass")
	os.Setenv("POSTGRES_DB", "test-db")
	defer cleanupEnv()
	
	cfg, err := LoadConfig()
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}
	
	if cfg.RiotAPIKey != "test-api-key" {
		t.Errorf("expected RiotAPIKey 'test-api-key', got %s", cfg.RiotAPIKey)
	}
	
	if cfg.RiotRegion != "BR1" {
		t.Errorf("expected default RiotRegion 'BR1', got %s", cfg.RiotRegion)
	}
	
	if cfg.PostgresHost != "localhost" {
		t.Errorf("expected default PostgresHost 'localhost', got %s", cfg.PostgresHost)
	}
	
	if cfg.PostgresPort != "5432" {
		t.Errorf("expected default PostgresPort '5432', got %s", cfg.PostgresPort)
	}
	
	if cfg.RedisDB != 0 {
		t.Errorf("expected default RedisDB 0, got %d", cfg.RedisDB)
	}
	
	if !cfg.CacheEnabled {
		t.Error("expected CacheEnabled to be true by default")
	}
	
	if !cfg.DatabaseEnabled {
		t.Error("expected DatabaseEnabled to be true by default")
	}
}

func TestLoadConfig_CustomValues(t *testing.T) {
	os.Setenv("RIOT_API_KEY", "custom-key")
	os.Setenv("RIOT_BASE_URL", "https://custom.api.riot.com")
	os.Setenv("RIOT_REGION", "NA1")
	os.Setenv("POSTGRES_HOST", "custom-host")
	os.Setenv("POSTGRES_PORT", "5433")
	os.Setenv("POSTGRES_USER", "custom-user")
	os.Setenv("POSTGRES_PASSWORD", "custom-pass")
	os.Setenv("POSTGRES_DB", "custom-db")
	os.Setenv("POSTGRES_SSL_MODE", "require")
	os.Setenv("REDIS_HOST", "redis-host")
	os.Setenv("REDIS_PORT", "6380")
	os.Setenv("REDIS_PASSWORD", "redis-pass")
	os.Setenv("REDIS_DB", "5")
	os.Setenv("NATS_URL", "nats://custom:4223")
	os.Setenv("APP_PORT", "8080")
	os.Setenv("APP_ENV", "production")
	os.Setenv("LOG_LEVEL", "debug")
	os.Setenv("CACHE_ENABLED", "false")
	os.Setenv("DATABASE_ENABLED", "false")
	defer cleanupEnv()
	
	cfg, err := LoadConfig()
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}
	
	if cfg.RiotRegion != "NA1" {
		t.Errorf("expected RiotRegion 'NA1', got %s", cfg.RiotRegion)
	}
	
	if cfg.PostgresHost != "custom-host" {
		t.Errorf("expected PostgresHost 'custom-host', got %s", cfg.PostgresHost)
	}
	
	if cfg.PostgresPort != "5433" {
		t.Errorf("expected PostgresPort '5433', got %s", cfg.PostgresPort)
	}
	
	if cfg.PostgresSSLMode != "require" {
		t.Errorf("expected PostgresSSLMode 'require', got %s", cfg.PostgresSSLMode)
	}
	
	if cfg.RedisHost != "redis-host" {
		t.Errorf("expected RedisHost 'redis-host', got %s", cfg.RedisHost)
	}
	
	if cfg.RedisPort != "6380" {
		t.Errorf("expected RedisPort '6380', got %s", cfg.RedisPort)
	}
	
	if cfg.RedisDB != 5 {
		t.Errorf("expected RedisDB 5, got %d", cfg.RedisDB)
	}
	
	if cfg.NATSUrl != "nats://custom:4223" {
		t.Errorf("expected NATSUrl 'nats://custom:4223', got %s", cfg.NATSUrl)
	}
	
	if cfg.AppPort != "8080" {
		t.Errorf("expected AppPort '8080', got %s", cfg.AppPort)
	}
	
	if cfg.AppEnv != "production" {
		t.Errorf("expected AppEnv 'production', got %s", cfg.AppEnv)
	}
	
	if cfg.LogLevel != "debug" {
		t.Errorf("expected LogLevel 'debug', got %s", cfg.LogLevel)
	}
	
	if cfg.CacheEnabled {
		t.Error("expected CacheEnabled to be false")
	}
	
	if cfg.DatabaseEnabled {
		t.Error("expected DatabaseEnabled to be false")
	}
}

func TestLoadConfig_MissingRiotAPIKey(t *testing.T) {
	os.Setenv("RIOT_BASE_URL", "https://test.api.riot.com")
	defer cleanupEnv()
	
	_, err := LoadConfig()
	if err == nil {
		t.Error("expected error for missing RIOT_API_KEY")
	}
	
	if err.Error() != "RIOT_API_KEY is required" {
		t.Errorf("unexpected error message: %v", err)
	}
}

func TestLoadConfig_MissingRiotBaseURL(t *testing.T) {
	os.Setenv("RIOT_API_KEY", "test-key")
	defer cleanupEnv()
	
	_, err := LoadConfig()
	if err == nil {
		t.Error("expected error for missing RIOT_BASE_URL")
	}
	
	if err.Error() != "RIOT_BASE_URL is required" {
		t.Errorf("unexpected error message: %v", err)
	}
}

func TestLoadConfig_MissingDatabaseConfig(t *testing.T) {
	os.Setenv("RIOT_API_KEY", "test-key")
	os.Setenv("RIOT_BASE_URL", "https://test.api.riot.com")
	os.Setenv("DATABASE_ENABLED", "true")
	defer cleanupEnv()
	
	_, err := LoadConfig()
	if err == nil {
		t.Error("expected error for missing database config")
	}
	
	if err.Error() != "POSTGRES_USER is required when database is enabled" {
		t.Errorf("unexpected error message: %v", err)
	}
}

func TestLoadConfig_InvalidRedisDB(t *testing.T) {
	os.Setenv("RIOT_API_KEY", "test-key")
	os.Setenv("RIOT_BASE_URL", "https://test.api.riot.com")
	os.Setenv("REDIS_DB", "invalid")
	defer cleanupEnv()
	
	_, err := LoadConfig()
	if err == nil {
		t.Error("expected error for invalid REDIS_DB")
	}
	
	if err.Error() != "invalid REDIS_DB value" {
		t.Errorf("unexpected error message: %v", err)
	}
}

func TestGetEnvDefault(t *testing.T) {
	// Test with existing environment variable
	os.Setenv("TEST_VAR", "test-value")
	result := getEnvDefault("TEST_VAR", "default")
	if result != "test-value" {
		t.Errorf("expected 'test-value', got %s", result)
	}
	
	// Test with non-existing environment variable
	result = getEnvDefault("NON_EXISTING_VAR", "default")
	if result != "default" {
		t.Errorf("expected 'default', got %s", result)
	}
	
	// Clean up
	os.Unsetenv("TEST_VAR")
}

func TestGetBoolEnvDefault(t *testing.T) {
	tests := []struct {
		envValue string
		defaultVal bool
		expected bool
	}{
		{"true", false, true},
		{"false", true, false},
		{"", true, true},
		{"", false, false},
		{"invalid", true, true},
		{"invalid", false, false},
	}
	
	for _, tt := range tests {
		if tt.envValue != "" {
			os.Setenv("TEST_BOOL_VAR", tt.envValue)
		} else {
			os.Unsetenv("TEST_BOOL_VAR")
		}
		
		result := getBoolEnvDefault("TEST_BOOL_VAR", tt.defaultVal)
		if result != tt.expected {
			t.Errorf("getBoolEnvDefault(%s, %v): expected %v, got %v", 
				tt.envValue, tt.defaultVal, tt.expected, result)
		}
	}
	
	// Clean up
	os.Unsetenv("TEST_BOOL_VAR")
}

func TestConfig_Validate_DatabaseDisabled(t *testing.T) {
	cfg := &Config{
		RiotAPIKey:      "test-key",
		RiotBaseURL:     "https://test.api.riot.com",
		DatabaseEnabled: false,
	}
	
	err := cfg.validate()
	if err != nil {
		t.Errorf("expected no error when database is disabled, got %v", err)
	}
}

func TestConfig_Validate_DatabaseEnabledComplete(t *testing.T) {
	cfg := &Config{
		RiotAPIKey:       "test-key",
		RiotBaseURL:      "https://test.api.riot.com",
		DatabaseEnabled:  true,
		PostgresUser:     "user",
		PostgresPassword: "pass",
		PostgresDB:       "db",
	}
	
	err := cfg.validate()
	if err != nil {
		t.Errorf("expected no error with complete database config, got %v", err)
	}
}

func cleanupEnv() {
	envVars := []string{
		"RIOT_API_KEY", "RIOT_BASE_URL", "RIOT_REGION",
		"POSTGRES_HOST", "POSTGRES_PORT", "POSTGRES_USER", 
		"POSTGRES_PASSWORD", "POSTGRES_DB", "POSTGRES_SSL_MODE",
		"REDIS_HOST", "REDIS_PORT", "REDIS_PASSWORD", "REDIS_DB",
		"NATS_URL", "APP_PORT", "APP_ENV", "LOG_LEVEL",
		"CACHE_ENABLED", "DATABASE_ENABLED",
	}
	
	for _, env := range envVars {
		os.Unsetenv(env)
	}
}