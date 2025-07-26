package main

import (
	"fmt"
	"os"

	"github.com/CryptoElementals/common/config"
	"github.com/CryptoElementals/common/db"
	"github.com/CryptoElementals/common/log"
	"github.com/spf13/cobra"
)

// dbMigrateCmd represents the dbMigrate command
var dbMigrateCmd = &cobra.Command{
	Use:   "db-migrate",
	Short: "create tables in the target mysql db",
	Run: func(cmd *cobra.Command, args []string) {
		err := config.InitScannerConfig(configPath)
		if err != nil {
			fmt.Printf("load config failed: %+v", err)
			os.Exit(-1)
		}
		fmt.Printf("config: %+v\n", config.ScannerGConf)
		// Initialize logger
		if err := log.InitGlobalLogger(&config.ScannerGConf.LogCfg); err != nil {
			fmt.Printf("failed to initialize logger: %s\n", err.Error())
			return
		}
		log.Info("Logger system initialized successfully")

		// Initialize database
		if err := db.Init(&config.ScannerGConf.DbCfg); err != nil {
			log.Errorf("failed to initialize database: %s", err.Error())
			fmt.Printf("failed to initialize database: %s\n", err.Error())
			return
		}
		log.Info("Database connection initialized successfully")

		err = migrate()
		if err != nil {
			fmt.Printf("create db table failed: %+v", err)
			os.Exit(-1)
		}
	},
}

func init() {
	rootCmd.AddCommand(dbMigrateCmd)
	dbMigrateCmd.Flags().StringVarP(&configPath, "config", "c", "", "config file path")
	dbMigrateCmd.MarkFlagRequired("config")
}

func migrate() error {
	return db.Migrate()
}
