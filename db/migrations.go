package db

import (
	"fmt"

	dao "github.com/CryptoElementals/common/models"
)

func Migrate() error {
	migrates := []any{
		&dao.UserProfile{},
		&dao.CardStat{},
		&dao.GameArgs{},
		&dao.Game{},
		&dao.Turn{},
		&dao.PlayerTurnInfo{},
		&dao.GamePlayerInfo{},
		&dao.PlayerReward{},
		&dao.BattleReward{},
		&dao.GameResult{},
		&dao.Room{},
		&dao.Card{},
		&dao.BlockSync{},
		&dao.UserToken{},
		&dao.LockedUserToken{},
		&dao.DevTempKey{},
		&dao.UserStat{},
		&dao.TournamentSchedule{},
		&dao.Tournament{},
		&dao.TournamentEntry{},
		&dao.TournamentMatch{},
		&dao.GameMatch{},
	}
	err := Get().Set("gorm:table_options", "ENGINE=InnoDB").AutoMigrate(migrates...)
	if err != nil {
		return err
	}
	return nil
}

func MigrateMemDb() error {
	var migrates = []any{
		&dao.UserProfile{},
		&dao.CardStat{},
		&dao.GameArgs{},
		&dao.Game{},
		&dao.Turn{},
		&dao.PlayerTurnInfo{},
		&dao.GamePlayerInfo{},
		&dao.PlayerReward{},
		&dao.BattleReward{},
		&dao.GameResult{},
		&dao.Room{},
		&dao.Card{},
		&dao.BlockSync{},
		&dao.UserToken{},
		&dao.LockedUserToken{},
		&dao.DevTempKey{},
		&dao.TournamentSchedule{},
		&dao.Tournament{},
		&dao.TournamentEntry{},
		&dao.TournamentMatch{},
		&dao.GameMatch{},
	}
	err := Get().AutoMigrate(migrates...)
	if err != nil {
		return err
	}
	for _, table := range migrates {
		exist := Get().Migrator().HasTable(table)
		if !exist {
			return fmt.Errorf("table not found: %T", table)
		}
	}
	return nil
}
