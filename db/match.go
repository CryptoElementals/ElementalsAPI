package db

import (
	dao "github.com/CryptoElementals/common/models"
	"gorm.io/gorm"
)

func preload(d *gorm.DB) *gorm.DB {
	return d.Preload("Children", preload)
}

// CreateMatch 创建匹配记录（为单个用户创建）
func CreateMatch(match *dao.Match) error {
	return Get().Create(match).Error
}

// GetMatchByMatchID 根据MatchID获取匹配记录（返回第一个，用于兼容旧接口）
func GetMatchByMatchID(matchID string) (*dao.MatchPlayer, error) {
	var match dao.MatchPlayer
	err := Get().Where("id = ?", matchID).First(&match).Error
	if err != nil {
		return nil, err
	}
	return &match, nil
}

func LoadFullMatchByMatchID(matchID string) (*dao.Match, error) {
	var match dao.Match
	err := Get().Where("id = ?", matchID).
		Preload("Players").
		Preload("Players.Player").
		Preload("Rounds").
		Preload("Rounds.RoundPlayers").
		Preload("Rounds.RoundPlayers.RoundCards").
		Preload("Rounds.RoundPlayers.RoundCards.Card").
		Preload("Rounds.RoundPlayers.RoundCards.Items").
		Preload("Rounds.RoundPlayers.RoundCards.Items.Effects").
		First(&match).Error
	if err != nil {
		return nil, err
	}
	return &match, nil
}

func SaveMatch(match *dao.Match) error {
	return Get().Save(match).Error
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
