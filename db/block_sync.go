package db

import (
	"errors"

	dao "github.com/CryptoElementals/common/models"
	"gorm.io/gorm"
)

func FindBlockSyncs() ([]dao.BlockSync, error) {
	var blockSync []dao.BlockSync
	err := db.Table("block_sync").Find(&blockSync).Error
	return blockSync, err
}

func FindBlockSync(chainID uint64, syncType string) (*dao.BlockSync, error) {
	var blockSync dao.BlockSync
	err := db.Where("chain_id = ? AND type = ?", chainID, syncType).First(&blockSync).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &blockSync, nil
}

func SaveBlockSyncs(blockSyncs []dao.BlockSync) error {
	// Use Save method: if the record exists, update it; if not, insert it
	if err := db.Save(&blockSyncs).Error; err != nil {
		return err // Return error if save fails
	}

	return nil // Return after update or insert
}

func SaveBlockSync(blockSync dao.BlockSync) error {
	var existingSync dao.BlockSync
	err := db.Where("chain_id = ? AND type = ?", blockSync.ChainID, blockSync.Type).First(&existingSync).Error

	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return err // Return error if query fails
	}

	if errors.Is(err, gorm.ErrRecordNotFound) {
		// If not found, insert a new record
		if err := db.Create(&blockSync).Error; err != nil {
			return err // Return error if insert fails
		}
	} else {
		// If found, update the existing record
		existingSync.BlockHeight = blockSync.BlockHeight // Update Slot value
		existingSync.Type = blockSync.Type               // Update Type value (if needed)
		if err := db.Save(&existingSync).Error; err != nil {
			return err // Return error if update fails
		}
	}

	return nil // Return success
}
