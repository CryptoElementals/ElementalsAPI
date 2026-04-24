package db

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	dao "github.com/CryptoElementals/common/models"
	"github.com/CryptoElementals/common/snowflake"
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

func lobbyNormalizePairRef(a, b LobbyPlayerRef) (LobbyPlayerRef, LobbyPlayerRef) {
	if a.PlayerID < b.PlayerID || (a.PlayerID == b.PlayerID && a.TempAddress < b.TempAddress) {
		return a, b
	}
	return b, a
}

// lobbyMatchPlayersTx inserts a pending GameMatch for two players, then updates or inserts their PlayerQueueEntry rows
// to game_match_id (bots usually have no row yet and get an insert; humans in the queue get an update).
// lastGameID non-zero sets game_match.last_game_id (e.g. continue rematch); zero leaves the DB default.
func lobbyMatchPlayersTx(tx *gorm.DB, playerA, playerB LobbyPlayerRef, gameType uint, lastGameID int64) (*dao.GameMatch, error) {
	p1, p2 := lobbyNormalizePairRef(playerA, playerB)
	m := &dao.GameMatch{
		ID:                 snowflake.GenerateID(),
		Player1ID:          p1.PlayerID,
		Player1TempAddress: p1.TempAddress,
		Player2ID:          p2.PlayerID,
		Player2TempAddress: p2.TempAddress,
		GameType:           gameType,
		Status:             dao.GameMatchStatusPending,
	}
	if lastGameID != 0 {
		m.LastGameID = lastGameID
	}
	if e := tx.Create(m).Error; e != nil {
		return nil, e
	}
	for _, p := range []LobbyPlayerRef{p1, p2} {
		var row dao.PlayerQueueEntry
		err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("player_id = ? AND temp_address = ?", p.PlayerID, p.TempAddress).First(&row).Error
		if err == nil && row.GameMatchID != 0 && row.GameMatchID != m.ID {
			return nil, fmt.Errorf("lobby: player %d already pending match %d", p.PlayerID, row.GameMatchID)
		}
		if errors.Is(err, gorm.ErrRecordNotFound) {
			nu := dao.PlayerQueueEntry{
				PlayerID:    p.PlayerID,
				TempAddress: p.TempAddress,
				GameMatchID: m.ID,
			}
			if e := tx.Create(&nu).Error; e != nil {
				return nil, e
			}
			continue
		}
		if err != nil {
			return nil, err
		}
		if e := tx.Model(&row).Update("game_match_id", m.ID).Error; e != nil {
			return nil, e
		}
	}
	return m, nil
}

// LobbyGetGameIDByPlayer returns whether the player has a player_game_entries row and its game_id.
func LobbyGetGameIDByPlayer(ctx context.Context, playerID int64, tempAddress string) (ok bool, gameID int64, err error) {
	tempAddress = normLobbyTemp(tempAddress)
	var row dao.PlayerGameEntry
	err = Get().WithContext(ctx).Where("player_id = ? AND temp_address = ?", playerID, tempAddress).First(&row).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return false, 0, nil
	}
	if err != nil {
		return false, 0, err
	}
	return true, row.GameID, nil
}

// LobbyMatchPlayersOrJoinQueue: (1) reject if already queued or pending, (2) reject if in-game,
// (3) insert a queue-only row, (4) look for an eligible opponent, (5) if found create game_match and set both rows to pending.
// Returns the new pending game_match row when a pair is formed, or nil when the player is only in the queue.
func LobbyMatchPlayersOrJoinQueue(ctx context.Context, playerID int64, tempAddress string, gameType uint) (*dao.GameMatch, error) {
	tempAddress = normLobbyTemp(tempAddress)
	nowMs := time.Now().UnixMilli()
	var out *dao.GameMatch
	err := Get().WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var existing dao.PlayerQueueEntry
		errPQ := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("player_id = ? AND temp_address = ?", playerID, tempAddress).
			First(&existing).Error
		if errPQ == nil {
			if existing.GameMatchID == 0 {
				return fmt.Errorf("lobby: player already in queue")
			}
			return fmt.Errorf("lobby: player already in pending match")
		}
		if !errors.Is(errPQ, gorm.ErrRecordNotFound) {
			return errPQ
		}

		var ingame int64
		if e := tx.Model(&dao.PlayerGameEntry{}).Where("player_id = ? AND temp_address = ?", playerID, tempAddress).Count(&ingame).Error; e != nil {
			return e
		}
		if ingame > 0 {
			return fmt.Errorf("lobby: player already in game")
		}

		t := time.UnixMilli(nowMs)
		joinerRow := &dao.PlayerQueueEntry{
			PlayerID:    playerID,
			TempAddress: tempAddress,
			GameMatchID: 0,
		}
		joinerRow.CreatedAt = t
		joinerRow.UpdatedAt = t
		if e := tx.Create(joinerRow).Error; e != nil {
			return e
		}

		joiner := LobbyPlayerRef{PlayerID: playerID, TempAddress: tempAddress}
		var opp dao.PlayerQueueEntry
		q := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("game_match_id = ? AND player_id <> ? AND temp_address <> ?", 0, playerID, tempAddress).
			Order("created_at ASC")
		if e := q.First(&opp).Error; e != nil {
			if errors.Is(e, gorm.ErrRecordNotFound) {
				return nil
			}
			return e
		}

		cand := LobbyPlayerRef{PlayerID: opp.PlayerID, TempAddress: opp.TempAddress}
		m, e := lobbyMatchPlayersTx(tx, joiner, cand, gameType, 0)
		if e != nil {
			return e
		}
		out = m
		return nil
	})
	return out, err
}

