package main

import (
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/CryptoElementals/common/cache"
	"github.com/CryptoElementals/common/config"
	"github.com/CryptoElementals/common/db"
	"github.com/CryptoElementals/common/log"
	"github.com/CryptoElementals/common/redis"
	"github.com/CryptoElementals/common/server"
	"github.com/CryptoElementals/common/session"
)

// Version information
var (
	Version   = "1.0.0"
	BuildTime = "unknown"
	GitCommit = "unknown"
)

func main() {
	// Parse command line arguments
	var (
		configPath = flag.String("config", "config.yaml", "Configuration file path")
		version    = flag.Bool("version", false, "Show version information")
	)
	flag.Parse()

	// Show version information
	if *version {
		fmt.Printf("BeastRoyale Backend Server\n")
		fmt.Printf("Version: %s\n", Version)
		fmt.Printf("Build Time: %s\n", BuildTime)
		fmt.Printf("Git Commit: %s\n", GitCommit)
		os.Exit(0)
	}

	// Load configuration
	cfg, err := config.LoadAppConfig(*configPath)
	if err != nil {
		fmt.Printf("Failed to load configuration file: %v\n", err)
		os.Exit(1)
	}

	// Validate configuration
	err = config.ValidateAppConfig(cfg)
	if err != nil {
		fmt.Printf("Configuration validation failed: %v\n", err)
		os.Exit(1)
	}

	// Initialize logger
	err = log.InitGlobalLogger(&cfg.LogCfg)
	if err != nil {
		fmt.Printf("Failed to initialize logger: %v\n", err)
		os.Exit(1)
	}
	log.Info("Logger system initialized successfully")

	// Initialize Redis
	err = redis.Init(&cfg.RedisCfg)
	if err != nil {
		log.Fatalf("Failed to initialize Redis: %v", err)
	}
	log.Info("Redis connection initialized successfully")

	// Initialize database
	err = db.Init(&cfg.DbCfg)
	if err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}
	log.Info("Database connection initialized successfully")

	// Execute database migration
	err = db.Migrate()
	if err != nil {
		log.Fatalf("Failed to migrate database: %v", err)
	}
	log.Info("Database migration completed successfully")

	// Get Redis connection pool
	pool, err := redis.GetRedigoPool()
	if err != nil {
		log.Fatalf("Failed to get Redis connection pool: %v", err)
	}

	// Initialize session store
	sessionStore, err := session.New(pool)
	if err != nil {
		log.Fatalf("Failed to initialize session store: %v", err)
	}

	// Initialize Redis cache
	redisCache, err := cache.NewRedisCache()
	if err != nil {
		log.Fatalf("Failed to initialize Redis cache: %v", err)
	}

	// Create and start server
	log.Infof("Starting BeastRoyale backend server on port: %d", cfg.ServerCfg.Port)
	svr := server.New(&cfg.ServerCfg, sessionStore, redisCache)
	svr.Run()

	// Wait for interrupt signal
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
	<-sigs

	log.Info("Received shutdown signal, closing server...")

	// Gracefully shutdown server
	err = svr.Stop()
	if err != nil {
		log.Errorf("Error occurred while closing server: %v", err)
		os.Exit(1)
	}

	log.Info("Server closed successfully")
}
