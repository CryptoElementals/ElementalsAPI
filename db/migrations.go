package db

import (
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
		&dao.GamePlayerInfo{},
		&dao.Room{},
		&dao.Card{},
		&dao.LockToken{},
	}
	err := Get().Set("gorm:table_options", "ENGINE=InnoDB").AutoMigrate(migrates...)
	if err != nil {
		return err
	}
	return nil
}

func MigrateMemDb() error {
	migrates := []any{
		&dao.UserProfile{},
		&dao.CardStat{},
		&dao.Game{},
		&dao.Round{},
		&dao.PlayerRoundInfo{},
		&dao.RoundSubmittedCard{},
		&dao.GamePlayerInfo{},
		&dao.Room{},
		// 以后有新表直接加在这里
	}
	err := Get().AutoMigrate(migrates...)
	if err != nil {
		return err
	}
	return nil
}
