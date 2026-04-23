package db

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	dao "github.com/CryptoElementals/common/models"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// LobbyPlayerRef identifies a player in lobby queue / game tables (temp address normalized lowercase).
type LobbyPlayerRef struct {
	PlayerID    int64
	TempAddress string
}

func normLobbyTemp(s string) string {
	return strings.ToLower(strings.TrimSpace(s))
}

// LobbyIsInQueue reports whether the player has a queue-only row (game_match_id = 0).
func LobbyIsInQueue(ctx context.Context, playerID int64, tempAddress string) (bool, error) {
	tempAddress = normLobbyTemp(tempAddress)
	var n int64
	err := Get().WithContext(ctx).Model(&dao.PlayerQueueEntry{}).
		Where("player_id = ? AND temp_address = ? AND game_match_id = ?", playerID, tempAddress, 0).
		Count(&n).Error
	return n > 0, err
}

// LobbyQueueJoinedAtMs returns CreatedAt as Unix milliseconds when the player is in the queue only.
func LobbyQueueJoinedAtMs(ctx context.Context, playerID int64, tempAddress string) (ms int64, ok bool, err error) {
	tempAddress = normLobbyTemp(tempAddress)
	var row dao.PlayerQueueEntry
	err = Get().WithContext(ctx).
		Where("player_id = ? AND temp_address = ? AND game_match_id = ?", playerID, tempAddress, 0).
		First(&row).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return 0, false, nil
	}
	if err != nil {
		return 0, false, err
	}
	return row.CreatedAt.UnixMilli(), true, nil
}

// LobbyListQueuedPlayers returns queue-only rows in FIFO order (created_at ASC).
func LobbyListQueuedPlayers(ctx context.Context) ([]dao.PlayerQueueEntry, error) {
	var rows []dao.PlayerQueueEntry
	err := Get().WithContext(ctx).Where("game_match_id = ?", 0).Order("created_at ASC").Find(&rows).Error
	return rows, err
}

// LobbyAddQueue inserts or refreshes queue membership when the player is not in-game or pending another match.
func LobbyAddQueue(ctx context.Context, playerID int64, tempAddress string, nowMs int64) (bool, error) {
	tempAddress = normLobbyTemp(tempAddress)
	var ingame int64
	if err := Get().WithContext(ctx).Model(&dao.PlayerGameEntry{}).
		Where("player_id = ? AND temp_address = ?", playerID, tempAddress).
		Count(&ingame).Error; err != nil {
		return false, err
	}
	if ingame > 0 {
		return false, nil
	}
	var row dao.PlayerQueueEntry
	err := Get().WithContext(ctx).Where("player_id = ? AND temp_address = ?", playerID, tempAddress).First(&row).Error
	t := time.UnixMilli(nowMs)
	if err == nil {
		if row.GameMatchID != 0 {
			return false, nil
		}
		if e := Get().WithContext(ctx).Model(&row).Updates(map[string]interface{}{
			"created_at": t,
			"updated_at": t,
		}).Error; e != nil {
			return false, e
		}
		return true, nil
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return false, err
	}
	row = dao.PlayerQueueEntry{
		PlayerID:    playerID,
		TempAddress: tempAddress,
		GameMatchID: 0,
	}
	row.CreatedAt = t
	row.UpdatedAt = t
	if e := Get().WithContext(ctx).Create(&row).Error; e != nil {
		return false, e
	}
	return true, nil
}

// LobbyRemoveFromQueue deletes a queue-only row for the player.
func LobbyRemoveFromQueue(ctx context.Context, playerID int64, tempAddress string) error {
	tempAddress = normLobbyTemp(tempAddress)
	return Get().WithContext(ctx).Unscoped().
		Where("player_id = ? AND temp_address = ? AND game_match_id = ?", playerID, tempAddress, 0).
		Delete(&dao.PlayerQueueEntry{}).Error
}

