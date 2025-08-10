package internal

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"time"
)

type APIError struct {
	Message string `json:"message"`
	Status  int    `json:"status"`
}

func (e APIError) Error() string {
	return e.Message
}

func NewAPIError(message string, status int) APIError {
	return APIError{Message: message, Status: status}
}

func writeError(w http.ResponseWriter, err error) {
	var apiErr APIError
	if e, ok := err.(APIError); ok {
		apiErr = e
	} else {
		apiErr = NewAPIError("Internal server error", http.StatusInternalServerError)
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(apiErr.Status)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"error":   apiErr.Message,
		"status":  apiErr.Status,
		"timestamp": time.Now().Unix(),
	})
}

func writeJSON(w http.ResponseWriter, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(data); err != nil {
		writeError(w, NewAPIError("Failed to encode response", http.StatusInternalServerError))
	}
}

func withCORS(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")
		if origin == "" {
			origin = "*"
		}

		w.Header().Set("Access-Control-Allow-Origin", origin)
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		w.Header().Set("Access-Control-Allow-Credentials", "true")

		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		next(w, r)
	}
}

func withRateLimit(rateLimiter *RateLimiter, key string) func(http.HandlerFunc) http.HandlerFunc {
	return func(next http.HandlerFunc) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			allowed, err := rateLimiter.Allow(r.Context(), key)
			if err != nil {
				writeError(w, NewAPIError("Rate limiter error", http.StatusInternalServerError))
				return
			}
			if !allowed {
				writeError(w, NewAPIError("Rate limit exceeded", http.StatusTooManyRequests))
				return
			}
			next(w, r)
		}
	}
}

func HealthHandler() http.HandlerFunc {
	return withCORS(func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, map[string]interface{}{
			"status":    "ok",
			"timestamp": time.Now().Unix(),
			"services": map[string]string{
				"redis": "connected",
				"nats":  "connected",
			},
		})
	})
}

func SummonerHandler(riotClient *RiotAPIClient, rateLimiter *RateLimiter) http.HandlerFunc {
	return withCORS(withRateLimit(rateLimiter, "summoner")(func(w http.ResponseWriter, r *http.Request) {
		puuid := r.URL.Query().Get("puuid")
		if puuid == "" {
			writeError(w, NewAPIError("puuid is required", http.StatusBadRequest))
			return
		}

		result, err := riotClient.GetSummonerByPUUID(puuid)
		if err != nil {
			if strings.Contains(err.Error(), "404") {
				writeError(w, NewAPIError("Summoner not found", http.StatusNotFound))
				return
			}
			writeError(w, NewAPIError("Failed to fetch summoner data", http.StatusBadGateway))
			return
		}

		writeJSON(w, result)
	}))
}

func SearchPlayerHandler(riotClient *RiotAPIClient, rateLimiter *RateLimiter) http.HandlerFunc {
	return withCORS(withRateLimit(rateLimiter, "search")(func(w http.ResponseWriter, r *http.Request) {
		gameName := r.URL.Query().Get("gameName")
		tagLine := r.URL.Query().Get("tagLine")

		if gameName == "" {
			writeError(w, NewAPIError("gameName is required", http.StatusBadRequest))
			return
		}

		if tagLine == "" {
			tagLine = "BR1"
		}

		accountData, err := riotClient.GetAccountByGameName(gameName, tagLine)
		if err != nil {
			if strings.Contains(err.Error(), "404") {
				writeError(w, NewAPIError("Player not found", http.StatusNotFound))
				return
			}
			writeError(w, NewAPIError("Failed to fetch account data", http.StatusBadGateway))
			return
		}

		summonerData, _ := riotClient.GetSummonerByPUUID(accountData.PUUID)
		leagueData, _ := riotClient.GetLeagueByPUUID(accountData.PUUID)

		var tftLeague *LeagueEntry
		if leagueData != nil && len(leagueData) > 0 {
			for _, entry := range leagueData {
				if entry.QueueType == "RANKED_TFT" {
					tftLeague = &entry
					break
				}
			}
		}

		result := map[string]interface{}{
			"account":  accountData,
			"summoner": summonerData,
			"puuid":    accountData.PUUID,
			"gameName": accountData.GameName,
			"tagLine":  accountData.TagLine,
			"league":   tftLeague,
		}

		writeJSON(w, result)
	}))
}

func ChallengerHandler(riotClient *RiotAPIClient, rateLimiter *RateLimiter) http.HandlerFunc {
	return withCORS(withRateLimit(rateLimiter, "challenger")(func(w http.ResponseWriter, r *http.Request) {
		result, err := riotClient.GetChallengerLeague()
		if err != nil {
			writeError(w, NewAPIError("Failed to fetch challenger league", http.StatusBadGateway))
			return
		}
		writeJSON(w, result)
	}))
}

func GrandmasterHandler(riotClient *RiotAPIClient, rateLimiter *RateLimiter) http.HandlerFunc {
	return withCORS(withRateLimit(rateLimiter, "grandmaster")(func(w http.ResponseWriter, r *http.Request) {
		result, err := riotClient.GetGrandmasterLeague()
		if err != nil {
			writeError(w, NewAPIError("Failed to fetch grandmaster league", http.StatusBadGateway))
			return
		}
		writeJSON(w, result)
	}))
}

func MasterHandler(riotClient *RiotAPIClient, rateLimiter *RateLimiter) http.HandlerFunc {
	return withCORS(withRateLimit(rateLimiter, "master")(func(w http.ResponseWriter, r *http.Request) {
		result, err := riotClient.GetMasterLeague()
		if err != nil {
			writeError(w, NewAPIError("Failed to fetch master league", http.StatusBadGateway))
			return
		}
		writeJSON(w, result)
	}))
}

func EntriesHandler(riotClient *RiotAPIClient, rateLimiter *RateLimiter) http.HandlerFunc {
	return withCORS(withRateLimit(rateLimiter, "entries")(func(w http.ResponseWriter, r *http.Request) {
		tier := r.URL.Query().Get("tier")
		division := r.URL.Query().Get("division")
		pageStr := r.URL.Query().Get("page")

		if tier == "" || division == "" {
			writeError(w, NewAPIError("tier and division are required", http.StatusBadRequest))
			return
		}

		page := 1
		if pageStr != "" {
			if p, err := strconv.Atoi(pageStr); err == nil && p > 0 {
				page = p
			}
		}

		result, err := riotClient.GetLeagueEntries(tier, division, page)
		if err != nil {
			writeError(w, NewAPIError("Failed to fetch league entries", http.StatusBadGateway))
			return
		}

		writeJSON(w, result)
	}))
}

func LeagueByPUUIDHandler(riotClient *RiotAPIClient, rateLimiter *RateLimiter) http.HandlerFunc {
	return withCORS(withRateLimit(rateLimiter, "league-by-puuid")(func(w http.ResponseWriter, r *http.Request) {
		puuid := r.URL.Query().Get("puuid")
		if puuid == "" {
			writeError(w, NewAPIError("puuid is required", http.StatusBadRequest))
			return
		}

		result, err := riotClient.GetLeagueByPUUID(puuid)
		if err != nil {
			writeError(w, NewAPIError("Failed to fetch league data", http.StatusBadGateway))
			return
		}

		writeJSON(w, result)
	}))
}