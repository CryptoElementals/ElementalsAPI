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
	tx := Get().Where("status != ? and status != ?", proto.GameStatus_GAME_END, proto.GameStatus_GAME_ABORTED)
	err := preloadGameInfo(tx).Find(&games).Error
	if err != nil {
		return nil, err
	}
	return games, nil
}

// GetActiveGameByPlayer finds a non-ended/non-aborted game that contains the given player.
func GetActiveGameByPlayer(playerID int64, tempAddr string) (*dao.Game, error) {
	var game dao.Game
	tx := preloadGameInfo(Get()).
		Joins("JOIN game_player_infos ON game_player_infos.game_id = games.id").
		Where("game_player_infos.player_id = ? AND LOWER(game_player_infos.temporary_address) = LOWER(?)",
			playerID, tempAddr).
		Where("games.status != ? AND games.status != ?",
			proto.GameStatus_GAME_END, proto.GameStatus_GAME_ABORTED)

	if err := tx.First(&game).Error; err != nil {
		return nil, err
	}
	return &game, nil
}

func preloadGameInfo(tx *gorm.DB) *gorm.DB {
	return tx.
		Preload("GameArgs").
		Preload("Players").
		Preload("GameResult").
		Preload("GameResult.BattleReward").
		Preload("GameResult.BattleReward.PlayerRewards").
		Preload("Turns", func(db *gorm.DB) *gorm.DB {
			return db.Order("round_number ASC, turn_number ASC")
		}).
		Preload("Turns.PlayerTurnInfos")
}

// UpdateMatchStatusByRoomID 根据RoomID更新匹配记录状态
func UpdateMatchStatusByRoomID(roomID string, status string) error {
	return Get().Model(&dao.Game{}).Where("room_id = ?", roomID).Update("status", status).Error
}

// 通过 MatchID 查询匹配记录
func GetMatchByMatchID(matchID string) (*dao.Game, error) {
	var game dao.Game
	err := Get().Where("match_id = ?", matchID).First(&game).Error
	if err != nil {
		return nil, err
	}
	return &game, nil
}
