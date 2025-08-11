package internal

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

type CacheManager struct {
	redis    *redis.Client
	database *DatabaseManager
	enabled  bool
}

func NewCacheManager(cfg *Config, db *DatabaseManager) *CacheManager {
	var redisClient *redis.Client
	if cfg.CacheEnabled {
		redisClient = redis.NewClient(&redis.Options{
			Addr:     fmt.Sprintf("%s:%s", cfg.RedisHost, cfg.RedisPort),
			Password: cfg.RedisPassword,
			DB:       cfg.RedisDB,
		})
	}

	return &CacheManager{
		redis:    redisClient,
		database: db,
		enabled:  cfg.CacheEnabled,
	}
}

func (cm *CacheManager) Get(ctx context.Context, key string, result interface{}) error {
	if !cm.enabled {
		return redis.Nil
	}

	data, err := cm.redis.Get(ctx, key).Result()
	if err != nil {
		return err
	}

	return json.Unmarshal([]byte(data), result)
}

func (cm *CacheManager) Set(ctx context.Context, key string, data interface{}, ttl time.Duration) error {
	if !cm.enabled {
		return nil
	}

	jsonData, err := json.Marshal(data)
	if err != nil {
		return err
	}

	return cm.redis.Set(ctx, key, jsonData, ttl).Err()
}

func (cm *CacheManager) Key(parts ...string) string {
	key := "tft"
	for _, part := range parts {
		key = fmt.Sprintf("%s:%s", key, part)
	}
	return key
}

func (cm *CacheManager) GetSummonerName(ctx context.Context, puuid string) (string, error) {
	// Try Redis first
	if cm.enabled && cm.redis != nil {
		key := cm.Key("summoner_name", puuid)
		name, err := cm.redis.Get(ctx, key).Result()
		if err == nil && name != "" && name != "Loading..." {
			return name, nil
		}
	}

	// Try PostgreSQL if Redis fails or has Loading...
	if cm.database != nil && cm.database.Enabled {
		name, err := cm.database.GetSummonerName(puuid)
		if err == nil && name != "" {
			// Cache the result in Redis for next time
			if cm.enabled && cm.redis != nil {
				key := cm.Key("summoner_name", puuid)
				cm.redis.Set(ctx, key, name, 24*time.Hour)
			}
			return name, nil
		}
	}

	return "", redis.Nil
}

func (cm *CacheManager) SetSummonerName(ctx context.Context, puuid, name string) error {
	// Save to Redis
	if cm.enabled && cm.redis != nil {
		key := cm.Key("summoner_name", puuid)
		cm.redis.Set(ctx, key, name, 24*time.Hour)
	}

	// Save to PostgreSQL
	if cm.database != nil && cm.database.Enabled {
		gameName, tagLine := parseName(name)
		return cm.database.SetSummonerName(puuid, gameName, tagLine, "", "BR1")
	}

	return nil
}

func parseName(fullName string) (gameName, tagLine string) {
	parts := splitName(fullName)
	if len(parts) == 2 {
		return parts[0], parts[1]
	}
	return fullName, "BR1"
}

func splitName(name string) []string {
	for i := len(name) - 1; i >= 0; i-- {
		if name[i] == '#' {
			return []string{name[:i], name[i+1:]}
		}
	}
	return []string{name}
}
