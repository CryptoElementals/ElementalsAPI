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
		&dao.BattleRewardPVP{},
		&dao.GameResult{},
		&dao.PlayerResultInfo{},
		&dao.Room{},
		&dao.Card{},
		&dao.BlockSync{},
		&dao.UserToken{},
		&dao.LockedUserToken{},
		&dao.DevTempKey{},
		&dao.UserStat{},
		&dao.Tournament{},
		&dao.TournamentParticipant{},
		&dao.TournamentRound{},
		&dao.TournamentMatch{},
		&dao.TournamentReward{},
		&dao.TournamentEntryLedger{},
		&dao.TournamentTierRewardConfig{},
		&dao.GameMatch{},
		&dao.PlayerQueueEntry{},
		&dao.PlayerGameEntry{},
		&dao.BotAccount{},
		&dao.GameChainID{},
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
		&dao.BattleRewardPVP{},
		&dao.GameResult{},
		&dao.PlayerResultInfo{},
		&dao.Room{},
		&dao.Card{},
		&dao.BlockSync{},
		&dao.UserToken{},
		&dao.LockedUserToken{},
		&dao.DevTempKey{},
		&dao.UserStat{},
		&dao.Tournament{},
		&dao.TournamentParticipant{},
		&dao.TournamentRound{},
		&dao.TournamentMatch{},
		&dao.TournamentReward{},
		&dao.TournamentEntryLedger{},
		&dao.TournamentTierRewardConfig{},
		&dao.GameMatch{},
		&dao.PlayerQueueEntry{},
		&dao.PlayerGameEntry{},
		&dao.BotAccount{},
		&dao.GameChainID{},
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
	if err := SeedTournamentTierRewardConfigs(); err != nil {
		return err
	}
	return nil
}
