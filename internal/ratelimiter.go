package internal

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/redis/go-redis/v9"
)

type RateLimiter struct {
	RedisClient *redis.Client
	Prefix      string
	Limit       int
	Window      time.Duration
}

func NewRateLimiter(cfg *Config, limit int, window time.Duration) *RateLimiter {
	redisDB, _ := strconv.Atoi(cfg.RedisDB)
	client := redis.NewClient(&redis.Options{
		Addr:     fmt.Sprintf("%s:%s", cfg.RedisHost, cfg.RedisPort),
		Password: cfg.RedisPassword,
		DB:       redisDB,
	})
	return &RateLimiter{
		RedisClient: client,
		Prefix:      cfg.RateLimitRedisPrefix,
		Limit:       limit,
		Window:      window,
	}
}

func (rl *RateLimiter) Allow(ctx context.Context, key string) (bool, error) {
	redisKey := fmt.Sprintf("%s:%s", rl.Prefix, key)
	count, err := rl.RedisClient.Incr(ctx, redisKey).Result()
	if err != nil {
		return false, err
	}
	if count == 1 {
		err = rl.RedisClient.Expire(ctx, redisKey, rl.Window).Err()
		if err != nil {
			return false, err
		}
	}
	if int(count) > rl.Limit {
		return false, nil
	}
	return true, nil
}