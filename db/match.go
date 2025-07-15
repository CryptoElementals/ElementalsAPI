package db

import (
	dao "github.com/CryptoElementals/common/models"
	"github.com/CryptoElementals/common/rpc/proto"
	"gorm.io/gorm"
)

// CreateMatch 创建匹配记录（为单个用户创建）
func CreateGame(game *dao.GameInfo) error {
	return Get().Create(game).Error
}

func LoadGameByGameID(gameID uint) (*dao.GameInfo, error) {
	var game dao.GameInfo
	tx := Get().Where("id = ?", gameID)
	err := preloadGameInfo(tx).First(&game).Error
	if err != nil {
		return nil, err
	}
	return &game, nil
}

func GetAllActiveGames() ([]*dao.GameInfo, error) {
	var games []*dao.GameInfo
	tx := Get().Where("status = ? or status = ?", proto.GameStatus_GAME_RUNNING, proto.GameStatus_GAME_WAITTING_CONTRACT)
	err := preloadGameInfo(tx).Find(&games).Error
	if err != nil {
		return nil, err
	}
	return games, nil
}

func preloadGameInfo(tx *gorm.DB) *gorm.DB {
	return tx.Preload("Players").
		Preload("Rounds").
		Preload("Rounds.PlayerRoundInfos").
		Preload("Rounds.PlayerRoundInfos.RoundSubmittedCards")
}

func SaveGame(match *dao.GameInfo) error {
	return Get().Save(match).Error
}

// UpdateMatchRoomID 更新匹配记录的RoomID（更新所有相关记录）
func UpdateMatchRoomID(matchID string, roomID string) error {
	return Get().Model(&dao.GamePlayer{}).Where("match_id = ?", matchID).Update("room_id", roomID).Error
}

// UpdateMatchStatus 更新匹配记录状态（更新所有相关记录）
func UpdateMatchStatus(matchID string, status string) error {
	return Get().Model(&dao.GamePlayer{}).Where("match_id = ?", matchID).Update("status", status).Error
}

// UpdatePlayerStatus 更新指定玩家的确认状态
func UpdatePlayerStatus(matchID string, playerAddress string, status string) error {
	return Get().Model(&dao.GamePlayer{}).Where("match_id = ? AND address = ?", matchID, playerAddress).Update("status", status).Error
}

// GetMatchesByAddress 根据地址获取用户的匹配记录
func GetMatchesByAddress(address string) ([]dao.GamePlayer, error) {
	var matches []dao.GamePlayer
	err := Get().Where("address = ?", address).Find(&matches).Error
	return matches, err
}

// GetActiveMatchByAddress 根据地址获取用户当前活跃的匹配记录
func GetActiveMatchByAddress(address string) (*dao.GamePlayer, error) {
	var match dao.GamePlayer
	err := Get().Where("address = ? AND status IN (?)", address, []string{"matched", "confirmed"}).First(&match).Error
	if err != nil {
		return nil, err
	}
	return &match, nil
}

// UpdateMatchStatusByRoomID 根据RoomID更新匹配记录状态
func UpdateMatchStatusByRoomID(roomID string, status string) error {
	return Get().Model(&dao.GameInfo{}).Where("room_id = ?", roomID).Update("status", status).Error
}
