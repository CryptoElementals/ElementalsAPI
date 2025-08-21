package db

import (
	"errors"
	"fmt"
	"math"

	"github.com/CryptoElementals/common/log"

	dao "github.com/CryptoElementals/common/models"
	"github.com/CryptoElementals/common/rpc/proto"
	"gorm.io/gorm"
)

// CardStatInfo API响应用的卡牌统计信息结构
type CardStatInfo struct {
	CardID      uint    `json:"CardID"`
	Frequency   float64 `json:"Frequency"`
	WinningRate float64 `json:"WinningRate"`
}

// GetCardStatsByAddress 根据用户地址获取所有卡牌统计
func GetCardStatsByAddress(address string) ([]dao.CardStat, error) {
	var cardStats []dao.CardStat
	err := Get().Where("address = ?", address).Find(&cardStats).Error
	return cardStats, err
}

// GetCardStatByAddressAndName 根据用户地址和卡牌名称获取特定卡牌统计
func GetCardStatByAddressAndName(address, cardName string) (*dao.CardStat, error) {
	var cardStat dao.CardStat
	err := Get().Where("address = ? AND card_name = ?", address, cardName).First(&cardStat).Error
	if err != nil {
		return nil, err
	}
	return &cardStat, nil
}

// CreateCardStat 创建卡牌统计记录
func CreateCardStat(cardStat *dao.CardStat) error {
	return Get().Create(cardStat).Error
}

// UpdateCardStat 更新卡牌统计记录
func UpdateCardStat(cardStat *dao.CardStat) error {
	return Get().Save(cardStat).Error
}

// GetOrCreateCardStat 获取或创建卡牌统计记录
func GetOrCreateCardStat(address string, cardID uint) (*dao.CardStat, error) {
	var cardStat dao.CardStat
	err := Get().Where("address = ? AND card_name = ?", address, cardID).First(&cardStat).Error
	if err != nil {
		// 卡牌统计不存在，创建新记录
		cardStat = dao.CardStat{
			Address:    address,
			CardID:     cardID,
			RoundCount: 0,
			UsageCount: 0,
			WinCount:   0,
			LoseCount:  0,
			TieCount:   0,
		}
		err = Get().Create(&cardStat).Error
		if err != nil {
			return nil, err
		}
	}
	return &cardStat, nil
}

// GetCardStatsInfo 获取卡牌统计信息的辅助方法（转换为API响应格式）
func GetCardStatsInfo(cardStats []dao.CardStat) []CardStatInfo {
	result := make([]CardStatInfo, len(cardStats))
	for i, stat := range cardStats {
		result[i] = CardStatInfo{
			CardID: stat.CardID,
		}
		if stat.RoundCount == 0 {
			result[i].Frequency = 0.0
			result[i].WinningRate = 0.0
		} else {
			frequency := float64(stat.UsageCount) / float64(stat.RoundCount)
			winningRate := float64(stat.WinCount) / float64(stat.UsageCount)

			// 保留2位小数
			result[i].Frequency = math.Round(frequency*100) / 100
			result[i].WinningRate = math.Round(winningRate*100) / 100
		}
	}
	return result
}

