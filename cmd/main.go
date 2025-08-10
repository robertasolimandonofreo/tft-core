package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/robertasolimandonofreo/tft-core/internal"
)

func main() {
	cfg, err := internal.LoadConfig()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	var dbManager *internal.DatabaseManager
	if cfg.DatabaseEnabled {
		dbManager = internal.NewDatabaseManager(cfg)
		if dbManager != nil {
			defer dbManager.Close()
		}
	}

	cacheManager := internal.NewCacheManager(cfg, dbManager)
	rateLimiter := internal.NewRateLimiter(cfg)
	riotClient := internal.NewRiotAPIClient(cfg, cacheManager)

	var natsClient *internal.NATSClient
	if cfg.NATSUrl != "" {
		natsClient, err = internal.NewNATSClient(cfg)
		if err != nil {
			log.Printf("Warning: NATS connection failed: %v", err)
		} else {
			defer natsClient.Conn.Close()
			riotClient.SetNATSClient(natsClient)
			setupNATSWorkers(natsClient, riotClient, cacheManager)
			scheduleLeagueUpdates(natsClient, cfg.RiotRegion)
		}
	}

	setupRoutes(riotClient, rateLimiter)
	startServer(cfg.AppPort)
}

func setupNATSWorkers(natsClient *internal.NATSClient, riotClient *internal.RiotAPIClient, cache *internal.CacheManager) {
	if _, err := natsClient.StartSummonerNameWorker(riotClient, cache); err != nil {
		log.Printf("Warning: Failed to start summoner name worker: %v", err)
	}

	if _, err := natsClient.StartLeagueUpdateWorker(riotClient, cache); err != nil {
		log.Printf("Warning: Failed to start league update worker: %v", err)
	}
}

func scheduleLeagueUpdates(natsClient *internal.NATSClient, region string) {
	ticker := time.NewTicker(30 * time.Minute)
	go func() {
		defer ticker.Stop()
		for range ticker.C {
			tasks := []internal.LeagueUpdateTask{
				{Type: "challenger", Region: region},
				{Type: "grandmaster", Region: region},
				{Type: "master", Region: region},
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

func setupRoutes(riotClient *internal.RiotAPIClient, rateLimiter *internal.RateLimiter) {
	http.HandleFunc("/healthz", internal.HealthHandler())
	http.HandleFunc("/summoner", internal.SummonerHandler(riotClient, rateLimiter))
	http.HandleFunc("/search/player", internal.SearchPlayerHandler(riotClient, rateLimiter))
	http.HandleFunc("/league/challenger", internal.ChallengerHandler(riotClient, rateLimiter))
	http.HandleFunc("/league/grandmaster", internal.GrandmasterHandler(riotClient, rateLimiter))
	http.HandleFunc("/league/master", internal.MasterHandler(riotClient, rateLimiter))
	http.HandleFunc("/league/entries", internal.EntriesHandler(riotClient, rateLimiter))
	http.HandleFunc("/league/by-puuid", internal.LeagueByPUUIDHandler(riotClient, rateLimiter))
}

func startServer(port string) {
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
			log.Fatalf("Server failed to start: %v", err)
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