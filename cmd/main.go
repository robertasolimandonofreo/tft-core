package main

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/nats-io/nats.go"
	"github.com/robertasolimandonofreo/tft-core/internal"
)

var (
	cfg          *internal.Config
	ratelimiter  *internal.RateLimiter
	riotClient   *internal.RiotAPIClient
	natsClient   *internal.NATSClient
	cacheManager *internal.CacheManager
)

func withCORS(h http.HandlerFunc) http.HandlerFunc {
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
		h(w, r)
	}
}

func healthzHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status": "ok",
		"timestamp": time.Now().Unix(),
		"services": map[string]string{
			"redis": "connected",
			"nats": "connected",
		},
	})
}

func summonerHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	puuid := r.URL.Query().Get("puuid")
	if puuid == "" {
		http.Error(w, "puuid is required", http.StatusBadRequest)
		return
	}
	
	allowed, err := ratelimiter.Allow(ctx, "summoner:"+puuid)
	if err != nil {
		log.Printf("Rate limiter error: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}
	if !allowed {
		http.Error(w, "Rate limit exceeded", http.StatusTooManyRequests)
		return
	}
	
	result, err := riotClient.GetSummonerByPUUID(puuid)
	if err != nil {
		log.Printf("Riot API error: %v", err)
		http.Error(w, "Failed to fetch summoner data", http.StatusBadGateway)
		return
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

func searchPlayerHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	gameName := r.URL.Query().Get("gameName")
	tagLine := r.URL.Query().Get("tagLine")
	
	if gameName == "" {
		http.Error(w, "gameName is required", http.StatusBadRequest)
		return
	}
	
	if tagLine == "" {
		tagLine = "BR1"
	}
	
	allowed, err := ratelimiter.Allow(ctx, "search:"+gameName+":"+tagLine)
	if err != nil {
		log.Printf("Rate limiter error: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}
	if !allowed {
		http.Error(w, "Rate limit exceeded", http.StatusTooManyRequests)
		return
	}
	
	accountData, err := riotClient.GetAccountByGameName(gameName, tagLine)
	if err != nil {
		log.Printf("Error finding account: %v", err)
		http.Error(w, "Player not found", http.StatusNotFound)
		return
	}
	
	summonerData, err := riotClient.GetSummonerByPUUID(accountData.PUUID)
	if err != nil {
		log.Printf("Error fetching summoner data: %v", err)
		http.Error(w, "Failed to fetch player data", http.StatusBadGateway)
		return
	}
	
	result := map[string]interface{}{
		"account":  accountData,
		"summoner": summonerData,
		"puuid":    accountData.PUUID,
		"gameName": accountData.GameName,
		"tagLine":  accountData.TagLine,
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

func setupRoutes() {
	http.HandleFunc("/healthz", withCORS(healthzHandler))
	http.HandleFunc("/summoner", withCORS(summonerHandler))
	http.HandleFunc("/search/player", withCORS(searchPlayerHandler))
	
	http.HandleFunc("/league/challenger", withCORS(internal.NewChallengerHandler(riotClient, ratelimiter)))
	http.HandleFunc("/league/grandmaster", withCORS(internal.NewGrandmasterHandler(riotClient, ratelimiter)))
	http.HandleFunc("/league/master", withCORS(internal.NewMasterHandler(riotClient, ratelimiter)))
	http.HandleFunc("/league/entries", withCORS(internal.NewEntriesHandler(riotClient, ratelimiter)))
	http.HandleFunc("/league/by-puuid", withCORS(internal.NewLeagueByPUUIDHandler(riotClient, ratelimiter)))
}

func scheduleLeagueUpdates() {
	ticker := time.NewTicker(30 * time.Minute)
	go func() {
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				tasks := []internal.LeagueUpdateTask{
					{Type: "challenger", Region: cfg.RiotRegion},
					{Type: "grandmaster", Region: cfg.RiotRegion},
					{Type: "master", Region: cfg.RiotRegion},
				}
				
				for _, task := range tasks {
					if err := natsClient.PublishLeagueUpdateTask(task); err != nil {
						log.Printf("Error publishing league update task: %v", err)
					}
				}
			}
		}
	}()
	log.Println("League update scheduler started")
}

func main() {
	cfg = internal.LoadConfig()

	ratelimiter = internal.NewRateLimiter(cfg, 100, 10*time.Second)

	cacheManager = internal.NewCacheManager(cfg)

	riotClient = internal.NewRiotAPIClient(cfg, cacheManager)

	var err error
	natsClient, err = internal.NewNATSClient(cfg)
	if err != nil {
		log.Fatalf("Error connecting to NATS: %v", err)
	}
	defer natsClient.Conn.Close()

	riotClient.SetNATSClient(natsClient)

	_, err = natsClient.StartSummonerFetchWorker(func(msg *nats.Msg) {
		log.Printf("Message received in summoner worker: %s", string(msg.Data))
	})
	if err != nil {
		log.Fatalf("Error starting summoner NATS worker: %v", err)
	}

	_, err = natsClient.StartLeagueUpdateWorker(riotClient, cacheManager)
	if err != nil {
		log.Fatalf("Error starting league NATS worker: %v", err)
	}

	_, err = natsClient.StartSummonerNameWorker(riotClient, cacheManager)
	if err != nil {
		log.Fatalf("Error starting summoner name NATS worker: %v", err)
	}

	setupRoutes()
	scheduleLeagueUpdates()

	port := cfg.AppPort
	if port == "" {
		port = "8000"
	}

	server := &http.Server{
		Addr:         ":" + port,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	go func() {
		log.Printf("Server starting on port %s", port)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Error starting server: %v", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down server...")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		log.Fatalf("Server forced to shutdown: %v", err)
	}

	log.Println("Server exited")
}