// LobbySetPendingPair links both players to matchID in player_queue_entries (creates rows if missing).
func LobbySetPendingPair(ctx context.Context, matchID int64, p1, p2 LobbyPlayerRef) (bool, error) {
	p1.TempAddress = normLobbyTemp(p1.TempAddress)
	p2.TempAddress = normLobbyTemp(p2.TempAddress)
	var ok bool
	err := Get().WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		for _, p := range []LobbyPlayerRef{p1, p2} {
			var n int64
			if e := tx.Model(&dao.PlayerGameEntry{}).Where("player_id = ? AND temp_address = ?", p.PlayerID, p.TempAddress).Count(&n).Error; e != nil {
				return e
			}
			if n > 0 {
				return nil
			}
		}
		for _, p := range []LobbyPlayerRef{p1, p2} {
			var row dao.PlayerQueueEntry
			err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("player_id = ? AND temp_address = ?", p.PlayerID, p.TempAddress).First(&row).Error
			if err == nil && row.GameMatchID != 0 && row.GameMatchID != matchID {
				return nil
			}
			if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
				return err
			}
		}
		for _, p := range []LobbyPlayerRef{p1, p2} {
			var row dao.PlayerQueueEntry
			err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("player_id = ? AND temp_address = ?", p.PlayerID, p.TempAddress).First(&row).Error
			if errors.Is(err, gorm.ErrRecordNotFound) {
				nu := dao.PlayerQueueEntry{
					PlayerID:    p.PlayerID,
					TempAddress: p.TempAddress,
					GameMatchID: matchID,
				}
				if e := tx.Create(&nu).Error; e != nil {
					return e
				}
				continue
			}
			if err != nil {
				return err
			}
			if row.GameMatchID == matchID {
				continue
			}
			if row.GameMatchID != 0 {
				return nil
			}
			if e := tx.Model(&row).Update("game_match_id", matchID).Error; e != nil {
				return e
			}
		}
		ok = true
		return nil
	})
	return ok, err
}

// LobbyCancelPendingPair removes lobby rows for both players when both reference matchID.
func LobbyCancelPendingPair(ctx context.Context, matchID int64, p1, p2 LobbyPlayerRef) (bool, error) {
	p1.TempAddress = normLobbyTemp(p1.TempAddress)
	p2.TempAddress = normLobbyTemp(p2.TempAddress)
	var ok bool
	err := Get().WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var r1, r2 dao.PlayerQueueEntry
		if e := tx.Where("player_id = ? AND temp_address = ?", p1.PlayerID, p1.TempAddress).First(&r1).Error; e != nil {
			if errors.Is(e, gorm.ErrRecordNotFound) {
				return nil
			}
			return e
		}
		if e := tx.Where("player_id = ? AND temp_address = ?", p2.PlayerID, p2.TempAddress).First(&r2).Error; e != nil {
			if errors.Is(e, gorm.ErrRecordNotFound) {
				return nil
			}
			return e
		}
		if r1.GameMatchID != matchID || r2.GameMatchID != matchID {
			return nil
		}
		if e := tx.Unscoped().Delete(&dao.PlayerQueueEntry{}, r1.ID).Error; e != nil {
			return e
		}
		if e := tx.Unscoped().Delete(&dao.PlayerQueueEntry{}, r2.ID).Error; e != nil {
			return e
		}
		ok = true
		return nil
	})
	return ok, err
}

