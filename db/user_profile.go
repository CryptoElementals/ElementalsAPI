package db

import (
	"errors"
	"strconv"
	"strings"
	"time"

	"github.com/CryptoElementals/common/log"
	dao "github.com/CryptoElementals/common/models"
	"gorm.io/gorm"
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

// Removed: GetUserProfileByEmail

// GetUserProfileByUserID 根据用户ID获取用户档案
func GetUserProfileByUserID(userID string) (*dao.UserProfile, error) {
	var userProfile dao.UserProfile
	// userID 存在于 session 中为字符串，这里解析为 uint64
	id, err := strconv.ParseUint(userID, 10, 64)
	if err != nil {
		return nil, err
	}
	if err := Get().Where("user_id = ?", id).First(&userProfile).Error; err != nil {
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
		if errors.Is(err, gorm.ErrRecordNotFound) {
			userProfile = dao.UserProfile{
				Address: strings.ToLower(address),
				Name:    strings.ToLower(address),
			}
			if err = Get().Create(&userProfile).Error; err != nil {
				return nil, err
			}
		} else {
			return nil, err
		}
	}
	return &userProfile, nil
}

// GetOrCreateUserProfileByEmail 根据邮箱获取或创建用户档案
func GetOrCreateUserProfileByEmail(email string, name string) (*dao.UserProfile, error) {
	var userProfile dao.UserProfile
	log.Infof("GetOrCreateUserProfileByEmail: email: %s, name: %s", email, name)
	log.Infof("GetOrCreateUserProfileByEmail: userProfile: %+v", userProfile)
	err := Get().Where("email = ?", email).First(&userProfile).Error
	log.Infof("GetOrCreateUserProfileByEmail: err: %v", err)
	if err != nil {
		log.Infof("GetOrCreateUserProfileByEmail: errors.Is(err, gorm.ErrRecordNotFound): %v", errors.Is(err, gorm.ErrRecordNotFound))
		if errors.Is(err, gorm.ErrRecordNotFound) {
			if name == "" {
				name = email
			}
			userProfile = dao.UserProfile{
				Email: email,
				Name:  name,
			}
			log.Infof("GetOrCreateUserProfileByEmail: userProfile: %+v", userProfile)
			if err = Get().Create(&userProfile).Error; err != nil {
				log.Infof("GetOrCreateUserProfileByEmail: err: %v", err)
				return nil, err
			}
		} else {
			return nil, err
		}
	}
	return &userProfile, nil
}

// HasCollectedDailyReward 检查用户是否已领取今日奖励
func HasCollectedDailyReward(address string) (bool, error) {
	userProfile, err := GetUserProfileByAddress(address)
	if err != nil {
		return false, err
	}

	// 如果从未领取过奖励，返回false
	if userProfile.CollectedRewardAt == nil {
		return false, nil
	}

	// 检查领取时间是否是今天
	now := time.Now()
	collectedTime := *userProfile.CollectedRewardAt

	// 比较年月日是否相同
	return now.Year() == collectedTime.Year() &&
		now.YearDay() == collectedTime.YearDay(), nil
}

// Removed: HasCollectedDailyRewardByEmail

// UpdateDailyRewardCollection 更新用户每日奖励领取时间
func UpdateDailyRewardCollection(address string) error {
	now := time.Now()
	return Get().Model(&dao.UserProfile{}).
		Where("address = ?", address).
		Update("collected_reward_at", now).Error
}

// Removed: UpdateDailyRewardCollectionByEmail

// HasCollectedDailyRewardByUserID 检查用户（按 user_id）是否已领取今日奖励
func HasCollectedDailyRewardByUserID(userID string) (bool, error) {
	profile, err := GetUserProfileByUserID(userID)
	if err != nil {
		return false, err
	}
	if profile.CollectedRewardAt == nil {
		return false, nil
	}
	now := time.Now()
	collectedTime := *profile.CollectedRewardAt
	return now.Year() == collectedTime.Year() &&
		now.YearDay() == collectedTime.YearDay(), nil
}

// UpdateDailyRewardCollectionByUserID 更新用户（按 user_id）每日奖励领取时间
func UpdateDailyRewardCollectionByUserID(userID string) error {
	id, err := strconv.ParseUint(userID, 10, 64)
	if err != nil {
		return err
	}
	now := time.Now()
	return Get().Model(&dao.UserProfile{}).
		Where("user_id = ?", id).
		Update("collected_reward_at", now).Error
}
