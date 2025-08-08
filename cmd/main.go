package main

import (
	"encoding/json"
	"log"
	"net/http"
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
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}
		h(w, r)
	}
}

func healthzHandler(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("ok"))
}

func summonerHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	puuid := r.URL.Query().Get("puuid")
	if puuid == "" {
		http.Error(w, "puuid is required", http.StatusBadRequest)
		return
	}
	allowed, err := ratelimiter.Allow(ctx, puuid)
	if err != nil {
		http.Error(w, "Error on rate limiter", http.StatusInternalServerError)
		return
	}
	if !allowed {
		http.Error(w, "Rate limit exceeded", http.StatusTooManyRequests)
		return
	}
	result, err := riotClient.GetSummonerByPUUID(puuid)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadGateway)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

func setupRoutes() {
	http.HandleFunc("/healthz", withCORS(healthzHandler))
	http.HandleFunc("/summoner", withCORS(summonerHandler))
	
	http.HandleFunc("/league/challenger", withCORS(internal.NewChallengerHandler(riotClient, ratelimiter)))
	http.HandleFunc("/league/grandmaster", withCORS(internal.NewGrandmasterHandler(riotClient, ratelimiter)))
	http.HandleFunc("/league/master", withCORS(internal.NewMasterHandler(riotClient, ratelimiter)))
	http.HandleFunc("/league/entries", withCORS(internal.NewEntriesHandler(riotClient, ratelimiter)))
	http.HandleFunc("/league/by-puuid", withCORS(internal.NewLeagueByPUUIDHandler(riotClient, ratelimiter)))
	http.HandleFunc("/league/rated-ladder", withCORS(internal.NewRatedLadderHandler(riotClient, ratelimiter)))
}

func scheduleLeagueUpdates() {
	ticker := time.NewTicker(30 * time.Minute)
	go func() {
		for range ticker.C {
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
	}()
	log.Println("League update scheduler started")
}

func main() {
	cfg = internal.LoadConfig()

	ratelimiter = internal.NewRateLimiter(cfg, 5, 10*time.Second)

	cacheManager = internal.NewCacheManager(cfg)

	riotClient = internal.NewRiotAPIClient(cfg, cacheManager)

	var err error
	natsClient, err = internal.NewNATSClient(cfg)
	if err != nil {
		log.Fatalf("Error connecting to NATS: %v", err)
	}
	defer natsClient.Conn.Close()

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

	setupRoutes()

	scheduleLeagueUpdates()

	port := cfg.AppPort
	if port == "" {
		port = "8000"
	}
	log.Printf("Server started on port %s", port)
	if err := http.ListenAndServe(":"+port, nil); err != nil {
		log.Fatalf("Error starting server: %v", err)
	}
}