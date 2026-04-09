package main

import (
	"fmt"
	"os"

	"github.com/CryptoElementals/common/cache"
	"github.com/CryptoElementals/common/config"
	"github.com/CryptoElementals/common/db"
	"github.com/CryptoElementals/common/log"
	"github.com/CryptoElementals/common/redis"
	"github.com/spf13/cobra"
)

var clearMatchmakingCmd = &cobra.Command{
	Use:   "clear-matchmaking",
	Short: "Clear room-server queue Redis keys and all locked_user_tokens rows",
	Long: `Deletes Redis keys under prefixes queue_info: and locked_token: (same as room server matchmaking),
and removes every row from the locked_user_tokens table.

Requires database config. Redis is optional: if redis.address is set in the same config file, cache keys are cleared;
otherwise only the database cleanup runs.`,
	Run: func(cmd *cobra.Command, args []string) {
		if err := config.InitToolsConfig(configPath); err != nil {
			fmt.Printf("Failed to load config: %v\n", err)
			os.Exit(1)
		}

		logCfg := &log.Config{Level: "info", Development: false}
		if err := log.InitGlobalLogger(logCfg); err != nil {
			fmt.Printf("Failed to initialize logger: %v\n", err)
			os.Exit(1)
		}

		if err := db.Init(&config.ToolsGConf.DbCfg); err != nil {
			fmt.Printf("Failed to initialize database: %v\n", err)
			os.Exit(1)
		}

		nLocked, err := db.DeleteAllLockedUserTokens()
		if err != nil {
			fmt.Printf("Failed to delete locked user tokens: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("Deleted %d locked_user_tokens row(s)\n", nLocked)

		rc := config.ToolsGConf.RedisCfg
		if rc.Address == "" {
			fmt.Println("Redis address not set in config; skipped queue cache clear (add redis.address to tools YAML to clear Redis)")
			return
		}
		if rc.Size == 0 {
			rc.Size = 10
		}
		if err := redis.Init(&rc); err != nil {
			fmt.Printf("Failed to connect to Redis: %v\n", err)
			os.Exit(1)
		}
		c, err := cache.NewRedisCache()
		if err != nil {
			fmt.Printf("Failed to create Redis cache: %v\n", err)
			os.Exit(1)
		}
		q, t, err := cache.ClearRoomServerQueueAndTokenKeys(c)
		if err != nil {
			fmt.Printf("Failed to clear Redis queue keys: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("Removed %d queue_info key(s) and %d locked_token key(s) from Redis\n", q, t)
	},
}

func init() {
	rootCmd.AddCommand(clearMatchmakingCmd)
	clearMatchmakingCmd.Flags().StringVarP(&configPath, "config", "c", "", "config file path")
	clearMatchmakingCmd.MarkFlagRequired("config")
}