// LobbyFinalizeConfirmedPair moves a confirmed pair from pending queue rows into player_game_entries.
func LobbyFinalizeConfirmedPair(ctx context.Context, matchID int64, p1, p2 LobbyPlayerRef) (bool, error) {
	p1.TempAddress = normLobbyTemp(p1.TempAddress)
	p2.TempAddress = normLobbyTemp(p2.TempAddress)
	var m dao.GameMatch
	if err := Get().WithContext(ctx).First(&m, "id = ?", matchID).Error; err != nil {
		return false, err
	}
	if m.GameID == nil || *m.GameID == 0 {
		return false, fmt.Errorf("game_match %d: game_id not set", matchID)
	}
	gid := *m.GameID

	var ok bool
	err := Get().WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var r1, r2 dao.PlayerQueueEntry
		if e := tx.Where("player_id = ? AND temp_address = ?", p1.PlayerID, p1.TempAddress).First(&r1).Error; e != nil {
			if errors.Is(e, gorm.ErrRecordNotFound) {
				return nil
			}
			return e
		}
		if e := tx.Where("player_id = ? AND temp_address = ?", p2.PlayerID, p2.TempAddress).First(&r2).Error; e != nil {
			if errors.Is(e, gorm.ErrRecordNotFound) {
				return nil
			}
			return e
		}
		if r1.GameMatchID != matchID || r2.GameMatchID != matchID {
			return nil
		}
		if e := tx.Unscoped().Delete(&dao.PlayerQueueEntry{}, r1.ID).Error; e != nil {
			return e
		}
		if e := tx.Unscoped().Delete(&dao.PlayerQueueEntry{}, r2.ID).Error; e != nil {
			return e
		}
		for _, p := range []LobbyPlayerRef{p1, p2} {
			ge := dao.PlayerGameEntry{
				PlayerID:    p.PlayerID,
				TempAddress: p.TempAddress,
				GameID:      gid,
			}
			if e := tx.Create(&ge).Error; e != nil {
				return e
			}
		}
		ok = true
		return nil
	})
	return ok, err
}

// LobbyMarkPlayersInGame upserts player_game_entries for each player with the given gameID.
func LobbyMarkPlayersInGame(ctx context.Context, gameID int64, players []LobbyPlayerRef) error {
	if len(players) == 0 {
		return nil
	}
	if gameID == 0 {
		return fmt.Errorf("lobby mark in game: gameID required")
	}
	for _, pl := range players {
		p := pl
		p.TempAddress = normLobbyTemp(p.TempAddress)
		var row dao.PlayerGameEntry
		err := Get().WithContext(ctx).Where("player_id = ? AND temp_address = ?", p.PlayerID, p.TempAddress).First(&row).Error
		if errors.Is(err, gorm.ErrRecordNotFound) {
			ge := dao.PlayerGameEntry{
				PlayerID:    p.PlayerID,
				TempAddress: p.TempAddress,
				GameID:      gameID,
			}
			if e := Get().WithContext(ctx).Create(&ge).Error; e != nil {
				return e
			}
			continue
		}
		if err != nil {
			return err
		}
		if e := Get().WithContext(ctx).Model(&row).Update("game_id", gameID).Error; e != nil {
			return e
		}
	}
	return nil
}

// LobbyMarkPlayersOutOfGame deletes in-game rows for the given players.
func LobbyMarkPlayersOutOfGame(ctx context.Context, players []LobbyPlayerRef) error {
	for _, pl := range players {
		p := pl
		p.TempAddress = normLobbyTemp(p.TempAddress)
		if err := Get().WithContext(ctx).Unscoped().
			Where("player_id = ? AND temp_address = ?", p.PlayerID, p.TempAddress).
			Delete(&dao.PlayerGameEntry{}).Error; err != nil {
			return err
		}
	}
	return nil
}

// LobbyIsInGame reports whether the player has a player_game_entries row.
func LobbyIsInGame(ctx context.Context, playerID int64, tempAddress string) (bool, error) {
	tempAddress = normLobbyTemp(tempAddress)
	var n int64
	err := Get().WithContext(ctx).Model(&dao.PlayerGameEntry{}).
		Where("player_id = ? AND temp_address = ?", playerID, tempAddress).
		Count(&n).Error
	return n > 0, err
}

