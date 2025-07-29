package db

import (
	"fmt"

	dao "github.com/CryptoElementals/common/models"
)

func Migrate() error {
	migrates := []any{
		&dao.UserProfile{},
		&dao.CardStat{},
		&dao.Game{},
		&dao.Round{},
		&dao.PlayerRoundInfo{},
		&dao.RoundSubmittedCard{},
		&dao.CardEffect{},
		&dao.GamePlayerInfo{},
		&dao.PlayerReward{},
		&dao.BattleReward{},
		&dao.GameResult{},
		&dao.Room{},
		&dao.Card{},
		&dao.LockToken{},
		&dao.CardsOnChainTx{},
		&dao.CommitmentOnChainTx{},
		&dao.CreateRoomTx{},
		&dao.SetRoundReadyTx{},
		&dao.BlockSync{},
		&dao.UserToken{},
		&dao.LockedUserToken{},
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
		&dao.Game{},
		&dao.Round{},
		&dao.PlayerRoundInfo{},
		&dao.RoundSubmittedCard{},
		&dao.CardEffect{},
		&dao.GamePlayerInfo{},
		&dao.PlayerReward{},
		&dao.BattleReward{},
		&dao.GameResult{},
		&dao.Room{},
		&dao.Card{},
		&dao.LockToken{},
		&dao.CardsOnChainTx{},
		&dao.CommitmentOnChainTx{},
		&dao.CreateRoomTx{},
		&dao.SetRoundReadyTx{},
		&dao.BlockSync{},
		&dao.UserToken{},
		&dao.LockedUserToken{},
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
