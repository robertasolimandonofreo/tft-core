package internal

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"time"

	"github.com/redis/go-redis/v9"
)

type CacheManager struct {
	RedisClient *redis.Client
	Enabled     bool
}

func NewCacheManager(cfg *Config) *CacheManager {
	redisDB, _ := strconv.Atoi(cfg.RedisDB)
	client := redis.NewClient(&redis.Options{
		Addr:     fmt.Sprintf("%s:%s", cfg.RedisHost, cfg.RedisPort),
		Password: cfg.RedisPassword,
		DB:       redisDB,
	})
	
	return &CacheManager{
		RedisClient: client,
		Enabled:     cfg.CacheEnabled,
	}
}

func (cm *CacheManager) GetCachedData(ctx context.Context, key string, result interface{}) error {
	if !cm.Enabled {
		return redis.Nil
	}
	
	data, err := cm.RedisClient.Get(ctx, key).Result()
	if err != nil {
		return err
	}
	
	return json.Unmarshal([]byte(data), result)
}

func (cm *CacheManager) SetCachedData(ctx context.Context, key string, data interface{}, ttl time.Duration) error {
	if !cm.Enabled {
		return nil
	}
	
	jsonData, err := json.Marshal(data)
	if err != nil {
		return err
	}
	
	return cm.RedisClient.Set(ctx, key, jsonData, ttl).Err()
}

func (cm *CacheManager) GenerateKey(prefix, region string, params ...string) string {
	key := fmt.Sprintf("tft:%s:%s", prefix, region)
	for _, param := range params {
		key = fmt.Sprintf("%s:%s", key, param)
	}
	return key
}

func (cm *CacheManager) DeletePattern(ctx context.Context, pattern string) error {
	if !cm.Enabled {
		return nil
	}
	
	keys, err := cm.RedisClient.Keys(ctx, pattern).Result()
	if err != nil {
		return err
	}
	
	if len(keys) > 0 {
		return cm.RedisClient.Del(ctx, keys...).Err()
	}
	
	return nil
}