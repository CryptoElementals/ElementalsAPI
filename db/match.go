package db

import (
	"fmt"

	dao "github.com/CryptoElementals/common/models"
)

// CreateMatch 创建匹配记录（为单个用户创建）
func CreateMatch(match *dao.Match) error {
	return Get().Create(match).Error
}

// GetMatchesByMatchID 根据MatchID获取所有匹配记录（两个用户的记录）
func GetMatchesByMatchID(matchID string) (dao.Match, error) {
	var match dao.Match
	err := Get().Where("id = ?", matchID).First(&match).Error
	return match, err
}

// GetMatchByMatchID 根据MatchID获取匹配记录（返回第一个，用于兼容旧接口）
func GetMatchByMatchID(matchID string) (*dao.MatchPlayer, error) {
	var match dao.MatchPlayer
	err := Get().Where("match_id = ?", matchID).First(&match).Error
	if err != nil {
		return nil, err
	}
	return &match, nil
}

// UpdateMatchRoomID 更新匹配记录的RoomID（更新所有相关记录）
func UpdateMatchRoomID(matchID string, roomID string) error {
	return Get().Model(&dao.MatchPlayer{}).Where("match_id = ?", matchID).Update("room_id", roomID).Error
}

// UpdateMatchStatus 更新匹配记录状态（更新所有相关记录）
func UpdateMatchStatus(matchID string, status string) error {
	return Get().Model(&dao.MatchPlayer{}).Where("match_id = ?", matchID).Update("status", status).Error
}

// UpdatePlayerStatus 更新指定玩家的确认状态
func UpdatePlayerStatus(matchID string, playerAddress string, status string) error {
	return Get().Model(&dao.MatchPlayer{}).Where("match_id = ? AND address = ?", matchID, playerAddress).Update("status", status).Error
}

// CheckBothPlayersConfirmed 检查双方是否都已确认
func CheckBothPlayersConfirmed(matchID string) (bool, error) {
	var matches []dao.MatchPlayer
	err := Get().Where("match_id = ?", matchID).Find(&matches).Error
	if err != nil {
		return false, err
	}

	if len(matches) != 2 {
		return false, fmt.Errorf("匹配记录数量不正确")
	}

	// 检查两个玩家是否都已确认
	bothConfirmed := true
	for _, match := range matches {
		if match.Status != "confirmed" {
			bothConfirmed = false
			break
		}
	}

	return bothConfirmed, nil
}

// GetMatchesByAddress 根据地址获取用户的匹配记录
func GetMatchesByAddress(address string) ([]dao.MatchPlayer, error) {
	var matches []dao.MatchPlayer
	err := Get().Where("address = ?", address).Find(&matches).Error
	return matches, err
}

// GetActiveMatchByAddress 根据地址获取用户当前活跃的匹配记录
func GetActiveMatchByAddress(address string) (*dao.MatchPlayer, error) {
	var match dao.MatchPlayer
	err := Get().Where("address = ? AND status IN (?)", address, []string{"matched", "confirmed"}).First(&match).Error
	if err != nil {
		return nil, err
	}
	return &match, nil
}
