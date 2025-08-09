package internal

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"
)

type RateLimitConfig struct {
	Limit  int
	Window time.Duration
}

var leagueRateLimits = map[string]RateLimitConfig{
	"challenger":   {Limit: 25, Window: 10 * time.Second},
	"grandmaster":  {Limit: 25, Window: 10 * time.Second},
	"master":       {Limit: 25, Window: 10 * time.Second},
	"entries":      {Limit: 200, Window: 10 * time.Second},
	"by-puuid":     {Limit: 16000, Window: 10 * time.Second},
}

func NewChallengerHandler(riotClient *RiotAPIClient, rateLimiter *RateLimiter) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		
		// Debug logs
		fmt.Printf("Rate limiter config - Limit: %d, Window: %v\n", rateLimiter.Limit, rateLimiter.Window)
		fmt.Printf("Redis client options: %+v\n", rateLimiter.RedisClient.Options())
		
		allowed, err := rateLimiter.Allow(ctx, "challenger")
		if err != nil {
			fmt.Printf("Rate limiter error details: %v\n", err)
			http.Error(w, fmt.Sprintf("Rate limiter error: %v", err), http.StatusInternalServerError)
			return
		}
		if !allowed {
			http.Error(w, "Rate limit exceeded", http.StatusTooManyRequests)
			return
		}
		
		result, err := riotClient.GetChallengerLeague()
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadGateway)
			return
		}
		
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(result)
	}
}

func NewGrandmasterHandler(riotClient *RiotAPIClient, rateLimiter *RateLimiter) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		
		allowed, err := rateLimiter.Allow(ctx, "grandmaster")
		if err != nil {
			fmt.Printf("Grandmaster rate limiter error: %v\n", err)
			http.Error(w, fmt.Sprintf("Rate limiter error: %v", err), http.StatusInternalServerError)
			return
		}
		if !allowed {
			http.Error(w, "Rate limit exceeded", http.StatusTooManyRequests)
			return
		}
		
		result, err := riotClient.GetGrandmasterLeague()
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadGateway)
			return
		}
		
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(result)
	}
}

func NewMasterHandler(riotClient *RiotAPIClient, rateLimiter *RateLimiter) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		
		allowed, err := rateLimiter.Allow(ctx, "master")
		if err != nil {
			fmt.Printf("Master rate limiter error: %v\n", err)
			http.Error(w, fmt.Sprintf("Rate limiter error: %v", err), http.StatusInternalServerError)
			return
		}
		if !allowed {
			http.Error(w, "Rate limit exceeded", http.StatusTooManyRequests)
			return
		}
		
		result, err := riotClient.GetMasterLeague()
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadGateway)
			return
		}
		
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(result)
	}
}

func NewEntriesHandler(riotClient *RiotAPIClient, rateLimiter *RateLimiter) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		tier := r.URL.Query().Get("tier")
		division := r.URL.Query().Get("division")
		pageStr := r.URL.Query().Get("page")
		
		if tier == "" || division == "" {
			http.Error(w, "tier and division are required", http.StatusBadRequest)
			return
		}
		
		page := 1
		if pageStr != "" {
			if p, err := strconv.Atoi(pageStr); err == nil && p > 0 {
				page = p
			}
		}
		
		ctx := r.Context()
		rateLimitKey := fmt.Sprintf("entries:%s:%s", tier, division)
		
		allowed, err := rateLimiter.Allow(ctx, rateLimitKey)
		if err != nil {
			fmt.Printf("Entries rate limiter error: %v\n", err)
			http.Error(w, fmt.Sprintf("Rate limiter error: %v", err), http.StatusInternalServerError)
			return
		}
		if !allowed {
			http.Error(w, "Rate limit exceeded", http.StatusTooManyRequests)
			return
		}
		
		result, err := riotClient.GetLeagueEntries(tier, division, page)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadGateway)
			return
		}
		
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(result)
	}
}

func NewLeagueByPUUIDHandler(riotClient *RiotAPIClient, rateLimiter *RateLimiter) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		puuid := r.URL.Query().Get("puuid")
		if puuid == "" {
			http.Error(w, "puuid is required", http.StatusBadRequest)
			return
		}
		
		ctx := r.Context()
		allowed, err := rateLimiter.Allow(ctx, fmt.Sprintf("league:puuid:%s", puuid))
		if err != nil {
			fmt.Printf("League by PUUID rate limiter error: %v\n", err)
			http.Error(w, fmt.Sprintf("Rate limiter error: %v", err), http.StatusInternalServerError)
			return
		}
		if !allowed {
			http.Error(w, "Rate limit exceeded", http.StatusTooManyRequests)
			return
		}
		
		result, err := riotClient.GetLeagueByPUUID(puuid)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadGateway)
			return
		}
		
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(result)
	}
}