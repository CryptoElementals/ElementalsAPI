package db

import (
	"fmt"

	dao "github.com/CryptoElementals/common/models"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// LockGameForUpdateTx locks the games row for the given id (SELECT … FOR UPDATE on supported drivers).
func LockGameForUpdateTx(tx *gorm.DB, gameID int64) error {
	var row dao.Game
	return tx.Clauses(clause.Locking{Strength: "UPDATE"}).
		Where("id = ?", gameID).
		First(&row).Error
}

// LoadGameByGameIDTx loads a full game graph using tx (same preloads as LoadGameByGameID).
func LoadGameByGameIDTx(tx *gorm.DB, gameID int64) (*dao.Game, error) {
	var game dao.Game
	err := preloadGameInfo(tx.Where("id = ?", gameID)).First(&game).Error
	if err != nil {
		return nil, err
	}
	return &game, nil
}

// WithGameMutationTx runs fn inside a single transaction after locking the game row.
// All writes for this game in fn should use the same tx so commits are atomic across instances.
func WithGameMutationTx(gameID int64, fn func(tx *gorm.DB, game *dao.Game) error) error {
	if gameID == 0 {
		return fmt.Errorf("WithGameMutationTx: game id required")
	}
	return Get().Transaction(func(tx *gorm.DB) error {
		if err := LockGameForUpdateTx(tx, gameID); err != nil {
			return err
		}
		game, err := LoadGameByGameIDTx(tx, gameID)
		if err != nil {
			return err
		}
		return fn(tx, game)
	})
}
