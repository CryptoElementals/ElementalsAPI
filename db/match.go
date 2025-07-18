package db

import (
	dao "github.com/CryptoElementals/common/models"
	"github.com/CryptoElementals/common/rpc/proto"
	"gorm.io/gorm"
)

// CreateMatch 创建匹配记录（为单个用户创建）
func CreateGame(game *dao.Game) error {
	return Get().Create(game).Error
}

func LoadGameByGameID(gameID uint) (*dao.Game, error) {
	var game dao.Game
	tx := Get().Where("id = ?", gameID)
	err := preloadGameInfo(tx).First(&game).Error
	if err != nil {
		return nil, err
	}
	return &game, nil
}

func GetAllActiveGames() ([]*dao.Game, error) {
	var games []*dao.Game
	tx := Get().Where("status != ?", proto.GameStatus_GAME_END)
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

func SaveGame(game *dao.Game) error {
	return Get().Save(game).Error
}

func SaveRound(round *dao.Round) error {
	return Get().Save(round).Error
}

func SavePlayerRoundInfo(playerRoundInfo *dao.PlayerRoundInfo) error {
	return Get().Save(playerRoundInfo).Error
}

func SaveRoundSubmittedCard(card *dao.RoundSubmittedCard) error {
	return Get().Save(card).Error
}

func SaveGamePlayerInfo(gamePlayerInfo *dao.GamePlayerInfo) error {
	return Get().Save(gamePlayerInfo).Error
}

// UpdateMatchStatusByRoomID 根据RoomID更新匹配记录状态
func UpdateMatchStatusByRoomID(roomID string, status string) error {
	return Get().Model(&dao.Game{}).Where("room_id = ?", roomID).Update("status", status).Error
}
