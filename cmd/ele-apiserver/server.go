package main

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
	"github.com/CryptoElementals/common/rpc/client"
	"github.com/CryptoElementals/common/server"
	"github.com/CryptoElementals/common/server/api"
	"github.com/CryptoElementals/common/session"
	"github.com/CryptoElementals/common/snowflake"
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
	cfg, err := config.LoadApiServerConfig(configPath)
	if err != nil {
		return fmt.Errorf("failed to load configuration file: %w", err)
	}

	// Validate configuration
	if err := config.ValidateApiServerConfig(cfg); err != nil {
		return fmt.Errorf("configuration validation failed: %w", err)
	}

	// Initialize logger
	if err := log.InitGlobalLogger(&cfg.LogCfg); err != nil {
		return fmt.Errorf("failed to initialize logger: %w", err)
	}
	log.Info("Logger system initialized successfully")

	snowflakeNode, err := snowflake.InitFromConfig(cfg.Snowflake.NodeID)
	if err != nil {
		return fmt.Errorf("failed to initialize snowflake: %w", err)
	}
	log.Infof("snowflake node id=%d", snowflakeNode)

	// Initialize Redis: default pool for API server (sessions, cache) + named pools per game environment (streams).
	var envRedis []*redis.ConfigWithName
	for i := range cfg.EnvironmentConfigs {
		env := &cfg.EnvironmentConfigs[i]
		envRedis = append(envRedis, &redis.ConfigWithName{Name: env.Name, Cfg: &env.RedisCfg})
	}
	if err := redis.Init(&cfg.RedisCfg, envRedis...); err != nil {
		return fmt.Errorf("failed to initialize Redis: %w", err)
	}
	log.Info("Redis connection initialized successfully")

	// Initialize database
	if err := db.Init(&cfg.DbCfg); err != nil {
		return fmt.Errorf("failed to initialize database: %w", err)
	}
	log.Info("Database connection initialized successfully")

	// Initialize gRPC clients: GLOBAL aliases normal; trial and other envs use named contexts.
	normalEnv, ok := cfg.EnvironmentByName(config.ServerTypeNormal)
	if !ok {
		return fmt.Errorf("environment %q is required", config.ServerTypeNormal)
	}
	if err := client.InitGlobalClients(normalEnv.RoomServerAddress, normalEnv.LobbyServerAddress, normalEnv.LedgerServerAddress); err != nil {
		return fmt.Errorf("failed to initialize gRPC clients: %w", err)
	}
	if err := client.InitClientContext(config.ServerTypeNormal, normalEnv.RoomServerAddress, normalEnv.LobbyServerAddress, normalEnv.LedgerServerAddress); err != nil {
		return fmt.Errorf("failed to initialize gRPC clients for environment %q: %w", config.ServerTypeNormal, err)
	}
	for i := range cfg.EnvironmentConfigs {
		env := cfg.EnvironmentConfigs[i]
		if env.Name == config.ServerTypeNormal {
			continue
		}
		if err := client.InitClientContext(env.Name, env.RoomServerAddress, env.LobbyServerAddress, env.LedgerServerAddress); err != nil {
			return fmt.Errorf("failed to initialize gRPC clients for environment %q: %w", env.Name, err)
		}
	}
	log.Info("gRPC clients initialized successfully")

	if err := api.StartTokenServerTypeListener(); err != nil {
		return fmt.Errorf("failed to start token server type listener: %w", err)
	}

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

	// 创建并启动调度器
	scheduler := cron.NewScheduler()
	scheduler.RegisterAllTasks()

	// 创建上下文用于优雅关闭
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// 启动调度器
	scheduler.Start(ctx)
	log.Info("任务调度器已启动（包含匹配任务）")

	// Create and start server
	log.Infof("Starting BeastRoyale backend server on port: %d", cfg.ServerCfg.Port)
	svr := server.New(&cfg.ServerCfg, sessionStore, redisCache)
	svr.Run()

	// Wait for interrupt signal
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
	<-sigs

	log.Info("Received shutdown signal, closing server...")

	api.StopTokenServerTypeListener()

	// 取消上下文，停止调度器
	cancel()

	// Close gRPC clients
	if err := client.CloseGlobalClients(); err != nil {
		log.Errorf("Error occurred while closing gRPC clients: %v", err)
	} else {
		log.Info("gRPC clients closed successfully")
	}

	// Gracefully shutdown server
	if err := svr.Stop(); err != nil {
		log.Errorf("Error occurred while closing server: %v", err)
		return err
	}

	log.Info("Server closed successfully")
	return nil
}
