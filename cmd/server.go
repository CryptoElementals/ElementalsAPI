package cmd

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/CryptoElementals/common/cache"
	"github.com/CryptoElementals/common/config"
	"github.com/CryptoElementals/common/cron"
	"github.com/CryptoElementals/common/db"
	"github.com/CryptoElementals/common/log"
	"github.com/CryptoElementals/common/redis"
	"github.com/CryptoElementals/common/server"
	"github.com/CryptoElementals/common/session"
	"github.com/spf13/cobra"
)

// runCmd represents the run command
var runCmd = &cobra.Command{
	Use:   "run",
	Short: "Start the BeastRoyale backend server",
	Long: `Start the BeastRoyale backend server with the specified configuration.
	
The server provides:
- User authentication via Web3 wallet
- User profile management
- Game statistics tracking
- RESTful API endpoints`,
	Run: func(cmd *cobra.Command, args []string) {
		if err := startServer(); err != nil {
			fmt.Printf("Failed to start server: %v\n", err)
			os.Exit(1)
		}
	},
}

func init() {
	rootCmd.AddCommand(runCmd)
	runCmd.MarkFlagRequired("config")
}

// startServer starts the backend server
func startServer() error {
	// Load configuration
	cfg, err := config.LoadAppConfig(configPath)
	if err != nil {
		return fmt.Errorf("failed to load configuration file: %w", err)
	}

	// Validate configuration
	if err := config.ValidateAppConfig(cfg); err != nil {
		return fmt.Errorf("configuration validation failed: %w", err)
	}

	// Initialize logger
	if err := log.InitGlobalLogger(&cfg.LogCfg); err != nil {
		return fmt.Errorf("failed to initialize logger: %w", err)
	}
	log.Info("Logger system initialized successfully")

	// Initialize Redis
	if err := redis.Init(&cfg.RedisCfg); err != nil {
		return fmt.Errorf("failed to initialize Redis: %w", err)
	}
	log.Info("Redis connection initialized successfully")

	// Initialize database
	if err := db.Init(&cfg.DbCfg); err != nil {
		return fmt.Errorf("failed to initialize database: %w", err)
	}
	log.Info("Database connection initialized successfully")

	// Get Redis connection pool
	pool, err := redis.GetRedigoPool()
	if err != nil {
		return fmt.Errorf("failed to get Redis connection pool: %w", err)
	}

	// Initialize session store
	sessionStore, err := session.New(pool)
	if err != nil {
		return fmt.Errorf("failed to initialize session store: %w", err)
	}

	// Initialize Redis cache
	redisCache, err := cache.NewRedisCache()
	if err != nil {
		return fmt.Errorf("failed to initialize Redis cache: %w", err)
	}

	// 注册匹配任务
	cron.RegisterMatchmakingTask()

	// 创建并启动调度器
	scheduler := cron.NewScheduler()
	scheduler.RegisterAllTasks()

	// 创建上下文用于优雅关闭
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// 启动调度器
	scheduler.Start(ctx)
	log.Info("匹配任务调度器已启动")

	// Create and start server
	log.Infof("Starting BeastRoyale backend server on port: %d", cfg.ServerCfg.Port)
	svr := server.New(&cfg.ServerCfg, sessionStore, redisCache)
	svr.Run()

	// Wait for interrupt signal
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
	<-sigs

	log.Info("Received shutdown signal, closing server...")

	// 取消上下文，停止调度器
	cancel()

	// Gracefully shutdown server
	if err := svr.Stop(); err != nil {
		log.Errorf("Error occurred while closing server: %v", err)
		return err
	}

	log.Info("Server closed successfully")
	return nil
}
