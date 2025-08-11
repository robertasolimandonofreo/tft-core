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

func writeError(w http.ResponseWriter, err error, logger *Logger, r *http.Request) {
	var apiErr APIError
	if e, ok := err.(APIError); ok {
		apiErr = e
	} else {
		apiErr = NewAPIError("Internal server error", http.StatusInternalServerError)
	}

	requestID := GetRequestID(r.Context())
	
	logger.Error("api_error").
		Component("http").
		Operation("write_error").
		HTTP(r.Method, r.URL.Path, apiErr.Status).
		Request(r.UserAgent(), r.RemoteAddr, requestID).
		Err(err).
		ErrorCode(strconv.Itoa(apiErr.Status)).
		Log()

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(apiErr.Status)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"error":     apiErr.Message,
		"status":    apiErr.Status,
		"timestamp": time.Now().Unix(),
		"requestId": requestID,
	})
}

func writeJSON(w http.ResponseWriter, data interface{}, logger *Logger, r *http.Request) {
	requestID := GetRequestID(r.Context())
	
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(data); err != nil {
		logger.Error("json_encode_failed").
			Component("http").
			Operation("write_json").
			Request("", "", requestID).
			Err(err).
			Log()
		writeError(w, NewAPIError("Failed to encode response", http.StatusInternalServerError), logger, r)
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

func withRateLimit(rateLimiter *RateLimiter, key string, logger *Logger) func(http.HandlerFunc) http.HandlerFunc {
	return func(next http.HandlerFunc) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			requestID := GetRequestID(r.Context())
			
			allowed, err := rateLimiter.Allow(r.Context(), key)
			if err != nil {
				logger.Error("rate_limiter_error").
					Component("rate_limiter").
					Operation("check_limit").
					Request("", "", requestID).
					Err(err).
					Meta("key", key).
					Log()
				writeError(w, NewAPIError("Rate limiter error", http.StatusInternalServerError), logger, r)
				return
			}
			
			if !allowed {
				logger.Warn("rate_limit_exceeded").
					Component("rate_limiter").
					Operation("check_limit").
					Request("", "", requestID).
					Meta("key", key).
					Log()
				writeError(w, NewAPIError("Rate limit exceeded", http.StatusTooManyRequests), logger, r)
				return
			}
			
			next(w, r)
		}
	}
}

func HealthHandler(logger *Logger) http.HandlerFunc {
	return withCORS(func(w http.ResponseWriter, r *http.Request) {
		logger.Debug("health_check").
			Component("health").
			Operation("check").
			Log()
			
		writeJSON(w, map[string]interface{}{
			"status":    "ok",
			"timestamp": time.Now().Unix(),
			"services": map[string]string{
				"redis": "connected",
				"nats":  "connected",
			},
		}, logger, r)
	})
}

func SummonerHandler(riotClient *RiotAPIClient, rateLimiter *RateLimiter, logger *Logger) http.HandlerFunc {
	return withCORS(withRateLimit(rateLimiter, "summoner", logger)(func(w http.ResponseWriter, r *http.Request) {
		puuid := r.URL.Query().Get("puuid")
		requestID := GetRequestID(r.Context())
		
		if puuid == "" {
			logger.Warn("missing_puuid_parameter").
				Component("summoner").
				Operation("get_summoner").
				Request("", "", requestID).
				Log()
			writeError(w, NewAPIError("puuid is required", http.StatusBadRequest), logger, r)
			return
		}

		logger.Info("summoner_request").
			Component("summoner").
			Operation("get_summoner").
			Request("", "", requestID).
			Game(puuid, "", "").
			Log()

		result, err := riotClient.GetSummonerByPUUID(puuid)
		if err != nil {
			if strings.Contains(err.Error(), "404") {
				logger.Warn("summoner_not_found").
					Component("summoner").
					Operation("get_summoner").
					Request("", "", requestID).
					Game(puuid, "", "").
					Err(err).
					Log()
				writeError(w, NewAPIError("Summoner not found", http.StatusNotFound), logger, r)
				return
			}
			logger.Error("summoner_fetch_failed").
				Component("summoner").
				Operation("get_summoner").
				Request("", "", requestID).
				Game(puuid, "", "").
				Err(err).
				Log()
			writeError(w, NewAPIError("Failed to fetch summoner data", http.StatusBadGateway), logger, r)
			return
		}

		logger.Info("summoner_success").
			Component("summoner").
			Operation("get_summoner").
			Request("", "", requestID).
			Game(puuid, "", "").
			Log()

		writeJSON(w, result, logger, r)
	}))
}

