package main

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/CryptoElementals/common/config"
	"github.com/CryptoElementals/common/db"
	"github.com/CryptoElementals/common/log"
	dao "github.com/CryptoElementals/common/models"
	"github.com/spf13/cobra"
)

var (
	chainTxPoolCleanupBefore  string
	chainTxPoolCleanupDays    int
	chainTxPoolCleanupExecute bool
	chainTxPoolCleanupBatch   int
	chainTxPoolCleanupSleep   time.Duration
)

var chainTxPoolCleanupCmd = &cobra.Command{
	Use:   "cleanup-tx-pool",
	Short: "Hard-delete soft-deleted chain_tx_pool_items before a cutoff time",
	Long: `Permanently deletes rows from chain_tx_pool_items that were previously soft-deleted
and have deleted_at earlier than the cutoff time.

By default this is a dry-run that only prints the count; pass --execute to actually delete.`,
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

		before, err := resolveCleanupCutoff(chainTxPoolCleanupBefore, chainTxPoolCleanupDays)
		if err != nil {
			fmt.Printf("Invalid cutoff options: %v\n", err)
			os.Exit(1)
		}

		if chainTxPoolCleanupBatch <= 0 {
			chainTxPoolCleanupBatch = 5000
		}

		q := db.Get().Unscoped().
			Model(&dao.ChainTxPoolItem{}).
			Where("deleted_at IS NOT NULL").
			Where("deleted_at < ?", before)

		var n int64
		if err := q.Count(&n).Error; err != nil {
			fmt.Printf("Failed to count rows: %v\n", err)
			os.Exit(1)
		}

		fmt.Printf("Found %d soft-deleted chain_tx_pool_items row(s) with deleted_at < %s\n", n, before.UTC().Format(time.RFC3339))
		if !chainTxPoolCleanupExecute {
			fmt.Println("Dry-run only. Re-run with --execute to delete them.")
			return
		}
		if n == 0 {
			return
		}

		type batchBoundary struct {
			LastID uint
		}
		var deleted int64
		for {
			batchScope := q.Select("id").Order("id").Limit(chainTxPoolCleanupBatch)
			var boundary batchBoundary
			err := db.Get().
				Table("(?) AS batch_ids", batchScope).
				Select("MAX(id) AS last_id").
				Scan(&boundary).Error
			if err != nil {
				fmt.Printf("Failed to resolve batch boundary id: %v\n", err)
				os.Exit(1)
			}
			if boundary.LastID == 0 {
				break
			}

			res := db.Get().Unscoped().
				Where("deleted_at IS NOT NULL").
				Where("deleted_at < ?", before).
				Where("id <= ?", boundary.LastID).
				Delete(&dao.ChainTxPoolItem{})
			if res.Error != nil {
				fmt.Printf("Failed to delete batch: %v\n", res.Error)
				os.Exit(1)
			}
			deleted += res.RowsAffected
			if chainTxPoolCleanupSleep > 0 {
				time.Sleep(chainTxPoolCleanupSleep)
			}
		}

		fmt.Printf("Deleted %d chain_tx_pool_items row(s)\n", deleted)
	},
}

func init() {
	rootCmd.AddCommand(chainTxPoolCleanupCmd)
	chainTxPoolCleanupCmd.Flags().StringVarP(&configPath, "config", "c", "", "config file path")
	chainTxPoolCleanupCmd.MarkFlagRequired("config")

	chainTxPoolCleanupCmd.Flags().StringVar(&chainTxPoolCleanupBefore, "before", "", "explicit cutoff time (RFC3339, '2006-01-02', '2006-01-02 15:04:05', or unix seconds)")
	chainTxPoolCleanupCmd.Flags().IntVar(&chainTxPoolCleanupDays, "days", -1, "delete records soft-deleted before UTC midnight of (today - days); use 0 for before today")

	chainTxPoolCleanupCmd.Flags().BoolVar(&chainTxPoolCleanupExecute, "execute", false, "actually delete rows (default: dry-run)")
	chainTxPoolCleanupCmd.Flags().IntVar(&chainTxPoolCleanupBatch, "batch", 5000, "batch size for deletes")
	chainTxPoolCleanupCmd.Flags().DurationVar(&chainTxPoolCleanupSleep, "sleep-interval", 0, "sleep duration between delete batches (e.g. 200ms, 1s)")
}

func parseFlexibleTime(s string) (time.Time, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return time.Time{}, fmt.Errorf("empty time")
	}

	// unix seconds
	if n, err := strconv.ParseInt(s, 10, 64); err == nil && len(s) >= 9 {
		return time.Unix(n, 0).UTC(), nil
	}

	if t, err := time.Parse(time.RFC3339, s); err == nil {
		return t.UTC(), nil
	}
	if t, err := time.ParseInLocation("2006-01-02 15:04:05", s, time.UTC); err == nil {
		return t.UTC(), nil
	}
	if t, err := time.ParseInLocation("2006-01-02", s, time.UTC); err == nil {
		return t.UTC(), nil
	}

	return time.Time{}, fmt.Errorf("unsupported time format %q", s)
}

func resolveCleanupCutoff(beforeRaw string, days int) (time.Time, error) {
	beforeRaw = strings.TrimSpace(beforeRaw)
	hasBefore := beforeRaw != ""
	hasDays := days >= 0

	if hasBefore && hasDays {
		return time.Time{}, fmt.Errorf("use either --before or --days, not both")
	}
	if !hasBefore && !hasDays {
		return time.Time{}, fmt.Errorf("one of --before or --days is required")
	}
	if hasBefore {
		return parseFlexibleTime(beforeRaw)
	}
	if days < 0 {
		return time.Time{}, fmt.Errorf("--days must be greater than or equal to 0")
	}

	nowUTC := time.Now().UTC()
	todayUTC := time.Date(nowUTC.Year(), nowUTC.Month(), nowUTC.Day(), 0, 0, 0, 0, time.UTC)
	return todayUTC.AddDate(0, 0, -days), nil
}