// LobbyCountLongWaitingQueuedPlayers counts queue-only rows whose created_at is at or before (now - minWait).
func LobbyCountLongWaitingQueuedPlayers(ctx context.Context, minWait time.Duration) (int, error) {
	if minWait < 0 {
		minWait = 0
	}
	cutoff := time.Now().Add(-minWait)
	var n int64
	err := Get().WithContext(ctx).Model(&dao.PlayerQueueEntry{}).
		Where("game_match_id = ? AND created_at <= ?", 0, cutoff).
		Count(&n).Error
	return int(n), err
}

// LobbyMatchEarliestQueuedPlayerWithBot pairs the long-waiting human who has been in queue longest (earliest
// created_at among rows with game_match_id=0 and created_at <= now-minWait) with a single bot. Bots never have
// PlayerQueueEntry. Returns the new pending game_match row, or nil if no human row qualifies.
func LobbyMatchEarliestQueuedPlayerWithBot(ctx context.Context, bot LobbyPlayerRef, gameType uint, minWait time.Duration) (*dao.GameMatch, error) {
	bot.TempAddress = normLobbyTemp(bot.TempAddress)
	if minWait < 0 {
		minWait = 0
	}
	var out *dao.GameMatch
	err := Get().WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		cutoff := time.Now().Add(-minWait)
		var entry dao.PlayerQueueEntry
		if e := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("game_match_id = ? AND created_at <= ?", 0, cutoff).
			Order("created_at ASC").
			First(&entry).Error; e != nil {
			if errors.Is(e, gorm.ErrRecordNotFound) {
				return nil
			}
			return e
		}
		human := LobbyPlayerRef{PlayerID: entry.PlayerID, TempAddress: entry.TempAddress}
		m, e := lobbyMatchPlayersTx(tx, human, bot, gameType, 0)
		if e != nil {
			return e
		}
		out = m
		return nil
	})
	return out, err
}

// LobbyMatchPair creates a pending GameMatch for two players and updates or inserts their player_queue_entries
// to that match id. Returns the new pending game_match row. lastGameID non-zero is stored (continue rematch).
func LobbyMatchPair(ctx context.Context, p1, p2 LobbyPlayerRef, gameType uint, lastGameID int64) (*dao.GameMatch, error) {
	p1.TempAddress = normLobbyTemp(p1.TempAddress)
	p2.TempAddress = normLobbyTemp(p2.TempAddress)
	var out *dao.GameMatch
	err := Get().WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		m, e := lobbyMatchPlayersTx(tx, p1, p2, gameType, lastGameID)
		if e != nil {
			return e
		}
		out = m
		return nil
	})
	return out, err
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
	return Get().WithContext(ctx).Where("player_id = ? AND temp_address = ? AND game_match_id = ?", playerID, tempAddress, 0).
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

// LobbyCancelPendingMatch marks game_match as cancelled and hard-deletes all player_queue_entries
// with that game_match_id. The match must exist and be in pending status.
func LobbyCancelPendingMatch(ctx context.Context, matchID int64) error {
	return Get().WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var m dao.GameMatch
		if err := tx.First(&m, "id = ?", matchID).Error; err != nil {
			return err
		}
		if m.Status != dao.GameMatchStatusPending {
			return fmt.Errorf("game_match %d: cancel requires pending status, got %s", matchID, m.Status)
		}
		res := tx.Model(&dao.GameMatch{}).Where("id = ? AND status = ?", matchID, dao.GameMatchStatusPending).
			Update("status", dao.GameMatchStatusCancelled)
		if res.Error != nil {
			return res.Error
		}
		if err := tx.Where("game_match_id = ?", matchID).Delete(&dao.PlayerQueueEntry{}).Error; err != nil {
			return err
		}
		return nil
	})
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
		if e := tx.Delete(&dao.PlayerQueueEntry{}, r1.ID).Error; e != nil {
			return e
		}
		if e := tx.Delete(&dao.PlayerQueueEntry{}, r2.ID).Error; e != nil {
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
		if err := Get().WithContext(ctx).
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
