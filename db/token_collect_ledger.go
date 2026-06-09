package db

import (
	"errors"
	"strings"

	dao "github.com/CryptoElementals/common/models"
)

func InsertTokenCollectLedger(row *dao.TokenCollectLedger) (uint, error) {
	if row == nil {
		return 0, errors.New("nil token collect ledger row")
	}
	row.PlayerAddress = strings.ToLower(strings.TrimSpace(row.PlayerAddress))
	row.CollectorAddress = strings.ToLower(strings.TrimSpace(row.CollectorAddress))
	row.TxHash = strings.ToLower(strings.TrimSpace(row.TxHash))
	if err := Get().Create(row).Error; err != nil {
		return 0, err
	}
	return row.ID, nil
}
