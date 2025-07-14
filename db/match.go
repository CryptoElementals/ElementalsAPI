package db

import (
	"fmt"
	"strings"

	dao "github.com/CryptoElementals/common/models"
)

// CreateMatch 创建匹配记录（一行记录包含两个玩家）
func CreateMatch(match *dao.Match) error {
	return Get().Create(match).Error
}

// GetMatchByMatchID 根据MatchID获取匹配记录（返回单行记录）
func GetMatchByMatchID(matchID string) (*dao.Match, error) {
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

// UpdateMatchRoomID 更新匹配记录的RoomID
func UpdateMatchRoomID(matchID string, roomID string) error {
	return Get().Model(&dao.MatchPlayer{}).Where("match_id = ?", matchID).Update("room_id", roomID).Error
}

// UpdatePlayerStatus 更新指定玩家的确认状态
func UpdatePlayerStatus(matchID string, playerAddress string, status string) error {
	// 检查是玩家1还是玩家2
	match, err := GetMatchByMatchID(matchID)
	if err != nil {
		return err
	}

	// 统一地址格式为小写进行比较
	playerAddress = strings.ToLower(playerAddress)
	player1Address := strings.ToLower(match.Player1Address)
	player2Address := strings.ToLower(match.Player2Address)

	if playerAddress == player1Address {
		return Get().Model(&dao.Match{}).Where("match_id = ?", matchID).Update("player1_status", status).Error
	} else if playerAddress == player2Address {
		return Get().Model(&dao.Match{}).Where("match_id = ?", matchID).Update("player2_status", status).Error
	}

	return fmt.Errorf("player address %s not found in match %s", playerAddress, matchID)
}

// CheckBothPlayersConfirmed 检查双方是否都已确认
func CheckBothPlayersConfirmed(matchID string) (bool, error) {
	match, err := GetMatchByMatchID(matchID)
	if err != nil {
		return false, err
	}

	// 检查两个玩家是否都已确认
	return match.Player1Status == "confirmed" && match.Player2Status == "confirmed", nil
}

// GetMatchesByAddress 根据地址获取用户的匹配记录
func GetMatchesByAddress(address string) ([]dao.Match, error) {
	var matches []dao.Match
	address = strings.ToLower(address)
	err := Get().Where("LOWER(player1_address) = ? OR LOWER(player2_address) = ?", address, address).Find(&matches).Error
	return matches, err
}

// GetActiveMatchByAddress 根据地址获取用户当前活跃的匹配记录
func GetActiveMatchByAddress(address string) (*dao.Match, error) {
	var match dao.Match
	address = strings.ToLower(address)
	err := Get().Where("(LOWER(player1_address) = ? OR LOWER(player2_address) = ?) AND (player1_status IN (?) OR player2_status IN (?))",
		address, address, []string{"matched", "confirmed"}, []string{"matched", "confirmed"}).First(&match).Error
	if err != nil {
		return nil, err
	}
	return &match, nil
}

// UpdateMatchStatusByRoomID 根据RoomID更新匹配记录状态
func UpdateMatchStatusByRoomID(roomID string, status string) error {
	return Get().Model(&dao.Match{}).Where("room_id = ?", roomID).Update("player1_status", status).Update("player2_status", status).Error
}
