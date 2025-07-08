package cmd

import (
	"fmt"
	"os"

	"github.com/CryptoElementals/common/config"
	"github.com/CryptoElementals/common/db"
	"github.com/CryptoElementals/common/log"
	"github.com/spf13/cobra"
)

// dbMigrateCmd represents the db-migrate command
var dbMigrateCmd = &cobra.Command{
	Use:   "db-migrate",
	Short: "Apply all database migrations (AutoMigrate)",
	Run: func(cmd *cobra.Command, args []string) {
		if err := initApp(); err != nil {
			fmt.Printf("Failed to initialize application: %v\n", err)
			os.Exit(1)
		}
		log.Info("Starting database migration...")
		if err := db.Migrate(); err != nil {
			log.Fatalf("Failed to migrate database: %v", err)
		}
		log.Info("Database migration completed successfully")
	},
}

func init() {
	rootCmd.AddCommand(dbMigrateCmd)
	dbMigrateCmd.MarkFlagRequired("config")
}

// initApp initializes the application (logger, database, etc.)
func initApp() error {
	// Debug: 打印配置路径
	fmt.Printf("Config path: '%s'\n", configPath)

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

	// Initialize database
	if err := db.Init(&cfg.DbCfg); err != nil {
		return fmt.Errorf("failed to initialize database: %w", err)
	}

	return nil
}
