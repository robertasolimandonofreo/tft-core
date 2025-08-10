package internal

import (
	"os"
)

type Config struct {
	RiotAPIKey    string
	RiotRegion    string
	RiotBaseURL   string
	PostgresHost     string
	PostgresPort     string
	PostgresUser     string
	PostgresPassword string
	PostgresDb       string
	PostgresSSLMode  string
	RedisHost     string
	RedisPort     string
	RedisPassword string
	RedisDB       string
	NATSUrl       string
	NATSClusterID string
	NATSClientID  string
	RateLimitBuffer      string
	RateLimitRedisPrefix string
	AppPort  string
	AppEnv   string
	LogLevel string
	CacheEnabled bool
	DatabaseEnabled bool
}

func LoadConfig() *Config {
	cacheEnabled := os.Getenv("CACHE_ENABLED")
	enabled := cacheEnabled == "true" || cacheEnabled == ""
	
	dbEnabled := os.Getenv("DATABASE_ENABLED")
	dbEnabledBool := dbEnabled == "true" || dbEnabled == ""
	
	return &Config{
		RiotAPIKey:    os.Getenv("RIOT_API_KEY"),
		RiotRegion:    os.Getenv("RIOT_REGION"),
		RiotBaseURL:   os.Getenv("RIOT_BASE_URL"),

		PostgresHost:     os.Getenv("POSTGRES_HOST"),
		PostgresPort:     os.Getenv("POSTGRES_PORT"),
		PostgresUser:     os.Getenv("POSTGRES_USER"),
		PostgresPassword: os.Getenv("POSTGRES_PASSWORD"),
		PostgresDb: 	  os.Getenv("POSTGRES_DB"),
		PostgresSSLMode:  os.Getenv("POSTGRES_SSL_MODE"),

		RedisHost:     os.Getenv("REDIS_HOST"),
		RedisPort:     os.Getenv("REDIS_PORT"),
		RedisPassword: os.Getenv("REDIS_PASSWORD"),
		RedisDB:       os.Getenv("REDIS_DB"),

		NATSUrl:       os.Getenv("NATS_URL"),
		NATSClusterID: os.Getenv("NATS_CLUSTER_ID"),
		NATSClientID:  os.Getenv("NATS_CLIENT_ID"),

		RateLimitBuffer:      os.Getenv("RATE_LIMIT_BUFFER"),
		RateLimitRedisPrefix: os.Getenv("RATE_LIMIT_REDIS_PREFIX"),

		AppPort:  os.Getenv("APP_PORT"),
		AppEnv:   os.Getenv("APP_ENV"),
		LogLevel: os.Getenv("LOG_LEVEL"),
		CacheEnabled: enabled,
		DatabaseEnabled: dbEnabledBool,
	}
}