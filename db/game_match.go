package db

import (
	"context"
	"errors"
	"strings"
	"time"

	dao "github.com/CryptoElementals/common/models"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// InsertGameMatch creates a pending game_match row (snowflake id from BeforeCreate).
func InsertGameMatch(ctx context.Context, m *dao.GameMatch) error {
	return Get().WithContext(ctx).Create(m).Error
}

// GetGameMatchByID loads by primary key.
func GetGameMatchByID(ctx context.Context, id int64) (*dao.GameMatch, error) {
	var m dao.GameMatch
	err := Get().WithContext(ctx).First(&m, "id = ?", id).Error
	if err != nil {
		return nil, err
	}
	return &m, nil
}

// TryConfirmGameMatchPlayer sets the confirming player's timestamp if still pending and not yet set.
// Returns updated row, whether this call flipped the column, and whether both are now confirmed.
func TryConfirmGameMatchPlayer(ctx context.Context, matchID int64, playerID int64, tempAddress string) (m *dao.GameMatch, thisPlayerJustConfirmed bool, bothConfirmed bool, err error) {
	tempAddress = strings.ToLower(strings.TrimSpace(tempAddress))
	err = Get().WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var cur dao.GameMatch
		if e := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&cur, "id = ?", matchID).Error; e != nil {
			return e
		}
		if cur.Status != dao.GameMatchStatusPending {
			return errors.New("game_match not pending")
		}
		isP1 := cur.Player1ID == playerID && strings.EqualFold(cur.Player1TempAddress, tempAddress)
		isP2 := cur.Player2ID == playerID && strings.EqualFold(cur.Player2TempAddress, tempAddress)
		if !isP1 && !isP2 {
			return errors.New("player not in match")
		}
		now := time.Now()
		if isP1 {
			if cur.Player1ConfirmedAt == nil {
				if e := tx.Model(&dao.GameMatch{}).Where("id = ? AND player1_confirmed_at IS NULL AND status = ?", matchID, dao.GameMatchStatusPending).
					Update("player1_confirmed_at", now).Error; e != nil {
					return e
				}
				thisPlayerJustConfirmed = true
			}
		} else if isP2 {
			if cur.Player2ConfirmedAt == nil {
				if e := tx.Model(&dao.GameMatch{}).Where("id = ? AND player2_confirmed_at IS NULL AND status = ?", matchID, dao.GameMatchStatusPending).
					Update("player2_confirmed_at", now).Error; e != nil {
					return e
				}
				thisPlayerJustConfirmed = true
			}
		}
		if e := tx.First(&m, "id = ?", matchID).Error; e != nil {
			return e
		}
		bothConfirmed = m.Player1ConfirmedAt != nil && m.Player2ConfirmedAt != nil
		return nil
	})
	return m, thisPlayerJustConfirmed, bothConfirmed, err
}

// ClaimGameMatchForCreation transitions pending → creating when both players confirmed (single winner).
func ClaimGameMatchForCreation(ctx context.Context, matchID int64) (*dao.GameMatch, bool, error) {
	var m dao.GameMatch
	var claimed bool
	err := Get().WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var cur dao.GameMatch
		if e := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&cur, "id = ?", matchID).Error; e != nil {
			return e
		}
		if cur.Status != dao.GameMatchStatusPending {
			return nil
		}
		if cur.Player1ConfirmedAt == nil || cur.Player2ConfirmedAt == nil {
			return nil
		}
		res := tx.Model(&dao.GameMatch{}).Where("id = ? AND status = ?", matchID, dao.GameMatchStatusPending).
			Update("status", dao.GameMatchStatusCreating)
		if res.Error != nil {
			return res.Error
		}
		if res.RowsAffected == 0 {
			return nil
		}
		claimed = true
		return nil
	})
	if err != nil || !claimed {
		return nil, false, err
	}
	if e := Get().WithContext(ctx).First(&m, "id = ?", matchID).Error; e != nil {
		return nil, false, e
	}
	return &m, true, nil
}

// RevertGameMatchToPending rolls back creating → pending after a failed game insert.
func RevertGameMatchToPending(ctx context.Context, matchID int64) error {
	res := Get().WithContext(ctx).Model(&dao.GameMatch{}).
		Where("id = ? AND status = ?", matchID, dao.GameMatchStatusCreating).
		Update("status", dao.GameMatchStatusPending)
	return res.Error
}

// CancelPendingGameMatch marks a still-pending game_match as cancelled (e.g. continue rematch timeout or player joined queue).
func CancelPendingGameMatch(ctx context.Context, matchID int64) error {
	res := Get().WithContext(ctx).Model(&dao.GameMatch{}).
		Where("id = ? AND status = ?", matchID, dao.GameMatchStatusPending).
		Update("status", dao.GameMatchStatusCancelled)
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return nil
	}
	return nil
}

// CompleteClaimedGameMatch sets game_created and game_id from creating state.
func CompleteClaimedGameMatch(ctx context.Context, matchID int64, gameID uint) error {
	res := Get().WithContext(ctx).Model(&dao.GameMatch{}).
		Where("id = ? AND status = ?", matchID, dao.GameMatchStatusCreating).
		Updates(map[string]interface{}{
			"status":  dao.GameMatchStatusGameCreated,
			"game_id": gameID,
		})
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return errors.New("game_match not in creating state")
	}
	return nil
}
