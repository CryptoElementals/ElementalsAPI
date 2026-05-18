package db

import (
	"errors"
	"strconv"
	"strings"
	"time"

	cmnErrors "github.com/CryptoElementals/common/errors"
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

// GetUserProfileByPlayerID 根据玩家ID获取用户档案
func GetUserProfileByPlayerID(playerID string) (*dao.UserProfile, error) {
	return GetUserProfileByPlayerIDWithDB(playerID, Get())
}

// GetUserProfileByPlayerIDWithDB loads a profile using the given DB session (e.g. transaction)
// so sqlite dev mode with a single pooled connection does not deadlock inside Transaction callbacks.
func GetUserProfileByPlayerIDWithDB(playerID string, gdb *gorm.DB) (*dao.UserProfile, error) {
	var userProfile dao.UserProfile
	id, err := strconv.ParseUint(playerID, 10, 64)
	if err != nil {
		return nil, err
	}
	if err := gdb.Where("player_id = ?", id).First(&userProfile).Error; err != nil {
		return nil, err
	}
	return &userProfile, nil
}

// GetUserProfileByPlayerID 根据玩家ID获取用户档案
func GetUserProfileByPlayerIDInt(playerID int64) (*dao.UserProfile, error) {
	var userProfile dao.UserProfile
	if err := Get().Where("player_id = ?", playerID).First(&userProfile).Error; err != nil {
		return nil, err
	}
	return &userProfile, nil
}

// EffectiveServerType returns the profile server type or trial when unset.
func EffectiveServerType(profile *dao.UserProfile) string {
	if profile == nil {
		return dao.ServerTypeTrial
	}
	return dao.NormalizeServerType(profile.ServerType)
}

// CreateUserProfile 创建用户档案
func CreateUserProfile(userProfile *dao.UserProfile) error {
	if userProfile != nil {
		userProfile.ServerType = dao.NormalizeServerType(userProfile.ServerType)
	}
	return Get().Create(userProfile).Error
}

// UpdateUserProfile 更新用户档案
func UpdateUserProfile(userProfile *dao.UserProfile) error {
	err := Get().Save(userProfile).Error
	if err != nil {
		// 检查是否是唯一性约束错误（用户名重复）
		if isDuplicateEntryError(err) {
			return cmnErrors.UserNameDuplicate(userProfile.Name)
		}
	}
	return err
}

// isDuplicateEntryError 检查是否是 MySQL 唯一性约束错误
func isDuplicateEntryError(err error) bool {
	if err == nil {
		return false
	}
	errStr := strings.ToLower(err.Error())
	// MySQL 唯一性约束错误通常包含 "duplicate entry" 或错误码 1062
	return strings.Contains(errStr, "duplicate entry") ||
		strings.Contains(errStr, "1062") ||
		strings.Contains(errStr, "unique constraint")
}

// GetOrCreateUserProfile 获取或创建用户档案
func GetOrCreateUserProfile(address string) (*dao.UserProfile, error) {
	var userProfile dao.UserProfile
	err := Get().Where("address = ?", address).First(&userProfile).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			err = Get().Transaction(func(tx *gorm.DB) error {
				userProfile = dao.UserProfile{
					Address:    strings.ToLower(address),
					ServerType: dao.DefaultServerTypeForNewUser,
				}
				// 手动触发 BeforeCreate hook 来生成 PlayerID（传入 DB 实例）
				if err = userProfile.BeforeCreate(tx); err != nil {
					return err
				}
				// 直接使用生成的 PlayerID 设置 Name
				userProfile.Name = strconv.FormatInt(userProfile.PlayerID, 10)
				if err = tx.Create(&userProfile).Error; err != nil {
					return err
				}

				return nil
			})
			if err != nil {
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
			err = Get().Transaction(func(tx *gorm.DB) error {
				userProfile = dao.UserProfile{
					Email:      email,
					ServerType: dao.DefaultServerTypeForNewUser,
				}
				// 手动触发 BeforeCreate hook 来生成 PlayerID（传入 DB 实例）
				if err = userProfile.BeforeCreate(tx); err != nil {
					return err
				}
				// 统一使用 player_id 作为默认 name
				userProfile.Name = strconv.FormatInt(userProfile.PlayerID, 10)
				log.Infof("GetOrCreateUserProfileByEmail: userProfile: %+v", userProfile)
				if err = tx.Create(&userProfile).Error; err != nil {
					log.Infof("GetOrCreateUserProfileByEmail: err: %v", err)
					return err
				}
				return nil
			})
			if err != nil {
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

// HasCollectedDailyRewardByPlayerID 检查用户（按 player_id）是否已领取今日奖励
// 使用UTC时间统一判断，确保全球用户使用相同的"今天"标准
func HasCollectedDailyRewardByPlayerID(playerID string) (bool, error) {
	profile, err := GetUserProfileByPlayerID(playerID)
	if err != nil {
		return false, err
	}
	if profile.CollectedRewardAt == nil {
		return false, nil
	}
	// 使用UTC时间统一判断"今天"
	now := time.Now().UTC()
	collectedTime := profile.CollectedRewardAt.UTC()
	return now.Year() == collectedTime.Year() &&
		now.YearDay() == collectedTime.YearDay(), nil
}

// UpdateDailyRewardCollectionByPlayerID 更新用户（按 player_id）每日奖励领取时间
func UpdateDailyRewardCollectionByPlayerID(playerID string) error {
	id, err := strconv.ParseUint(playerID, 10, 64)
	if err != nil {
		return err
	}
	now := time.Now()
	return Get().Model(&dao.UserProfile{}).
		Where("player_id = ?", id).
		Update("collected_reward_at", now).Error
}

// HasCollectedNewUserRewardByPlayerID 检查用户（按 player_id）是否已领取新手奖励
func HasCollectedNewUserRewardByPlayerID(playerID string) (bool, error) {
	profile, err := GetUserProfileByPlayerID(playerID)
	if err != nil {
		return false, err
	}
	return profile.CollectedNewUserRewardAt != nil, nil
}

// MarkNewUserRewardCollectedByPlayerIDTx 原子标记新手奖励已领取。
// 当 marked=false 时表示该用户此前已领取（或用户不存在）。
func MarkNewUserRewardCollectedByPlayerIDTx(tx *gorm.DB, playerID string) (marked bool, err error) {
	id, err := strconv.ParseUint(playerID, 10, 64)
	if err != nil {
		return false, err
	}
	now := time.Now().UTC()
	res := tx.Model(&dao.UserProfile{}).
		Where("player_id = ? AND collected_new_user_reward_at IS NULL", id).
		Update("collected_new_user_reward_at", now)
	if res.Error != nil {
		return false, res.Error
	}
	return res.RowsAffected > 0, nil
}
