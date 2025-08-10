package internal

import (
	"context"
	"testing"
	"time"

	"github.com/redis/go-redis/v9"
)

type mockRedisForRateLimit struct {
	counters map[string]int64
	ttls     map[string]time.Duration
}

func (m *mockRedisForRateLimit) Incr(ctx context.Context, key string) *redis.IntCmd {
	cmd := redis.NewIntCmd(ctx)
	if m.counters == nil {
		m.counters = make(map[string]int64)
	}
	m.counters[key]++
	cmd.SetVal(m.counters[key])
	return cmd
}

func (m *mockRedisForRateLimit) Expire(ctx context.Context, key string, expiration time.Duration) *redis.BoolCmd {
	cmd := redis.NewBoolCmd(ctx)
	if m.ttls == nil {
		m.ttls = make(map[string]time.Duration)
	}
	m.ttls[key] = expiration
	cmd.SetVal(true)
	return cmd
}

func TestRateLimiter_Allow_FirstRequest(t *testing.T) {
	cfg := &Config{
		RedisHost:            "localhost",
		RedisPort:            "6379",
		RateLimitRedisPrefix: "test",
	}
	
	logger := createTestLogger()
	rateLimiter := NewRateLimiter(cfg, logger)
	
	mockRedis := &mockRedisForRateLimit{}
	rateLimiter.client = mockRedis
	
	ctx := context.Background()
	allowed, err := rateLimiter.Allow(ctx, "test-key")
	
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}
	
	if !allowed {
		t.Error("first request should be allowed")
	}
	
	if mockRedis.counters["test:test-key:1"] != 1 {
		t.Errorf("expected counter 1, got %d", mockRedis.counters["test:test-key:1"])
	}
	
	if mockRedis.ttls["test:test-key:1"] != 1*time.Second {
		t.Errorf("expected TTL 1s, got %v", mockRedis.ttls["test:test-key:1"])
	}
}

func TestRateLimiter_Allow_WithinLimit(t *testing.T) {
	cfg := &Config{
		RedisHost:            "localhost",
		RedisPort:            "6379",
		RateLimitRedisPrefix: "test",
	}
	
	logger := createTestLogger()
	rateLimiter := NewRateLimiter(cfg, logger)
	
	mockRedis := &mockRedisForRateLimit{
		counters: map[string]int64{
			"test:test-key:1":   10,
			"test:test-key:120": 50,
		},
	}
	rateLimiter.client = mockRedis
	
	ctx := context.Background()
	allowed, err := rateLimiter.Allow(ctx, "test-key")
	
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}
	
	if !allowed {
		t.Error("request within limit should be allowed")
	}
}

func TestRateLimiter_Allow_ExceedsLimit(t *testing.T) {
	cfg := &Config{
		RedisHost:            "localhost",
		RedisPort:            "6379",
		RateLimitRedisPrefix: "test",
	}
	
	logger := createTestLogger()
	rateLimiter := NewRateLimiter(cfg, logger)
	
	mockRedis := &mockRedisForRateLimit{
		counters: map[string]int64{
			"test:test-key:1":   25,
			"test:test-key:120": 50,
		},
	}
	rateLimiter.client = mockRedis
	
	ctx := context.Background()
	allowed, err := rateLimiter.Allow(ctx, "test-key")
	
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}
	
	if allowed {
		t.Error("request exceeding limit should not be allowed")
	}
}

func TestRateLimiter_CheckLimit(t *testing.T) {
	cfg := &Config{
		RedisHost:            "localhost",
		RedisPort:            "6379",
		RateLimitRedisPrefix: "test",
	}
	
	logger := createTestLogger()
	rateLimiter := NewRateLimiter(cfg, logger)
	
	mockRedis := &mockRedisForRateLimit{}
	rateLimiter.client = mockRedis
	
	ctx := context.Background()
	limit := RateLimit{requests: 5, window: 10 * time.Second}
	
	allowed, err := rateLimiter.checkLimit(ctx, "test-key", limit)
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}
	
	if !allowed {
		t.Error("first request should be allowed")
	}
	
	expectedKey := "test:test-key:10"
	if mockRedis.counters[expectedKey] != 1 {
		t.Errorf("expected counter 1, got %d", mockRedis.counters[expectedKey])
	}
	
	if mockRedis.ttls[expectedKey] != 10*time.Second {
		t.Errorf("expected TTL 10s, got %v", mockRedis.ttls[expectedKey])
	}
}

func TestRateLimiter_MultipleWindows(t *testing.T) {
	cfg := &Config{
		RedisHost:            "localhost",
		RedisPort:            "6379",
		RateLimitRedisPrefix: "test",
	}
	
	logger := createTestLogger()
	rateLimiter := NewRateLimiter(cfg, logger)
	
	mockRedis := &mockRedisForRateLimit{
		counters: map[string]int64{
			"test:test-key:1":   15,
			"test:test-key:120": 80,
		},
	}
	rateLimiter.client = mockRedis
	
	ctx := context.Background()
	allowed, err := rateLimiter.Allow(ctx, "test-key")
	
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}
	
	if allowed {
		t.Error("request should be blocked by 1-second window limit")
	}
}

func TestRateLimiter_EdgeCases(t *testing.T) {
	cfg := &Config{
		RedisHost:            "localhost",
		RedisPort:            "6379",
		RateLimitRedisPrefix: "test",
	}
	
	logger := createTestLogger()
	rateLimiter := NewRateLimiter(cfg, logger)
	
	tests := []struct {
		name         string
		counters     map[string]int64
		expectAllowed bool
	}{
		{
			name: "exactly at 1s limit",
			counters: map[string]int64{
				"test:test-key:1":   19,
				"test:test-key:120": 99,
			},
			expectAllowed: true,
		},
		{
			name: "exactly at 2m limit",
			counters: map[string]int64{
				"test:test-key:1":   19,
				"test:test-key:120": 99,
			},
			expectAllowed: true,
		},
		{
			name: "exceeds 1s limit by 1",
			counters: map[string]int64{
				"test:test-key:1":   20,
				"test:test-key:120": 50,
			},
			expectAllowed: false,
		},
		{
			name: "exceeds 2m limit by 1",
			counters: map[string]int64{
				"test:test-key:1":   10,
				"test:test-key:120": 100,
			},
			expectAllowed: false,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockRedis := &mockRedisForRateLimit{counters: tt.counters}
			rateLimiter.client = mockRedis
			
			ctx := context.Background()
			allowed, err := rateLimiter.Allow(ctx, "test-key")
			
			if err != nil {
				t.Errorf("expected no error, got %v", err)
			}
			
			if allowed != tt.expectAllowed {
				t.Errorf("expected allowed=%v, got %v", tt.expectAllowed, allowed)
			}
		})
	}
}