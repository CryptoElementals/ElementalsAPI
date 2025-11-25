package db

import (
	"errors"
	"fmt"
	"math"
	"strconv"

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

// GetCardStatsByPlayerID 根据 player_id 获取该用户的所有卡牌统计
func GetCardStatsByPlayerID(playerID string) ([]dao.CardStat, error) {
	profile, err := GetUserProfileByPlayerID(playerID)
	if err != nil {
		return nil, err
	}
	// 当前 card_stats 表以 address 为维度，先按地址查询
	return GetCardStatsByAddress(profile.Address)
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

func UpdateCardStatByPlayerIds(playerIds []string) ([]*dao.CardStat, error) {
	if len(playerIds) == 0 {
		return []*dao.CardStat{}, nil
	}

	// 去重 playerIds 并转换为 int64
	uniquePlayerIDs := make(map[int64]bool)
	var deduplicatedPlayerIDs []int64
	for _, playerIDStr := range playerIds {
		playerID, err := strconv.ParseInt(playerIDStr, 10, 64)
		if err != nil {
			log.Debugf("invalid player ID %s, skipping: %v", playerIDStr, err)
			continue
		}
		if !uniquePlayerIDs[playerID] {
			uniquePlayerIDs[playerID] = true
			deduplicatedPlayerIDs = append(deduplicatedPlayerIDs, playerID)
		}
	}

	if len(deduplicatedPlayerIDs) == 0 {
		return []*dao.CardStat{}, nil
	}

	// 使用事务确保原子性
	var results []*dao.CardStat
	err := Get().Transaction(func(tx *gorm.DB) error {
		results = make([]*dao.CardStat, 0, len(deduplicatedPlayerIDs))

		for _, userID := range deduplicatedPlayerIDs {
			// 步骤1：通过 playerId (userID) 获取 UserProfile 和 address（在事务中查询）
			var profile dao.UserProfile
			if err := tx.Where("user_id = ?", userID).First(&profile).Error; err != nil {
				if errors.Is(err, gorm.ErrRecordNotFound) {
					// 用户不存在，跳过
					log.Debugf("user profile not found for player ID %d, skipping", userID)
					continue
				}
				return fmt.Errorf("failed to get user profile for player ID %d: %v", userID, err)
			}
			address := profile.Address

			// 步骤2：查询该地址在 card_stats 表中的所有记录
			var existingCardStats []dao.CardStat
			if err := tx.Where("address = ?", address).Find(&existingCardStats).Error; err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
				return fmt.Errorf("failed to query existing card stats for player ID %d (address %s): %v", userID, address, err)
			}

			// 创建卡牌ID到统计记录的映射
			cardStatMap := make(map[uint]*dao.CardStat)
			existedRoundCount := uint(0)
			var maxRoundID uint = 0
			for i := range existingCardStats {
				cardStatMap[existingCardStats[i].CardID] = &existingCardStats[i]
				if existedRoundCount < existingCardStats[i].RoundCount {
					existedRoundCount = existingCardStats[i].RoundCount
				}
				// LastPlayerRoundInfoID 现在存储的是 RoundID
				if existingCardStats[i].LastPlayerRoundInfoID > maxRoundID {
					maxRoundID = existingCardStats[i].LastPlayerRoundInfoID
				}
			}

			// 步骤3：查询新的 PlayerTurnInfo 记录
			// 查询该用户的所有 PlayerTurnInfo（TurnSubmittedCard 会在后续过滤）
			var newPlayerTurnInfos []dao.PlayerTurnInfo
			query := tx.Where("player_id = ?", userID)

			// 如果已有记录，只查询新的 Turn（通过 TurnID 关联到 Round，再通过 RoundID 过滤）
			if maxRoundID > 0 {
				// 先找到所有相关的 TurnID，这些 Turn 属于 RoundID > maxRoundID 的 Round
				var relevantTurnIDs []uint
				subQuery := tx.Table("turns").
					Select("id").
					Where("round_id > ?", maxRoundID)
				if err := subQuery.Pluck("id", &relevantTurnIDs).Error; err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
					return fmt.Errorf("failed to query relevant turn IDs for player ID %d: %v", userID, err)
				}

				if len(relevantTurnIDs) > 0 {
					query = query.Where("turn_id IN ?", relevantTurnIDs)
				} else {
					// 没有新的 Turn，直接处理现有记录
					if len(existingCardStats) > 0 {
						for _, stat := range existingCardStats {
							results = append(results, &stat)
						}
					}
					continue
				}
			}

			if err := query.Order("id ASC").Find(&newPlayerTurnInfos).Error; err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
				return fmt.Errorf("failed to query new player turn infos for player ID %d: %v", userID, err)
			}

			// 过滤掉 TurnSubmittedCard 为 nil 的记录
			filteredTurnInfos := make([]dao.PlayerTurnInfo, 0, len(newPlayerTurnInfos))
			for i := range newPlayerTurnInfos {
				if newPlayerTurnInfos[i].TurnSubmittedCard != nil {
					filteredTurnInfos = append(filteredTurnInfos, newPlayerTurnInfos[i])
				}
			}
			newPlayerTurnInfos = filteredTurnInfos

			// 如果没有新的记录，直接返回现有记录
			if len(newPlayerTurnInfos) == 0 {
				if len(existingCardStats) > 0 {
					for _, stat := range existingCardStats {
						results = append(results, &stat)
					}
				}
				continue
			}

			// 步骤4：收集所有相关的 TurnID 和 RoundID
			turnIDs := make([]uint, 0, len(newPlayerTurnInfos))
			turnIDMap := make(map[uint]*dao.PlayerTurnInfo)
			for i := range newPlayerTurnInfos {
				turnIDs = append(turnIDs, newPlayerTurnInfos[i].TurnID)
				turnIDMap[newPlayerTurnInfos[i].TurnID] = &newPlayerTurnInfos[i]
			}

			// 查询这些 Turn 对应的 RoundID
			var turns []dao.Turn
			if err := tx.Where("id IN ?", turnIDs).Select("id, round_id").Find(&turns).Error; err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
				return fmt.Errorf("failed to query turns for player ID %d: %v", userID, err)
			}

			// 创建 TurnID -> RoundID 的映射
			turnToRoundMap := make(map[uint]uint)
			roundIDSet := make(map[uint]bool)
			for _, turn := range turns {
				turnToRoundMap[turn.ID] = turn.RoundID
				roundIDSet[turn.RoundID] = true
			}

			// 计算新的 Round 数量
			totalNewRounds := uint(len(roundIDSet))

			// 步骤5：统计每个卡牌的使用情况
			cardUsageStats := make(map[uint]struct {
				usageCount uint
				winCount   uint
				loseCount  uint
				tieCount   uint
				maxRoundID uint // 该卡牌最新的 RoundID
			})

			for _, turnInfo := range newPlayerTurnInfos {
				if turnInfo.TurnSubmittedCard == nil {
					continue
				}

				cardID := uint(turnInfo.TurnSubmittedCard.CardID)
				roundID := turnToRoundMap[turnInfo.TurnID]

				stats, exists := cardUsageStats[cardID]
				if !exists {
					stats = struct {
						usageCount uint
						winCount   uint
						loseCount  uint
						tieCount   uint
						maxRoundID uint
					}{}
				}

				// 每张卡牌在每轮游戏中可以使用多次（每个 Turn 一次）
				stats.usageCount++

				// 根据 element_relation 判断胜负平
				switch turnInfo.TurnSubmittedCard.ElementRelation {
				case proto.ElementRelation_OVER_POWER, proto.ElementRelation_NURTURED:
					stats.winCount++
				case proto.ElementRelation_OVER_POWERED, proto.ElementRelation_NURTURE:
					stats.loseCount++
				case proto.ElementRelation_TIE:
					stats.tieCount++
				}

				// 更新该卡牌最新的 RoundID
				if roundID > stats.maxRoundID {
					stats.maxRoundID = roundID
				}

				cardUsageStats[cardID] = stats
			}

			// 步骤6：更新或创建卡牌统计记录
			for cardID, stats := range cardUsageStats {
				// 查找现有记录
				cardStat, exists := cardStatMap[cardID]
				if !exists {
					// 创建新记录
					cardStat = &dao.CardStat{
						Address:               address,
						CardID:                cardID,
						RoundCount:            existedRoundCount + totalNewRounds,
						UsageCount:            stats.usageCount,
						WinCount:              stats.winCount,
						LoseCount:             stats.loseCount,
						TieCount:              stats.tieCount,
						LastPlayerRoundInfoID: stats.maxRoundID, // 存储 RoundID
					}

					if err := tx.Create(cardStat).Error; err != nil {
						return fmt.Errorf("failed to create card stat for address %s, card %d: %v", address, cardID, err)
					}
				} else {
					// 更新现有记录
					updateData := map[string]interface{}{
						"round_count":               cardStat.RoundCount + totalNewRounds,
						"usage_count":               cardStat.UsageCount + stats.usageCount,
						"win_count":                 cardStat.WinCount + stats.winCount,
						"lose_count":                cardStat.LoseCount + stats.loseCount,
						"tie_count":                 cardStat.TieCount + stats.tieCount,
						"last_player_round_info_id": stats.maxRoundID, // 存储 RoundID
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

			// 步骤7：更新同一地址下其他没有新数据的卡牌记录的 round_count
			for cardID, cardStat := range cardStatMap {
				if _, hasNewData := cardUsageStats[cardID]; !hasNewData {
					// 跳过临时ID为0的默认记录
					if cardID != 0 && totalNewRounds > 0 {
						// 更新 round_count 以保持一致性
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

		return nil
	})

	if err != nil {
		return nil, err
	}

	return results, nil
}
