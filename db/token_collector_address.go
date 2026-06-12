package db

import (
	"errors"
	"strings"

	dao "github.com/CryptoElementals/common/models"
	"gorm.io/gorm"
)

func ListTokenCollectorAddresses(chainID uint64) ([]dao.TokenCollectorAddress, error) {
	var rows []dao.TokenCollectorAddress
	err := db.Where("chain_id = ?", chainID).Find(&rows).Error
	return rows, err
}

func UpsertTokenCollectorAddress(row dao.TokenCollectorAddress) (bool, error) {
	row.Address = strings.ToLower(row.Address)
	var existing dao.TokenCollectorAddress
	err := db.Where("chain_id = ? AND address = ?", row.ChainID, row.Address).First(&existing).Error
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return false, err
	}
	if errors.Is(err, gorm.ErrRecordNotFound) {
		if err := db.Create(&row).Error; err != nil {
			return false, err
		}
		return true, nil
	}
	return false, nil
}
