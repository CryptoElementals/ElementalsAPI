package main

import (
	"database/sql"
	"fmt"
	"os"

	"github.com/CryptoElementals/common/config"
	"github.com/CryptoElementals/common/db"
	"github.com/CryptoElementals/common/log"
	_ "github.com/go-sql-driver/mysql"
	"github.com/spf13/cobra"
)

var (
	configPath string
)

// createDbCmd represents the create-db command
var createDbCmd = &cobra.Command{
	Use:   "create-db",
	Short: "Create the database if it doesn't exist",
	Run: func(cmd *cobra.Command, args []string) {
		cfg, err := loadDbConfig()
		if err != nil {
			fmt.Printf("Failed to load config: %v\n", err)
			os.Exit(1)
		}

		if cfg.Development {
			fmt.Println("Development mode uses in-memory SQLite, database creation is not needed")
			return
		}

		if err := createDatabase(cfg); err != nil {
			fmt.Printf("Failed to create database: %v\n", err)
			os.Exit(1)
		}

		fmt.Printf("Database '%s' created successfully\n", cfg.DbName)
	},
}

// migrateDbCmd represents the migrate-db command
var migrateDbCmd = &cobra.Command{
	Use:   "migrate-db",
	Short: "Run database migrations",
	Run: func(cmd *cobra.Command, args []string) {
		cfg, err := loadDbConfig()
		if err != nil {
			fmt.Printf("Failed to load config: %v\n", err)
			os.Exit(1)
		}

		// Initialize logger (minimal, just for db operations)
		logCfg := &log.Config{
			Level:       "info",
			Development: false,
		}
		if err := log.InitGlobalLogger(logCfg); err != nil {
			fmt.Printf("Failed to initialize logger: %v\n", err)
			os.Exit(1)
		}

		// Initialize database
		if err := db.Init(cfg); err != nil {
			fmt.Printf("Failed to initialize database: %v\n", err)
			os.Exit(1)
		}

		// Run migrations
		if cfg.Development {
			if err := db.MigrateMemDb(); err != nil {
				fmt.Printf("Failed to migrate database: %v\n", err)
				os.Exit(1)
			}
			fmt.Println("Database migration completed successfully (in-memory SQLite)")
		} else {
			if err := db.Migrate(); err != nil {
				fmt.Printf("Failed to migrate database: %v\n", err)
				os.Exit(1)
			}
			fmt.Println("Database migration completed successfully")
		}
	},
}

func init() {
	rootCmd.AddCommand(createDbCmd)
	rootCmd.AddCommand(migrateDbCmd)

	createDbCmd.Flags().StringVarP(&configPath, "config", "c", "", "config file path")
	createDbCmd.MarkFlagRequired("config")

	migrateDbCmd.Flags().StringVarP(&configPath, "config", "c", "", "config file path")
	migrateDbCmd.MarkFlagRequired("config")
}

// loadDbConfig loads database configuration from YAML file
func loadDbConfig() (*db.Config, error) {
	err := config.InitToolsConfig(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to load config: %w", err)
	}
	return &config.ToolsGConf.DbCfg, nil
}

// createDatabase creates the database if it doesn't exist
func createDatabase(cfg *db.Config) error {
	// Validate required fields
	if cfg.DbName == "" {
		return fmt.Errorf("database name is required")
	}
	if cfg.Endpoint == "" {
		return fmt.Errorf("database endpoint is required")
	}
	if cfg.User == "" {
		return fmt.Errorf("database user is required")
	}

	// Connect to MySQL without specifying the database
	dsn := fmt.Sprintf("%s:%s@tcp(%s)/?multiStatements=true&charset=utf8mb4&parseTime=true",
		cfg.User, cfg.Password, cfg.Endpoint)

	sqlDB, err := sql.Open("mysql", dsn)
	if err != nil {
		return fmt.Errorf("failed to connect to MySQL: %w", err)
	}
	defer sqlDB.Close()

	// Test the connection
	if err := sqlDB.Ping(); err != nil {
		return fmt.Errorf("failed to ping MySQL server: %w", err)
	}

	// Check if database exists
	var exists int
	query := "SELECT 1 FROM INFORMATION_SCHEMA.SCHEMATA WHERE SCHEMA_NAME = ?"
	err = sqlDB.QueryRow(query, cfg.DbName).Scan(&exists)
	if err == nil {
		// Database already exists
		fmt.Printf("Database '%s' already exists\n", cfg.DbName)
		return nil
	}
	if err != sql.ErrNoRows {
		return fmt.Errorf("failed to check if database exists: %w", err)
	}

	// Create database (using backticks to escape database name)
	// Note: CREATE DATABASE doesn't support parameterized queries, so we use backticks for escaping
	createQuery := fmt.Sprintf("CREATE DATABASE IF NOT EXISTS `%s` CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci", cfg.DbName)
	_, err = sqlDB.Exec(createQuery)
	if err != nil {
		return fmt.Errorf("failed to create database: %w", err)
	}

	return nil
}
