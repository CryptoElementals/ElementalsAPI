package db

import (
	dao "github.com/CryptoElementals/common/models"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// GetDevTempKeyByAddress 查用户已绑定的临时密钥
func GetDevTempKeyByAddress(address string) (*dao.DevTempKey, error) {
	var rec dao.DevTempKey
	err := Get().Where("address = ?", address).First(&rec).Error
	if err != nil {
		return nil, err
	}
	return &rec, nil
}

// AssignNextAvailableDevTempKey 为给定 address 分配一个 address 为空的临时密钥，并更新 address 字段
func AssignNextAvailableDevTempKey(address string) (*dao.DevTempKey, error) {
	var out *dao.DevTempKey
	err := Get().Transaction(func(tx *gorm.DB) error {
		var rec dao.DevTempKey
		// 加锁取一条空闲记录
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("address = '' OR address IS NULL").First(&rec).Error; err != nil {
			return err
		}
		// 绑定给该 address
		rec.Address = address
		if err := tx.Save(&rec).Error; err != nil {
			return err
		}
		out = &rec
		return nil
	})
	return out, err
}
