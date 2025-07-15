package db

import (
	dao "github.com/CryptoElementals/common/models"
)

// CreateLockToken 创建锁定代币记录
func CreateLockToken(lockToken *dao.LockToken) error {
	return Get().Create(lockToken).Error
}

// GetLockTokensByAddress 根据地址获取所有未删除的锁定代币记录
func GetLockTokensByAddress(address string) ([]dao.LockToken, error) {
	var lockTokens []dao.LockToken
	err := Get().Where("address = ? AND deleted_at IS NULL", address).Find(&lockTokens).Error
	return lockTokens, err
}

// GetLockTokenByAddressAndTempAddress 根据地址和临时地址获取特定的锁定代币记录
func GetLockTokenByAddressAndTempAddress(address, tempAddress string) (*dao.LockToken, error) {
	var lockToken dao.LockToken
	err := Get().Where("address = ? AND temp_address = ? AND deleted_at IS NULL", address, tempAddress).First(&lockToken).Error
	if err != nil {
		return nil, err
	}
	return &lockToken, nil
}

// GetTotalLockedTokensByAddress 获取指定地址的总锁定代币数量（不包括已删除的）
func GetTotalLockedTokensByAddress(address string) (int, error) {
	var total int
	err := Get().Model(&dao.LockToken{}).
		Where("address = ? AND deleted_at IS NULL", address).
		Select("COALESCE(SUM(token), 0)").
		Scan(&total).Error
	return total, err
}

// SoftDeleteLockToken 软删除锁定代币记录
func SoftDeleteLockToken(id uint) error {
	return Get().Delete(&dao.LockToken{}, id).Error
}

// SoftDeleteLockTokenByAddressAndTempAddress 根据地址和临时地址软删除锁定代币记录
func SoftDeleteLockTokenByAddressAndTempAddress(address, tempAddress string) error {
	return Get().Model(&dao.LockToken{}).
		Where("address = ? AND temp_address = ? AND deleted_at IS NULL", address, tempAddress).
		Update("deleted_at", Get().NowFunc()).Error
}

// HardDeleteLockToken 硬删除锁定代币记录
func HardDeleteLockToken(id uint) error {
	return Get().Unscoped().Delete(&dao.LockToken{}, id).Error
}

// GetLockTokenByID 根据ID获取锁定代币记录
func GetLockTokenByID(id uint) (*dao.LockToken, error) {
	var lockToken dao.LockToken
	err := Get().Where("id = ?", id).First(&lockToken).Error
	if err != nil {
		return nil, err
	}
	return &lockToken, nil
}
