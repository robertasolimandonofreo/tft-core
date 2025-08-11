package internal

import (
	"errors"
	"os"
	"strconv"
)

type Config struct {
	RiotAPIKey  string
	RiotRegion  string
	RiotBaseURL string

	PostgresHost     string
	PostgresPort     string
	PostgresUser     string
	PostgresPassword string
	PostgresDB       string
	PostgresSSLMode  string

	RedisHost     string
	RedisPort     string
	RedisPassword string
	RedisDB       int

	NATSUrl       string
	NATSClusterID string
	NATSClientID  string

	RateLimitRedisPrefix string

	AppPort  string
	AppEnv   string
	LogLevel string

	CacheEnabled    bool
	DatabaseEnabled bool
}

func LoadConfig() (*Config, error) {
	redisDB, err := strconv.Atoi(getEnvDefault("REDIS_DB", "0"))
	if err != nil {
		return nil, errors.New("invalid REDIS_DB value")
	}

	cfg := &Config{
		RiotAPIKey:  os.Getenv("RIOT_API_KEY"),
		RiotRegion:  getEnvDefault("RIOT_REGION", "BR1"),
		RiotBaseURL: os.Getenv("RIOT_BASE_URL"),

		PostgresHost:     getEnvDefault("POSTGRES_HOST", "localhost"),
		PostgresPort:     getEnvDefault("POSTGRES_PORT", "5432"),
		PostgresUser:     os.Getenv("POSTGRES_USER"),
		PostgresPassword: os.Getenv("POSTGRES_PASSWORD"),
		PostgresDB:       os.Getenv("POSTGRES_DB"),
		PostgresSSLMode:  getEnvDefault("POSTGRES_SSL_MODE", "disable"),

		RedisHost:     getEnvDefault("REDIS_HOST", "localhost"),
		RedisPort:     getEnvDefault("REDIS_PORT", "6379"),
		RedisPassword: os.Getenv("REDIS_PASSWORD"),
		RedisDB:       redisDB,

		NATSUrl:       getEnvDefault("NATS_URL", "nats://localhost:4222"),
		NATSClusterID: getEnvDefault("NATS_CLUSTER_ID", "tft-cluster"),
		NATSClientID:  getEnvDefault("NATS_CLIENT_ID", "tft-service"),

		RateLimitRedisPrefix: getEnvDefault("RATE_LIMIT_REDIS_PREFIX", "tft:ratelimit"),

		AppPort:  getEnvDefault("APP_PORT", "8000"),
		AppEnv:   getEnvDefault("APP_ENV", "development"),
		LogLevel: getEnvDefault("LOG_LEVEL", "info"),

		CacheEnabled:    getBoolEnvDefault("CACHE_ENABLED", true),
		DatabaseEnabled: getBoolEnvDefault("DATABASE_ENABLED", true),
	}

	return cfg, cfg.validate()
}

func (c *Config) validate() error {
	if c.RiotAPIKey == "" {
		return errors.New("RIOT_API_KEY is required")
	}
	if c.RiotBaseURL == "" {
		return errors.New("RIOT_BASE_URL is required")
	}
	if c.DatabaseEnabled {
		if c.PostgresUser == "" {
			return errors.New("POSTGRES_USER is required when database is enabled")
		}
		if c.PostgresPassword == "" {
			return errors.New("POSTGRES_PASSWORD is required when database is enabled")
		}
		if c.PostgresDB == "" {
			return errors.New("POSTGRES_DATABASE is required when database is enabled")
		}
	}
	return nil
}

func getEnvDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getBoolEnvDefault(key string, defaultValue bool) bool {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}
	return value == "true"
}
