package db

import (
	dao "github.com/CryptoElementals/common/models"
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
			Address:      address,
			CardID:       cardID,
			RoundCount:   0,
			UsageCount:   0,
			WinningCount: 0,
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
			result[i].Frequency = float64(stat.UsageCount) / float64(stat.RoundCount)
			result[i].WinningRate = float64(stat.WinningCount) / float64(stat.UsageCount)
		}
	}
	return result
}
