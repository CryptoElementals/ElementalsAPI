package db

import (
	"time"

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

	// 玩家1的统计更新
	player1Profile.OverallGame++
	player1PointsChange := basePoints * int(multiplier)
	player1Profile.Points += player1PointsChange

	// 玩家2的统计更新
	player2Profile.OverallGame++
	player2PointsChange := basePoints * int(multiplier)
	player2Profile.Points += player2PointsChange

	// 根据胜负情况更新胜场数和代币
	if winner == player1Address {
		// 玩家1获胜
		player1Profile.WinCount++
		player1Profile.WinningRate = float64(player1Profile.WinCount) / float64(player1Profile.OverallGame)
		player1Profile.TokenAmount += int(float64(baseTokens) * multiplier * 0.98) // 赢家获得98%

		player2Profile.WinningRate = float64(player2Profile.WinCount) / float64(player2Profile.OverallGame)
		player2Profile.TokenAmount -= int(float64(baseTokens) * multiplier) // 输家扣除100%

	} else if winner == player2Address {
		// 玩家2获胜
		player2Profile.WinCount++
		player2Profile.WinningRate = float64(player2Profile.WinCount) / float64(player2Profile.OverallGame)
		player2Profile.TokenAmount += int(float64(baseTokens) * multiplier * 0.98) // 赢家获得98%

		player1Profile.WinningRate = float64(player1Profile.WinCount) / float64(player1Profile.OverallGame)
		player1Profile.TokenAmount -= int(float64(baseTokens) * multiplier) // 输家扣除100%

	} else {
		// 平局
		player1Profile.WinningRate = float64(player1Profile.WinCount) / float64(player1Profile.OverallGame)
		player1Profile.TokenAmount -= int(float64(baseTokens) * 0.005) // 平局双方都扣除0.5%

		player2Profile.WinningRate = float64(player2Profile.WinCount) / float64(player2Profile.OverallGame)
		player2Profile.TokenAmount -= int(float64(baseTokens) * 0.005) // 平局双方都扣除0.5%
	}

	// 确保代币数量不为负数，实际上不会小于0，因为限制门槛1000，最多扣800
	if player1Profile.TokenAmount < 0 {
		player1Profile.TokenAmount = 0
	}
	if player2Profile.TokenAmount < 0 {
		player2Profile.TokenAmount = 0
	}

	// 更新数据库
	err = UpdateUserProfile(player1Profile)
	if err != nil {
		return err
	}

	err = UpdateUserProfile(player2Profile)
	if err != nil {
		return err
	}

	return nil
}
