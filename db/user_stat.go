package db

import (
	"errors"
	"fmt"
	"strings"

	dao "github.com/CryptoElementals/common/models"
	"github.com/CryptoElementals/common/rpc/proto"
	"gorm.io/gorm"
)

// UpdateUserStatByAddresses 批量原子操作：增量更新多个用户统计数据
func UpdateUserStatByAddresses(addresses []string) ([]*dao.UserStat, error) {
	if len(addresses) == 0 {
		return []*dao.UserStat{}, nil
	}

	// 去重地址
	uniqueAddresses := make(map[string]bool)
	var deduplicatedAddresses []string
	for _, addr := range addresses {
		if !uniqueAddresses[addr] {
			uniqueAddresses[addr] = true
			deduplicatedAddresses = append(deduplicatedAddresses, addr)
		}
	}

	// 使用事务确保原子性
	var results []*dao.UserStat
	err := Get().Transaction(func(tx *gorm.DB) error {
		results = make([]*dao.UserStat, 0, len(deduplicatedAddresses))

		for _, address := range deduplicatedAddresses {
			address = strings.ToLower(address)
			// 解析 user_id
			profile, err := GetUserProfileByAddress(address)
			if err != nil {
				// 跳过不存在的地址
				if errors.Is(err, gorm.ErrRecordNotFound) {
					continue
				}
				return fmt.Errorf("failed to get user profile for address %s: %v", address, err)
			}

			// 步骤1：查找或创建用户统计记录
			userStat := &dao.UserStat{}
			dbResult := tx.Where("user_id = ?", profile.UserID).First(userStat)

			if dbResult.Error != nil {
				if errors.Is(dbResult.Error, gorm.ErrRecordNotFound) {
					// 如果记录不存在，创建新记录
					userStat = &dao.UserStat{
						UserID:             profile.UserID,
						TotalGameCount:     0,
						WinCount:           0,
						LoseCount:          0,
						TieCount:           0,
						LastPlayerRewardID: 0,
					}

					if err := tx.Create(userStat).Error; err != nil {
						return fmt.Errorf("failed to create user stat for user_id %s: %v", profile.UserID.String(), err)
					}
				} else {
					return fmt.Errorf("failed to query user stat for user_id %s: %v", profile.UserID.String(), dbResult.Error)
				}
			}

			// 步骤2：查找增量奖励记录
			var newRewards []dao.PlayerReward
			if err := tx.Where("wallet_address = ? AND id > ?", address, userStat.LastPlayerRewardID).
				Order("id ASC").
				Find(&newRewards).Error; err != nil {
				return fmt.Errorf("failed to query new rewards for address %s: %v", address, err)
			}

			// 如果没有新的奖励记录，直接添加到结果中
			if len(newRewards) == 0 {
				results = append(results, userStat)
				continue
			}

			// 步骤3：计算增量统计数据
			var totalGameCount, winCount, loseCount, tieCount uint
			var maxRewardID uint

			for _, reward := range newRewards {
				// 更新最大奖励ID
				if reward.ID > maxRewardID {
					maxRewardID = reward.ID
				}

				// 根据游戏结果状态统计
				switch reward.PlayerGameResultStatus {
				case proto.PlayerGameResultStatus_PLAYER_WIN:
					winCount++
				case proto.PlayerGameResultStatus_PLAYER_LOSE:
					loseCount++
				case proto.PlayerGameResultStatus_PLAYER_TIE:
					tieCount++
				}

				// 每有一条奖励记录代表一场游戏
				totalGameCount++
			}

			// 步骤4：原子更新用户统计
			updateData := map[string]interface{}{
				"total_game_count":      userStat.TotalGameCount + totalGameCount,
				"win_count":             userStat.WinCount + winCount,
				"lose_count":            userStat.LoseCount + loseCount,
				"tie_count":             userStat.TieCount + tieCount,
				"last_player_reward_id": maxRewardID,
			}

			if err := tx.Model(userStat).Updates(updateData).Error; err != nil {
				return fmt.Errorf("failed to update user stat for user_id %s: %v", profile.UserID.String(), err)
			}

			// 步骤5：重新查询更新后的记录
			if err := tx.Where("user_id = ?", profile.UserID).First(userStat).Error; err != nil {
				return fmt.Errorf("failed to query updated user stat for user_id %s: %v", profile.UserID.String(), err)
			}

			results = append(results, userStat)
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	return results, nil
}

// GetUserStatByAddress 根据地址获取用户统计数据
func GetUserStatByAddress(address string) (*dao.UserStat, error) {
	if address == "" {
		return nil, fmt.Errorf("address cannot be empty")
	}

	userStat := &dao.UserStat{}

	// 解析 user_id
	profile, err := GetUserProfileByAddress(strings.ToLower(address))
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			// 不存在用户时返回空对象
			return userStat, nil
		}
		return nil, fmt.Errorf("failed to query user profile for address %s: %v", address, err)
	}

	// 查询用户统计记录（按 user_id）
	if err := Get().Where("user_id = ?", profile.UserID).First(userStat).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			// 如果记录不存在，返回空对象
			return userStat, nil
		}
		// 其他数据库错误
		return nil, fmt.Errorf("failed to query user stat for user_id %s: %v", profile.UserID.String(), err)
	}

	return userStat, nil
}
