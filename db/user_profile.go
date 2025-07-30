package db

import (
	"context"
	"errors"
	"time"

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

// UpdateDailyRewardCollection 更新用户每日奖励领取时间
func UpdateDailyRewardCollection(address string) error {
	now := time.Now()
	return Get().Model(&dao.UserProfile{}).
		Where("address = ?", address).
		Update("collected_reward_at", now).Error
}

// UpdateUserGameStats 更新用户游戏统计数据
// winner: 获胜者地址，如果是平局则为""
// multiplier: 赢家的最高倍率
func UpdateUserGameStats(player1Address, player2Address, winner string, multiplier float64) error {
	// 获取两个用户的档案
	player1Profile, err := GetUserProfileByAddress(player1Address)
	if err != nil {
		return err
	}

	player2Profile, err := GetUserProfileByAddress(player2Address)
	if err != nil {
		return err
	}

	// 计算积分和代币变化
	basePoints := 1000
	baseTokens := 1000

	// 更新用户档案中的对局统计信息
	player1Profile.OverallGame++
	player2Profile.OverallGame++

	// 获取 / 创建用户 Token 信息
	player1Token, err := GetPlayerTokenSimple(context.Background(), player1Address)
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return err
	}
	if player1Token == nil {
		player1Token = &dao.UserToken{WalletAddress: player1Address}
	}

	player2Token, err := GetPlayerTokenSimple(context.Background(), player2Address)
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return err
	}
	if player2Token == nil {
		player2Token = &dao.UserToken{WalletAddress: player2Address}
	}

	// 玩家1、玩家2基础积分变动
	player1PointsChange := basePoints * int(multiplier)
	player2PointsChange := basePoints * int(multiplier)

	player1Token.Points += int32(player1PointsChange)
	player2Token.Points += int32(player2PointsChange)

	// 根据胜负情况更新胜场数和代币
	if winner == player1Address {
		// 玩家1获胜
		player1Profile.WinCount++
		player1Token.TokenAmount += int32(float64(baseTokens) * multiplier * 0.98) // 赢家获得98%
		player2Token.TokenAmount -= int32(float64(baseTokens) * multiplier)        // 输家扣100%
	} else if winner == player2Address {
		// 玩家2获胜
		player2Profile.WinCount++
		player2Token.TokenAmount += int32(float64(baseTokens) * multiplier * 0.98)
		player1Token.TokenAmount -= int32(float64(baseTokens) * multiplier)
	} else {
		// 平局，双方各扣0.5%
		deduction := int32(float64(baseTokens) * 0.005)
		player1Token.TokenAmount -= deduction
		player2Token.TokenAmount -= deduction
	}

	// 重新计算胜率
	player1Profile.WinningRate = float64(player1Profile.WinCount) / float64(player1Profile.OverallGame)
	player2Profile.WinningRate = float64(player2Profile.WinCount) / float64(player2Profile.OverallGame)

	// 确保代币数量不为负数
	if player1Token.TokenAmount < 0 {
		player1Token.TokenAmount = 0
	}
	if player2Token.TokenAmount < 0 {
		player2Token.TokenAmount = 0
	}

	// 保存用户 Token 及 Profile
	err = SaveUserToken(*player1Token, *player2Token)
	if err != nil {
		return err
	}

	// 更新用户档案信息
	if err := UpdateUserProfile(player1Profile); err != nil {
		return err
	}
	if err := UpdateUserProfile(player2Profile); err != nil {
		return err
	}

	return nil
}