func UpdateCardStatByAddresses(addresses []string) ([]*dao.CardStat, error) {
	if len(addresses) == 0 {
		return []*dao.CardStat{}, nil
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
	var results []*dao.CardStat
	err := db.Transaction(func(tx *gorm.DB) error {
		results = make([]*dao.CardStat, 0, len(deduplicatedAddresses))

		for _, address := range deduplicatedAddresses {
			// 步骤1：查询该地址在 card_stats 表中的所有记录
			var existingCardStats []dao.CardStat
			err := tx.Where("address = ?", address).Find(&existingCardStats).Error
			if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
				return fmt.Errorf("failed to query existing card stats for address %s: %v", address, err)
			}

			// 创建地址到卡牌统计的映射
			cardStatMap := make(map[uint]*dao.CardStat)
			existedRoundCount := uint(0)
			for i := range existingCardStats {
				cardStatMap[existingCardStats[i].CardID] = &existingCardStats[i]
				if existedRoundCount < existingCardStats[i].RoundCount {
					existedRoundCount = existingCardStats[i].RoundCount
				}
			}

			// 如果没有现有记录，初始化一个空的统计记录
			if len(existingCardStats) == 0 {
				// 创建一个默认的空统计记录，用于后续处理
				defaultStat := &dao.CardStat{
					Address:               address,
					CardID:                0, // 临时ID，会在后续处理中被替换
					RoundCount:            0, // 临时值，会在后续处理中被替换
					UsageCount:            0,
					WinCount:              0,
					LoseCount:             0,
					TieCount:              0,
					LastPlayerRoundInfoID: 0,
				}
				// 将默认记录添加到映射中，确保后续逻辑能正常执行
				cardStatMap[0] = defaultStat
			}

			// 步骤2：查询该地址在 game_round_infos 表中的新记录
			// 获取所有卡牌的最后 round_info_id
			var maxRoundInfoID uint = 0
			for _, stat := range existingCardStats {
				if stat.LastPlayerRoundInfoID > maxRoundInfoID {
					maxRoundInfoID = stat.LastPlayerRoundInfoID
				}
			}

			// 查询新的游戏轮次
			var newGameRounds []dao.PlayerRoundInfo
			query := tx.Where("wallet_address = ?", address)
			if maxRoundInfoID > 0 {
				query = query.Where("id > ?", maxRoundInfoID)
			}
			query = query.Where("player_ready > 0")

			log.Debugf("query sql: %#v", query.Statement.SQL.String())

			queryErr := query.Order("id ASC").Find(&newGameRounds).Error
			if queryErr != nil && !errors.Is(queryErr, gorm.ErrRecordNotFound) {
				return fmt.Errorf("failed to query new game rounds for address %s: %v", address, queryErr)
			}

			// 如果没有新的游戏轮次，直接添加到结果中
			if len(newGameRounds) == 0 {
				// 如果有现有记录，添加到结果中
				if len(existingCardStats) > 0 {
					for _, stat := range existingCardStats {
						results = append(results, &stat)
					}
				} else {
					// 如果没有现有记录，创建一个空的默认记录
					defaultStat := &dao.CardStat{
						Address:               address,
						CardID:                0,
						RoundCount:            0, // 没有新轮次，所以为0
						UsageCount:            0,
						WinCount:              0,
						LoseCount:             0,
						TieCount:              0,
						LastPlayerRoundInfoID: 0,
					}
					results = append(results, defaultStat)
				}
				continue
			}

			//log.Debugf("address: %s, existedRoundCount: %d, len(newGameRounds): %d, newGameRounds: %#v", address, existedRoundCount, len(newGameRounds), newGameRounds)
			log.Debugf("address: %s, existedRoundCount: %d, len(newGameRounds): %d", address, existedRoundCount, len(newGameRounds))

			// 收集所有新的 round_info_id
			var newRoundInfoIDs []uint
			for _, round := range newGameRounds {
				newRoundInfoIDs = append(newRoundInfoIDs, round.ID)
			}

			// 步骤3：查询这些轮次中的卡牌提交记录
			var cardSubmissions []dao.RoundSubmittedCard
			submissionErr := tx.Where("player_round_info_id IN ?", newRoundInfoIDs).
				Select("player_round_info_id, card_id, element_relation").
				Find(&cardSubmissions).Error
			if submissionErr != nil && !errors.Is(submissionErr, gorm.ErrRecordNotFound) {
				return fmt.Errorf("failed to query card submissions for address %s: %v", address, submissionErr)
			}

			// 步骤4：统计每个卡牌的使用情况
			cardUsageStats := make(map[uint]struct {
				roundCount uint
				usageCount uint
				winCount   uint
				loseCount  uint
				tieCount   uint
			})

			// 计算该地址的总轮次数量（所有新游戏轮次）
			totalNewRounds := uint(len(newGameRounds))

			var mapCardIDMaxRoundInfoID map[uint]uint = make(map[uint]uint)
			for _, submission := range cardSubmissions {
				cardID := submission.CardID
				stats, exists := cardUsageStats[cardID]
				if !exists {
					stats = struct {
						roundCount uint
						usageCount uint
						winCount   uint
						loseCount  uint
						tieCount   uint
					}{}
				}

				// 每张卡牌在每轮游戏中只能使用一次
				stats.usageCount++

				// 根据 element_relation 判断胜负平
				// 使用 proto 包中定义的枚举值
				// OVER_POWER = 0, OVER_POWERED = 1, NURTURE = 2, NURTURED = 3, TIE = 4
				switch submission.ElementRelation {
				case proto.ElementRelation_OVER_POWER, proto.ElementRelation_NURTURED:
					stats.winCount++
				case proto.ElementRelation_OVER_POWERED, proto.ElementRelation_NURTURE:
					stats.loseCount++
				case proto.ElementRelation_TIE:
					stats.tieCount++
				}

				cardUsageStats[cardID] = stats

				if mapCardIDMaxRoundInfoID[cardID] < submission.PlayerRoundInfoID {
					mapCardIDMaxRoundInfoID[cardID] = submission.PlayerRoundInfoID
				}
			}

			// 步骤5：更新或创建卡牌统计记录
			for cardID, stats := range cardUsageStats {
				// 查找现有记录
				cardStat, exists := cardStatMap[cardID]
				if !exists {
					// 创建新记录
					cardStat = &dao.CardStat{
						Address:    address,
						CardID:     cardID,
						RoundCount: existedRoundCount + totalNewRounds, // 使用总轮次数量
						UsageCount: stats.usageCount,
						WinCount:   stats.winCount,
						LoseCount:  stats.loseCount,
						TieCount:   stats.tieCount,
						//LastPlayerRoundInfoID: newGameRounds[len(newGameRounds)-1].ID, // 最新的轮次ID
						LastPlayerRoundInfoID: mapCardIDMaxRoundInfoID[cardID], // 该cardID最新的轮次ID
					}

					if err := tx.Create(cardStat).Error; err != nil {
						return fmt.Errorf("failed to create card stat for address %s, card %d: %v", address, cardID, err)
					}
				} else {
					// 更新现有记录（不更新 address 和 card_id）
					updateData := map[string]interface{}{
						"round_count":               cardStat.RoundCount + totalNewRounds, // 使用总轮次数量
						"usage_count":               cardStat.UsageCount + stats.usageCount,
						"win_count":                 cardStat.WinCount + stats.winCount,
						"lose_count":                cardStat.LoseCount + stats.loseCount,
						"tie_count":                 cardStat.TieCount + stats.tieCount,
						"last_player_round_info_id": mapCardIDMaxRoundInfoID[cardID],
					}

					if err := tx.Model(cardStat).Updates(updateData).Error; err != nil {
						return fmt.Errorf("failed to update card stat for address %s, card %d: %v", address, cardID, err)
					}

					// 重新查询更新后的记录
					if err := tx.Where("address = ? AND card_id = ?", address, cardID).First(cardStat).Error; err != nil {
						return fmt.Errorf("failed to query updated card stat for address %s, card %d: %v", address, cardID, err)
					}
				}

				results = append(results, cardStat)
			}

			// 更新同一地址下其他没有新数据的卡牌记录的 round_count
			for cardID, cardStat := range cardStatMap {
				if _, hasNewData := cardUsageStats[cardID]; !hasNewData {
					// 跳过临时ID为0的默认记录
					if cardID != 0 {
						// 更新 round_count 以保持一致性
						if totalNewRounds > 0 {
							otherUpdateData := map[string]interface{}{
								"round_count": cardStat.RoundCount + totalNewRounds,
							}
							if err := tx.Model(cardStat).Updates(otherUpdateData).Error; err != nil {
								return fmt.Errorf("failed to update round_count for other card stat, address %s, card %d: %v", address, cardID, err)
							}
							// 重新查询更新后的记录
							if err := tx.Where("address = ? AND card_id = ?", address, cardID).First(cardStat).Error; err != nil {
								return fmt.Errorf("failed to query updated other card stat for address %s, card %d: %v", address, cardID, err)
							}
						}
						// 添加到结果中
						results = append(results, cardStat)
					}
				}
			}
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	return results, nil
}