// LobbyPendingMatchID returns the pending game_match id for the player when game_match.status is still pending.
func LobbyPendingMatchID(ctx context.Context, playerID int64, tempAddress string) (int64, bool, error) {
	tempAddress = normLobbyTemp(tempAddress)
	var row dao.PlayerQueueEntry
	if err := Get().WithContext(ctx).Where("player_id = ? AND temp_address = ?", playerID, tempAddress).First(&row).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return 0, false, nil
		}
		return 0, false, err
	}
	if row.GameMatchID == 0 {
		return 0, false, nil
	}
	var m dao.GameMatch
	if err := Get().WithContext(ctx).First(&m, row.GameMatchID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return 0, false, nil
		}
		return 0, false, err
	}
	if m.Status != dao.GameMatchStatusPending {
		return 0, false, nil
	}
	return row.GameMatchID, true, nil
}

// LobbyJoinQueueOrGetMatchCandidate either pairs the player with an eligible queued opponent or enqueues the player.
// On match, the candidate is removed from the queue and returned; queued is false. On enqueue, candidate is nil and queued is true.
func LobbyJoinQueueOrGetMatchCandidate(ctx context.Context, playerID int64, tempAddress string, nowMs int64) (candidate *LobbyPlayerRef, queued bool, err error) {
	tempAddress = normLobbyTemp(tempAddress)
	err = Get().WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var ingame int64
		if e := tx.Model(&dao.PlayerGameEntry{}).Where("player_id = ? AND temp_address = ?", playerID, tempAddress).Count(&ingame).Error; e != nil {
			return e
		}
		if ingame > 0 {
			return nil
		}

		var self dao.PlayerQueueEntry
		errSelf := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("player_id = ? AND temp_address = ?", playerID, tempAddress).First(&self).Error
		if errSelf == nil && self.GameMatchID != 0 {
			return nil
		}
		if errSelf != nil && !errors.Is(errSelf, gorm.ErrRecordNotFound) {
			return errSelf
		}

		var candidates []dao.PlayerQueueEntry
		if e := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("game_match_id = ?", 0).Order("created_at ASC").Limit(100).Find(&candidates).Error; e != nil {
			return e
		}

		for _, c := range candidates {
			if c.PlayerID == playerID || c.TempAddress == tempAddress {
				continue
			}
			res := tx.Unscoped().Where("id = ? AND game_match_id = ?", c.ID, 0).Delete(&dao.PlayerQueueEntry{})
			if res.Error != nil {
				return res.Error
			}
			if res.RowsAffected == 1 {
				candidate = &LobbyPlayerRef{PlayerID: c.PlayerID, TempAddress: c.TempAddress}
				return nil
			}
		}

		t := time.UnixMilli(nowMs)
		if errors.Is(errSelf, gorm.ErrRecordNotFound) {
			row := &dao.PlayerQueueEntry{
				PlayerID:    playerID,
				TempAddress: tempAddress,
				GameMatchID: 0,
			}
			row.CreatedAt = t
			row.UpdatedAt = t
			if e := tx.Create(row).Error; e != nil {
				return e
			}
		} else {
			if e := tx.Model(&self).Updates(map[string]interface{}{
				"created_at": t,
				"updated_at": t,
			}).Error; e != nil {
				return e
			}
		}
		queued = true
		return nil
	})
	if err != nil {
		return nil, false, err
	}
	return candidate, queued, nil
}

// LobbyFirstWaitingPlayerBefore returns the longest-waiting queued player whose created_at is at or before cutoffMs (Unix ms).
func LobbyFirstWaitingPlayerBefore(ctx context.Context, cutoffMs int64) (*LobbyPlayerRef, error) {
	cutoff := time.UnixMilli(cutoffMs)
	var row dao.PlayerQueueEntry
	err := Get().WithContext(ctx).
		Where("game_match_id = ? AND created_at <= ?", 0, cutoff).
		Order("created_at ASC").
		First(&row).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &LobbyPlayerRef{PlayerID: row.PlayerID, TempAddress: row.TempAddress}, nil
}
