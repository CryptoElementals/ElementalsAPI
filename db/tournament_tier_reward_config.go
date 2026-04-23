package db

import (
	"fmt"

	dao "github.com/CryptoElementals/common/models"
	"gorm.io/gorm/clause"
)

// SeedTournamentTierRewardConfigs inserts/updates tournament reward templates.
// Data is aligned with tournament design sheet for bracket sizes 64~8192.
func SeedTournamentTierRewardConfigs() error {
	return SeedTournamentTierRewardConfigsWithUpdate(false)
}

// SeedTournamentTierRewardConfigsWithUpdate seeds tournament reward templates.
// When allowUpdate is false, existing non-empty table must exactly match expected seed.
// When allowUpdate is true, existing rows are upserted to expected values.
func SeedTournamentTierRewardConfigsWithUpdate(allowUpdate bool) error {
	db := Get()
	if db == nil {
		return fmt.Errorf("db is nil")
	}
	rows := buildTournamentTierRewardConfigs()
	if len(rows) == 0 {
		return fmt.Errorf("tournament tier reward seed rows is empty")
	}

	var count int64
	if err := db.Model(&dao.TournamentTierRewardConfig{}).Count(&count).Error; err != nil {
		return err
	}
	if count > 0 {
		var existing []dao.TournamentTierRewardConfig
		if err := db.
			Order("total_player_count asc, entry_fee asc, tier_no asc").
			Find(&existing).Error; err != nil {
			return err
		}
		if isSameTournamentTierRewardConfigs(existing, rows) {
			return nil
		}
		if !allowUpdate {
			return fmt.Errorf("tournament_tier_reward_configs already has data and does not match expected seed")
		}
		return db.Clauses(clause.OnConflict{
			Columns: []clause.Column{
				{Name: "total_player_count"},
				{Name: "entry_fee"},
				{Name: "tier_no"},
			},
			DoUpdates: clause.AssignmentColumns([]string{
				"commission",
				"total_tier_count",
				"reward_token",
				"point",
				"updated_at",
			}),
		}).Create(&rows).Error
	}
	return db.Clauses(clause.OnConflict{DoNothing: true}).Create(&rows).Error
}

func isSameTournamentTierRewardConfigs(a, b []dao.TournamentTierRewardConfig) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i].TotalPlayerCount != b[i].TotalPlayerCount ||
			a[i].EntryFee != b[i].EntryFee ||
			a[i].Commission != b[i].Commission ||
			a[i].TotalTierCount != b[i].TotalTierCount ||
			a[i].TierNo != b[i].TierNo ||
			a[i].RewardToken != b[i].RewardToken ||
			a[i].Point != b[i].Point {
			return false
		}
	}
	return true
}

func TournamentTierRewardConfigListByBracketSize(bracketSize int32) ([]dao.TournamentTierRewardConfig, error) {
	var out []dao.TournamentTierRewardConfig
	err := Get().Where("total_player_count = ?", bracketSize).Order("tier_no asc").Find(&out).Error
	return out, err
}

func buildTournamentTierRewardConfigs() []dao.TournamentTierRewardConfig {
	type tierRow struct {
		TierNo      int32
		RewardToken int32
		Point       int32
	}
	type tierGroup struct {
		EntryFee       int32
		Commission     int32
		TotalTierCount int32
		Tiers          []tierRow
	}

	grouped := map[int32]tierGroup{
		64: {
			EntryFee: 1000, Commission: 6400, TotalTierCount: 6,
			Tiers: []tierRow{{1, 0, 8}, {2, 0, 16}, {3, 2880, 36}, {4, 5760, 48}, {5, 11520, 60}, {6, 23040, 72}},
		},
		128: {
			EntryFee: 1000, Commission: 12800, TotalTierCount: 7,
			Tiers: []tierRow{{1, 0, 8}, {2, 0, 16}, {3, 2400, 36}, {4, 4800, 48}, {5, 9600, 60}, {6, 19200, 72}, {7, 38400, 84}},
		},
		256: {
			EntryFee: 1000, Commission: 25280, TotalTierCount: 8,
			Tiers: []tierRow{{1, 0, 8}, {2, 0, 16}, {3, 2060, 36}, {4, 4120, 48}, {5, 8240, 60}, {6, 16480, 72}, {7, 32960, 84}, {8, 65920, 96}},
		},
		512: {
			EntryFee: 1000, Commission: 51200, TotalTierCount: 9,
			Tiers: []tierRow{{1, 0, 8}, {2, 0, 16}, {3, 1800, 36}, {4, 3600, 48}, {5, 7200, 60}, {6, 14400, 72}, {7, 28800, 84}, {8, 57600, 96}, {9, 115200, 108}},
		},
		1024: {
			EntryFee: 1000, Commission: 102400, TotalTierCount: 10,
			Tiers: []tierRow{{1, 0, 8}, {2, 0, 16}, {3, 1600, 36}, {4, 3200, 48}, {5, 6400, 60}, {6, 12800, 72}, {7, 25600, 84}, {8, 51200, 96}, {9, 102400, 108}, {10, 204800, 120}},
		},
		2048: {
			EntryFee: 1000, Commission: 204800, TotalTierCount: 11,
			Tiers: []tierRow{{1, 0, 8}, {2, 0, 16}, {3, 1440, 36}, {4, 2880, 48}, {5, 5760, 60}, {6, 11520, 72}, {7, 23040, 84}, {8, 46080, 96}, {9, 92160, 108}, {10, 184320, 120}, {11, 368640, 132}},
		},
		4096: {
			EntryFee: 1000, Commission: 409856, TotalTierCount: 12,
			Tiers: []tierRow{{1, 0, 8}, {2, 0, 16}, {3, 1309, 36}, {4, 2618, 48}, {5, 5236, 60}, {6, 10472, 72}, {7, 20944, 84}, {8, 41888, 96}, {9, 83776, 108}, {10, 167552, 120}, {11, 335104, 132}, {12, 670208, 144}},
		},
		8192: {
			EntryFee: 1000, Commission: 819200, TotalTierCount: 13,
			Tiers: []tierRow{{1, 0, 8}, {2, 0, 16}, {3, 1200, 36}, {4, 2400, 48}, {5, 4800, 60}, {6, 9600, 72}, {7, 19200, 84}, {8, 38400, 96}, {9, 76800, 108}, {10, 153600, 120}, {11, 307200, 132}, {12, 614400, 144}, {13, 1228800, 156}},
		},
	}

	order := []int32{64, 128, 256, 512, 1024, 2048, 4096, 8192}
	rows := make([]dao.TournamentTierRewardConfig, 0, 76)
	for _, totalPlayerCount := range order {
		g, ok := grouped[totalPlayerCount]
		if !ok {
			continue
		}
		for _, t := range g.Tiers {
			rows = append(rows, dao.TournamentTierRewardConfig{
				TotalPlayerCount: totalPlayerCount,
				EntryFee:         g.EntryFee,
				Commission:       g.Commission,
				TotalTierCount:   g.TotalTierCount,
				TierNo:           t.TierNo,
				RewardToken:      t.RewardToken,
				Point:            t.Point,
			})
		}
	}
	return rows
}
