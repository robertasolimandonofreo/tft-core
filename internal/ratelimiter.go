package internal

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

type RateLimiter struct {
	client *redis.Client
	prefix string
}

type RateLimit struct {
	requests int
	window   time.Duration
}

var riotRateLimits = []RateLimit{
	{requests: 20, window: 1 * time.Second},
	{requests: 100, window: 2 * time.Minute},
}

func NewRateLimiter(cfg *Config) *RateLimiter {
	client := redis.NewClient(&redis.Options{
		Addr:     fmt.Sprintf("%s:%s", cfg.RedisHost, cfg.RedisPort),
		Password: cfg.RedisPassword,
		DB:       cfg.RedisDB,
	})

	return &RateLimiter{
		client: client,
		prefix: cfg.RateLimitRedisPrefix,
	}
}

func (rl *RateLimiter) Allow(ctx context.Context, key string) (bool, error) {
	for _, limit := range riotRateLimits {
		allowed, err := rl.checkLimit(ctx, key, limit)
		if err != nil {
			return false, err
		}
		if !allowed {
			return false, nil
		}
	}
	return true, nil
}

func (rl *RateLimiter) checkLimit(ctx context.Context, key string, limit RateLimit) (bool, error) {
	redisKey := fmt.Sprintf("%s:%s:%d", rl.prefix, key, int(limit.window.Seconds()))
	
	count, err := rl.client.Incr(ctx, redisKey).Result()
	if err != nil {
		return false, err
	}

	if count == 1 {
		err = rl.client.Expire(ctx, redisKey, limit.window).Err()
		if err != nil {
			return false, err
		}
	}

	return int(count) <= limit.requests, nil
}