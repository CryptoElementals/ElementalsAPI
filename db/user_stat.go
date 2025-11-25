package db

import (
	"errors"
	"fmt"
	"strconv"
	"strings"

	dao "github.com/CryptoElementals/common/models"
	"github.com/CryptoElementals/common/rpc/proto"
	"gorm.io/gorm"
)

// UpdateUserStatByPlayerIds 批量原子操作：增量更新多个用户统计数据
func UpdateUserStatByPlayerIds(playerIDs []int64) ([]*dao.UserStat, error) {
	if len(playerIDs) == 0 {
		return []*dao.UserStat{}, nil
	}

	// 去重地址
	uniquePlayerIDs := make(map[int64]bool)
	var deduplicatedPlayerIDs []int64
	for _, playerID := range playerIDs {
		if !uniquePlayerIDs[playerID] {
			uniquePlayerIDs[playerID] = true
			deduplicatedPlayerIDs = append(deduplicatedPlayerIDs, playerID)
		}
	}

	// 使用事务确保原子性
	var results []*dao.UserStat
	err := Get().Transaction(func(tx *gorm.DB) error {
		results = make([]*dao.UserStat, 0, len(deduplicatedPlayerIDs))

		for _, playerID := range deduplicatedPlayerIDs {
			// 在事务内查询用户档案
			var profile dao.UserProfile
			if err := tx.Where("player_id = ?", playerID).First(&profile).Error; err != nil {
				// 跳过不存在的地址
				if errors.Is(err, gorm.ErrRecordNotFound) {
					continue
				}
				return fmt.Errorf("failed to get user profile for playerID %d: %v", playerID, err)
			}

			// 步骤1：查找或创建用户统计记录
			userStat := &dao.UserStat{}
			dbResult := tx.Where("player_id = ?", profile.PlayerID).First(userStat)

			if dbResult.Error != nil {
				if errors.Is(dbResult.Error, gorm.ErrRecordNotFound) {
					// 如果记录不存在，创建新记录
					userStat = &dao.UserStat{
						PlayerID:           profile.PlayerID,
						TotalGameCount:     0,
						WinCount:           0,
						LoseCount:          0,
						TieCount:           0,
						LastPlayerRewardID: 0,
					}

					if err := tx.Create(userStat).Error; err != nil {
						return fmt.Errorf("failed to create user stat for player_id %d: %v", profile.PlayerID, err)
					}
				} else {
					return fmt.Errorf("failed to query user stat for player_id %d: %v", profile.PlayerID, dbResult.Error)
				}
			}

			// 步骤2：查找增量奖励记录
			var newRewards []dao.PlayerReward
			if err := tx.Where("player_id = ? AND id > ?", playerID, userStat.LastPlayerRewardID).
				Order("id ASC").
				Find(&newRewards).Error; err != nil {
				return fmt.Errorf("failed to query new rewards for player_id %d: %v", playerID, err)
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
				return fmt.Errorf("failed to update user stat for player_id %d: %v", profile.PlayerID, err)
			}

			// 步骤5：重新查询更新后的记录
			if err := tx.Where("player_id = ?", profile.PlayerID).First(userStat).Error; err != nil {
				return fmt.Errorf("failed to query updated user stat for player_id %d: %v", profile.PlayerID, err)
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

// GetUserStatByPlayerID 根据 player_id 获取用户统计数据
func GetUserStatByPlayerID(playerID string) (*dao.UserStat, error) {
	userStat := &dao.UserStat{}
	if strings.TrimSpace(playerID) == "" {
		return userStat, fmt.Errorf("playerID cannot be empty")
	}
	id, err := strconv.ParseUint(playerID, 10, 64)
	if err != nil {
		return nil, err
	}
	if err := Get().Where("player_id = ?", id).First(userStat).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return userStat, nil
		}
		return nil, err
	}
	return userStat, nil
}
