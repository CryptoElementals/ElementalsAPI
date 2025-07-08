package db

import (
	dao "github.com/CryptoElementals/common/models"
)

// GetUserProfileByAddress 根据地址获取用户档案
func GetUserProfileByAddress(address string) (*dao.UserProfile, error) {
	var userProfile dao.UserProfile
	err := Get().Where("address = ?", address).First(&userProfile).Error
	if err != nil {
		return nil, err
	}
	return &userProfile, nil
}

// CreateUserProfile 创建用户档案
func CreateUserProfile(userProfile *dao.UserProfile) error {
	return Get().Create(userProfile).Error
}

// UpdateUserProfile 更新用户档案
func UpdateUserProfile(userProfile *dao.UserProfile) error {
	return Get().Save(userProfile).Error
}

// GetOrCreateUserProfile 获取或创建用户档案
func GetOrCreateUserProfile(address string) (*dao.UserProfile, error) {
	var userProfile dao.UserProfile
	err := Get().Where("address = ?", address).First(&userProfile).Error
	if err != nil {
		// 用户不存在，创建新用户档案
		userProfile = dao.UserProfile{
			Address: address,
			Name:    address, // 默认用户名就是地址
		}
		err = Get().Create(&userProfile).Error
		if err != nil {
			return nil, err
		}
	}
	return &userProfile, nil
}