func SearchPlayerHandler(riotClient *RiotAPIClient, rateLimiter *RateLimiter, logger *Logger) http.HandlerFunc {
	return withCORS(withRateLimit(rateLimiter, "search", logger)(func(w http.ResponseWriter, r *http.Request) {
		gameName := r.URL.Query().Get("gameName")
		tagLine := r.URL.Query().Get("tagLine")
		requestID := GetRequestID(r.Context())

		if gameName == "" {
			logger.Warn("missing_gamename_parameter").
				Component("search").
				Operation("search_player").
				Request("", "", requestID).
				Log()
			writeError(w, NewAPIError("gameName is required", http.StatusBadRequest), logger, r)
			return
		}

		if tagLine == "" {
			tagLine = "BR1"
		}

		logger.Info("player_search_request").
			Component("search").
			Operation("search_player").
			Request("", "", requestID).
			Meta("game_name", gameName).
			Meta("tag_line", tagLine).
			Log()

		accountData, err := riotClient.GetAccountByGameName(gameName, tagLine)
		if err != nil {
			if strings.Contains(err.Error(), "404") {
				logger.Warn("player_not_found").
					Component("search").
					Operation("search_player").
					Request("", "", requestID).
					Meta("game_name", gameName).
					Meta("tag_line", tagLine).
					Err(err).
					Log()
				writeError(w, NewAPIError("Player not found", http.StatusNotFound), logger, r)
				return
			}
			logger.Error("account_fetch_failed").
				Component("search").
				Operation("search_player").
				Request("", "", requestID).
				Meta("game_name", gameName).
				Meta("tag_line", tagLine).
				Err(err).
				Log()
			writeError(w, NewAPIError("Failed to fetch account data", http.StatusBadGateway), logger, r)
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

		logger.Info("player_search_success").
			Component("search").
			Operation("search_player").
			Request("", "", requestID).
			Game(accountData.PUUID, "", "").
			Meta("game_name", gameName).
			Meta("tag_line", tagLine).
			Log()

		writeJSON(w, result, logger, r)
	}))
}

func ChallengerHandler(riotClient *RiotAPIClient, rateLimiter *RateLimiter, logger *Logger) http.HandlerFunc {
	return withCORS(withRateLimit(rateLimiter, "challenger", logger)(func(w http.ResponseWriter, r *http.Request) {
		requestID := GetRequestID(r.Context())
		
		logger.Info("challenger_request").
			Component("league").
			Operation("get_challenger").
			Request("", "", requestID).
			Log()

		result, err := riotClient.GetChallengerLeague()
		if err != nil {
			logger.Error("challenger_fetch_failed").
				Component("league").
				Operation("get_challenger").
				Request("", "", requestID).
				Err(err).
				Log()
			writeError(w, NewAPIError("Failed to fetch challenger league", http.StatusBadGateway), logger, r)
			return
		}
		
		logger.Info("challenger_success").
			Component("league").
			Operation("get_challenger").
			Request("", "", requestID).
			Meta("entries_count", len(result.Entries)).
			Log()
			
		writeJSON(w, result, logger, r)
	}))
}

func GrandmasterHandler(riotClient *RiotAPIClient, rateLimiter *RateLimiter, logger *Logger) http.HandlerFunc {
	return withCORS(withRateLimit(rateLimiter, "grandmaster", logger)(func(w http.ResponseWriter, r *http.Request) {
		requestID := GetRequestID(r.Context())
		
		logger.Info("grandmaster_request").
			Component("league").
			Operation("get_grandmaster").
			Request("", "", requestID).
			Log()

		result, err := riotClient.GetGrandmasterLeague()
		if err != nil {
			logger.Error("grandmaster_fetch_failed").
				Component("league").
				Operation("get_grandmaster").
				Request("", "", requestID).
				Err(err).
				Log()
			writeError(w, NewAPIError("Failed to fetch grandmaster league", http.StatusBadGateway), logger, r)
			return
		}
		
		logger.Info("grandmaster_success").
			Component("league").
			Operation("get_grandmaster").
			Request("", "", requestID).
			Meta("entries_count", len(result.Entries)).
			Log()
			
		writeJSON(w, result, logger, r)
	}))
}

func MasterHandler(riotClient *RiotAPIClient, rateLimiter *RateLimiter, logger *Logger) http.HandlerFunc {
	return withCORS(withRateLimit(rateLimiter, "master", logger)(func(w http.ResponseWriter, r *http.Request) {
		requestID := GetRequestID(r.Context())
		
		logger.Info("master_request").
			Component("league").
			Operation("get_master").
			Request("", "", requestID).
			Log()

		result, err := riotClient.GetMasterLeague()
		if err != nil {
			logger.Error("master_fetch_failed").
				Component("league").
				Operation("get_master").
				Request("", "", requestID).
				Err(err).
				Log()
			writeError(w, NewAPIError("Failed to fetch master league", http.StatusBadGateway), logger, r)
			return
		}
		
		logger.Info("master_success").
			Component("league").
			Operation("get_master").
			Request("", "", requestID).
			Meta("entries_count", len(result.Entries)).
			Log()
			
		writeJSON(w, result, logger, r)
	}))
}

func EntriesHandler(riotClient *RiotAPIClient, rateLimiter *RateLimiter, logger *Logger) http.HandlerFunc {
	return withCORS(withRateLimit(rateLimiter, "entries", logger)(func(w http.ResponseWriter, r *http.Request) {
		tier := r.URL.Query().Get("tier")
		division := r.URL.Query().Get("division")
		pageStr := r.URL.Query().Get("page")
		requestID := GetRequestID(r.Context())

		if tier == "" || division == "" {
			logger.Warn("missing_tier_division_parameters").
				Component("entries").
				Operation("get_entries").
				Request("", "", requestID).
				Log()
			writeError(w, NewAPIError("tier and division are required", http.StatusBadRequest), logger, r)
			return
		}

		page := 1
		if pageStr != "" {
			if p, err := strconv.Atoi(pageStr); err == nil && p > 0 {
				page = p
			}
		}

		logger.Info("entries_request").
			Component("entries").
			Operation("get_entries").
			Request("", "", requestID).
			Game("", "", tier).
			Meta("division", division).
			Meta("page", page).
			Log()

		result, err := riotClient.GetLeagueEntries(tier, division, page)
		if err != nil {
			logger.Error("entries_fetch_failed").
				Component("entries").
				Operation("get_entries").
				Request("", "", requestID).
				Game("", "", tier).
				Meta("division", division).
				Meta("page", page).
				Err(err).
				Log()
			writeError(w, NewAPIError("Failed to fetch league entries", http.StatusBadGateway), logger, r)
			return
		}

		logger.Info("entries_success").
			Component("entries").
			Operation("get_entries").
			Request("", "", requestID).
			Game("", "", tier).
			Meta("division", division).
			Meta("page", page).
			Meta("entries_count", len(result.Entries)).
			Log()

		writeJSON(w, result, logger, r)
	}))
}

func LeagueByPUUIDHandler(riotClient *RiotAPIClient, rateLimiter *RateLimiter, logger *Logger) http.HandlerFunc {
	return withCORS(withRateLimit(rateLimiter, "league-by-puuid", logger)(func(w http.ResponseWriter, r *http.Request) {
		puuid := r.URL.Query().Get("puuid")
		requestID := GetRequestID(r.Context())
		
		if puuid == "" {
			logger.Warn("missing_puuid_parameter").
				Component("league").
				Operation("get_league_by_puuid").
				Request("", "", requestID).
				Log()
			writeError(w, NewAPIError("puuid is required", http.StatusBadRequest), logger, r)
			return
		}

		logger.Info("league_by_puuid_request").
			Component("league").
			Operation("get_league_by_puuid").
			Request("", "", requestID).
			Game(puuid, "", "").
			Log()

		result, err := riotClient.GetLeagueByPUUID(puuid)
		if err != nil {
			logger.Error("league_by_puuid_fetch_failed").
				Component("league").
				Operation("get_league_by_puuid").
				Request("", "", requestID).
				Game(puuid, "", "").
				Err(err).
				Log()
			writeError(w, NewAPIError("Failed to fetch league data", http.StatusBadGateway), logger, r)
			return
		}

		logger.Info("league_by_puuid_success").
			Component("league").
			Operation("get_league_by_puuid").
			Request("", "", requestID).
			Game(puuid, "", "").
			Meta("entries_count", len(result)).
			Log()

		writeJSON(w, result, logger, r)
	}))
}

func MetricsHandler(logger *Logger, metrics *MetricsCollector) http.HandlerFunc {
	return withCORS(func(w http.ResponseWriter, r *http.Request) {
		requestID := GetRequestID(r.Context())
		
		logger.Debug("metrics_request").
			Component("metrics").
			Operation("get_metrics").
			Request("", "", requestID).
			Log()

		metricsData := metrics.GetMetrics()
		writeJSON(w, metricsData, logger, r)
	})
}