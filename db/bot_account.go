package db

import (
	"strings"

	dao "github.com/CryptoElementals/common/models"
)

func EnsureBotAccountTable() error {
	return Get().AutoMigrate(&dao.BotAccount{})
}

func ListBotAccounts(limit int) ([]dao.BotAccount, error) {
	q := Get().Model(&dao.BotAccount{}).Order("id ASC")
	if limit > 0 {
		q = q.Limit(limit)
	}
	var accounts []dao.BotAccount
	if err := q.Find(&accounts).Error; err != nil {
		return nil, err
	}
	return accounts, nil
}

func InsertBotAccount(account *dao.BotAccount) error {
	if account == nil {
		return nil
	}
	account.TempAddress = strings.ToLower(account.TempAddress)
	return Get().Create(account).Error
}
