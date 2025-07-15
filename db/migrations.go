package db

import (
	dao "github.com/CryptoElementals/common/models"
)

func Migrate() error {
	migrates := []any{
		&dao.UserProfile{},
		&dao.CardStat{},
		&dao.GameInfo{},
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
