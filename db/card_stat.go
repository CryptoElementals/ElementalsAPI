package db

import (
	"encoding/json"
	"errors"
	"fmt"
	"math"

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
func GetOrCreateCardStat(playerId int64, cardID uint) (*dao.CardStat, error) {
	var cardStat dao.CardStat
	err := Get().Where("player_id = ? AND card_name = ?", playerId, cardID).First(&cardStat).Error
	if err != nil {
		// 卡牌统计不存在，创建新记录
		cardStat = dao.CardStat{
			PlayerID:   playerId,
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

// UpdateCardStatByPlayerIDs 根据玩家ID列表增量更新卡牌统计数据
//
// 实现流程：
//  1. 入口处理：去重玩家ID列表，空列表直接返回
//  2. 事务处理：所有操作在单个事务中执行，确保原子性
//  3. 对每个玩家：
//     a. 查询现有统计：从 card_stats 表获取该玩家的所有卡牌统计记录
//     - 构建 cardID -> CardStat 映射
//     - 记录最大 RoundCount 和 LastPlayerRoundInfoID（存储的是 RoundID）
//     b. 查询新数据：从 player_turn_infos 表查询该玩家的新 turn 记录
//     - 如果已有记录，通过 JOIN turns 表过滤出 round_id > maxRoundID 的 turn
//     - 只查询该玩家在新 round 中的 turn，避免查询所有玩家的 turn
//     - 过滤掉 TurnSubmittedCard 为 nil 的记录
//     c. 建立映射：查询 turns 表，建立 turn_id -> round_id 映射
//     - 因为 PlayerTurnInfo 表没有 round_id 字段，需要通过 turns 表关联
//     - 统计新增的 unique round 数量（totalNewRounds）
//     d. 统计卡牌使用：遍历新 turn 记录，统计每张卡牌的使用情况
//     - 统计使用次数（usageCount）
//     - 根据 ElementRelation 判断胜负平（winCount/loseCount/tieCount）
//     - 记录每张卡牌最新的 RoundID
//     e. 更新/创建统计：更新或创建 card_stats 记录
//     - 有记录则累加统计数据，更新 round_count
//     - 无记录则创建新记录
//     f. 同步其他卡牌：更新同一玩家下没有新数据的卡牌的 round_count
//     - 确保所有卡牌的 round_count 保持一致（都加上新增的 round 数）
//  4. 返回结果：返回所有更新/创建的卡牌统计记录
func UpdateCardStatByPlayerIDs(playerIDs []int64) ([]*dao.CardStat, error) {
	if len(playerIDs) == 0 {
		return []*dao.CardStat{}, nil
	}

	// 去重玩家ID
	uniquePlayerIDs := make(map[int64]bool)
	var deduplicatedPlayerIDs []int64
	for _, playerID := range playerIDs {
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

		for _, playerID := range deduplicatedPlayerIDs {
			// 步骤1：查询该玩家在 card_stats 表中的所有记录
			var existingCardStats []dao.CardStat
			if err := tx.Where("player_id = ?", playerID).Find(&existingCardStats).Error; err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
				return fmt.Errorf("failed to query existing card stats for player ID: %d, error: %v", playerID, err)
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

			// 步骤2：查询新的 PlayerTurnInfo 记录
			// 使用原始 SQL 查询，手动解析 JSON 字段，避免 GORM 解析嵌套结构
			type playerTurnInfoRow struct {
				ID                    uint   `gorm:"column:id"`
				TurnID                uint   `gorm:"column:turn_id"`
				PlayerID              int64  `gorm:"column:player_id"`
				PlayerStatus          int32  `gorm:"column:player_status"` // 使用 int32 因为 proto 枚举底层是 int32
				TemporaryAddress      string `gorm:"column:temporary_address"`
				TurnSubmittedCardJSON string `gorm:"column:turn_submitted_card"`
			}

			var rows []playerTurnInfoRow
			baseQuery := "SELECT id, turn_id, player_id, player_status, temporary_address, turn_submitted_card FROM player_turn_infos WHERE player_id = ?"
			var args []interface{}
			args = append(args, playerID)

			// 如果已有记录，只查询新的 Turn（通过 TurnID 关联到 Round，再通过 RoundID 过滤）
			if maxRoundID > 0 {
				// 只保留该玩家在新的 round 中的 turn
				baseQuery += " AND turn_id IN (SELECT pti.turn_id FROM player_turn_infos AS pti JOIN turns t ON pti.turn_id = t.id WHERE pti.player_id = ? AND t.round_id > ?)"
				args = append(args, playerID, maxRoundID)
			}

			baseQuery += " ORDER BY id ASC"

			if err := tx.Raw(baseQuery, args...).Scan(&rows).Error; err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
				return fmt.Errorf("failed to query new player turn infos for player ID %d: %v", playerID, err)
			}

			// 手动解析 JSON 字段并构建 PlayerTurnInfo 列表
			newPlayerTurnInfos := make([]dao.PlayerTurnInfo, 0, len(rows))
			for _, row := range rows {
				turnInfo := dao.PlayerTurnInfo{
					BaseModel: dao.BaseModel{
						ID: row.ID,
					},
					TurnID:           row.TurnID,
					PlayerID:         row.PlayerID,
					PlayerStatus:     proto.PlayerTurnStatus(row.PlayerStatus),
					TemporaryAddress: row.TemporaryAddress,
				}

				// 解析 TurnSubmittedCard JSON
				if row.TurnSubmittedCardJSON != "" && row.TurnSubmittedCardJSON != "null" {
					var card dao.TurnSubmittedCard
					if err := json.Unmarshal([]byte(row.TurnSubmittedCardJSON), &card); err == nil {
						turnInfo.TurnSubmittedCard = &card
						newPlayerTurnInfos = append(newPlayerTurnInfos, turnInfo)
					}
				}
			}

			// 如果没有新的记录，直接返回现有记录
			if len(newPlayerTurnInfos) == 0 {
				if len(existingCardStats) > 0 {
					for _, stat := range existingCardStats {
						results = append(results, &stat)
					}
				}
				continue
			}

			// 步骤3：收集所有相关的 TurnID 和 RoundID
			turnIDs := make([]uint, 0, len(newPlayerTurnInfos))
			for i := range newPlayerTurnInfos {
				turnIDs = append(turnIDs, newPlayerTurnInfos[i].TurnID)
			}

			// 查询这些 Turn 对应的 RoundID（使用原始 SQL 避免 GORM 解析嵌套结构）
			type turnRow struct {
				ID      uint `gorm:"column:id"`
				RoundID uint `gorm:"column:round_id"`
			}
			var turnRows []turnRow
			if err := tx.Raw("SELECT id, round_id FROM turns WHERE id IN ?", turnIDs).Scan(&turnRows).Error; err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
				return fmt.Errorf("failed to query turns for player ID %d: %v", playerID, err)
			}

			// 创建 TurnID -> RoundID 的映射
			turnToRoundMap := make(map[uint]uint)
			roundIDSet := make(map[uint]bool)
			for _, turn := range turnRows {
				turnToRoundMap[turn.ID] = turn.RoundID
				roundIDSet[turn.RoundID] = true
			}

			// 计算新的 Round 数量
			totalNewRounds := uint(len(roundIDSet))

			// 步骤4：统计每个卡牌的使用情况
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

			// 步骤5：更新或创建卡牌统计记录
			for cardID, stats := range cardUsageStats {
				// 查找现有记录
				cardStat, exists := cardStatMap[cardID]
				if !exists {
					// 创建新记录
					cardStat = &dao.CardStat{
						PlayerID:              playerID,
						CardID:                cardID,
						RoundCount:            existedRoundCount + totalNewRounds,
						UsageCount:            stats.usageCount,
						WinCount:              stats.winCount,
						LoseCount:             stats.loseCount,
						TieCount:              stats.tieCount,
						LastPlayerRoundInfoID: stats.maxRoundID, // 存储 RoundID
					}

					if err := tx.Create(cardStat).Error; err != nil {
						return fmt.Errorf("failed to create card stat for playerID %d, card %d: %v", playerID, cardID, err)
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
						return fmt.Errorf("failed to update card stat for playerID %d, card %d: %v", playerID, cardID, err)
					}

					// 重新查询更新后的记录
					if err := tx.Where("player_id = ? AND card_id = ?", playerID, cardID).First(cardStat).Error; err != nil {
						return fmt.Errorf("failed to query updated card stat for playerID %d, card %d: %v", playerID, cardID, err)
					}
				}

				results = append(results, cardStat)
			}

			// 步骤6：更新同一地址下其他没有新数据的卡牌记录的 round_count
			for cardID, cardStat := range cardStatMap {
				if _, hasNewData := cardUsageStats[cardID]; !hasNewData {
					// 跳过临时ID为0的默认记录
					if cardID != 0 && totalNewRounds > 0 {
						// 更新 round_count 以保持一致性
						otherUpdateData := map[string]interface{}{
							"round_count": cardStat.RoundCount + totalNewRounds,
						}
						if err := tx.Model(cardStat).Updates(otherUpdateData).Error; err != nil {
							return fmt.Errorf("failed to update round_count for other card stat, player %d, card %d: %v", playerID, cardID, err)
						}
						// 重新查询更新后的记录
						if err := tx.Where("player_id = ? AND card_id = ?", playerID, cardID).First(cardStat).Error; err != nil {
							return fmt.Errorf("failed to query updated other card stat for player %d, card %d: %v", playerID, cardID, err)
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
