package main

import (
	"context"
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
		panic("Failed to load config: " + err.Error())
	}

	logger := internal.NewLogger(cfg)
	metrics := internal.NewMetricsCollector(logger)
	
	logger.Info("service_starting").
		Component("main").
		Operation("startup").
		Meta("port", cfg.AppPort).
		Meta("environment", cfg.AppEnv).
		Log()

	var dbManager *internal.DatabaseManager
	if cfg.DatabaseEnabled {
		dbManager = internal.NewDatabaseManager(cfg)
		if dbManager != nil {
			defer dbManager.Close()
			logger.Info("database_connected").Component("database").Log()
		} else {
			logger.Warn("database_connection_failed").Component("database").Log()
		}
	}

	cacheManager := internal.NewCacheManager(cfg, dbManager)
	rateLimiter := internal.NewRateLimiter(cfg, logger)
	riotClient := internal.NewRiotAPIClient(cfg, cacheManager, logger, metrics)

	var natsClient *internal.NATSClient
	if cfg.NATSUrl != "" {
		natsClient, err = internal.NewNATSClient(cfg)
		if err != nil {
			logger.Error("nats_connection_failed").
				Component("nats").
				Err(err).
				Log()
		} else {
			defer natsClient.Conn.Close()
			riotClient.SetNATSClient(natsClient)
			setupNATSWorkers(natsClient, riotClient, cacheManager, logger)
			scheduleLeagueUpdates(natsClient, cfg.RiotRegion, logger)
			logger.Info("nats_connected").Component("nats").Log()
		}
	}

	middleware := internal.NewLoggingMiddleware(logger, metrics)
	setupRoutes(riotClient, rateLimiter, middleware, logger, metrics)
	startServer(cfg.AppPort, logger)
}

func setupNATSWorkers(natsClient *internal.NATSClient, riotClient *internal.RiotAPIClient, cache *internal.CacheManager, logger *internal.Logger) {
	if _, err := natsClient.StartSummonerNameWorker(riotClient, cache); err != nil {
		logger.Error("summoner_name_worker_failed").
			Component("nats").
			Operation("start_worker").
			Err(err).
			Log()
	} else {
		logger.Info("summoner_name_worker_started").
			Component("nats").
			Operation("start_worker").
			Log()
	}

	if _, err := natsClient.StartLeagueUpdateWorker(riotClient, cache); err != nil {
		logger.Error("league_update_worker_failed").
			Component("nats").
			Operation("start_worker").
			Err(err).
			Log()
	} else {
		logger.Info("league_update_worker_started").
			Component("nats").
			Operation("start_worker").
			Log()
	}
}

func scheduleLeagueUpdates(natsClient *internal.NATSClient, region string, logger *internal.Logger) {
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
					logger.Error("league_update_task_failed").
						Component("nats").
						Operation("publish_task").
						Err(err).
						Meta("task_type", task.Type).
						Log()
				} else {
					logger.Debug("league_update_task_published").
						Component("nats").
						Operation("publish_task").
						Meta("task_type", task.Type).
						Log()
				}
			}
		}
	}()
	
	logger.Info("league_update_scheduler_started").
		Component("scheduler").
		Operation("start").
		Meta("interval", "30m").
		Log()
}

func setupRoutes(riotClient *internal.RiotAPIClient, rateLimiter *internal.RateLimiter, middleware *internal.LoggingMiddleware, logger *internal.Logger, metrics *internal.MetricsCollector) {
	http.HandleFunc("/healthz", middleware.Handler(internal.HealthHandler(logger)))
	http.HandleFunc("/summoner", middleware.Handler(internal.SummonerHandler(riotClient, rateLimiter, logger)))
	http.HandleFunc("/search/player", middleware.Handler(internal.SearchPlayerHandler(riotClient, rateLimiter, logger)))
	http.HandleFunc("/league/challenger", middleware.Handler(internal.ChallengerHandler(riotClient, rateLimiter, logger)))
	http.HandleFunc("/league/grandmaster", middleware.Handler(internal.GrandmasterHandler(riotClient, rateLimiter, logger)))
	http.HandleFunc("/league/master", middleware.Handler(internal.MasterHandler(riotClient, rateLimiter, logger)))
	http.HandleFunc("/league/entries", middleware.Handler(internal.EntriesHandler(riotClient, rateLimiter, logger)))
	http.HandleFunc("/league/by-puuid", middleware.Handler(internal.LeagueByPUUIDHandler(riotClient, rateLimiter, logger)))
	http.HandleFunc("/metrics", middleware.Handler(internal.MetricsHandler(logger, metrics)))
	
	logger.Info("routes_configured").Component("http").Log()
}

func startServer(port string, logger *internal.Logger) {
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
		logger.Info("server_starting").
			Component("http").
			Operation("listen").
			Meta("port", port).
			Log()
			
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Error("server_start_failed").
				Component("http").
				Operation("listen").
				Err(err).
				Log()
			os.Exit(1)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Info("shutdown_signal_received").
		Component("http").
		Operation("shutdown").
		Log()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		logger.Error("server_shutdown_failed").
			Component("http").
			Operation("shutdown").
			Err(err).
			Log()
		os.Exit(1)
	}

	logger.Info("server_shutdown_completed").
		Component("http").
		Operation("shutdown").
		Log()
}