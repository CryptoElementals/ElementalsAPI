package db

import (
	dao "github.com/CryptoElementals/common/models"
	"github.com/CryptoElementals/common/rpc/proto"
	"gorm.io/gorm"
)

// GamePersistTx runs fn inside a single SQL transaction (Phase 1 granular persistence).
func GamePersistTx(fn func(tx *gorm.DB) error) error {
	return Get().Transaction(fn)
}

// GameFieldsUpdate lists optional game row columns to patch. Nil pointer = omit field.
type GameFieldsUpdate struct {
	Status       *proto.GameStatus
	RoomContract *string
	Type         *uint
}

// UpdateGameFieldsTx updates whitelisted columns on games.
func UpdateGameFieldsTx(tx *gorm.DB, gameID int64, u GameFieldsUpdate) error {
	updates := map[string]interface{}{}
	if u.Status != nil {
		updates["status"] = *u.Status
	}
	if u.RoomContract != nil {
		updates["room_contract"] = *u.RoomContract
	}
	if u.Type != nil {
		updates["type"] = *u.Type
	}
	if len(updates) == 0 {
		return nil
	}
	return tx.Model(&dao.Game{}).Where("id = ?", gameID).Updates(updates).Error
}

// InsertNewGameGraph inserts games, game_player_infos, turns, and player_turn_infos in FK order.
// game.GameArgs must be non-nil and point at an existing game_args row (non-zero id); that row is not created here.
func InsertNewGameGraph(tx *gorm.DB, game *dao.Game) error {
	game.GameArgsID = game.GameArgs.ID
	if err := tx.Omit("GameArgs", "Players", "Turns", "GameResult").Create(game).Error; err != nil {
		return err
	}
	for _, p := range game.Players {
		if p == nil {
			continue
		}
		p.GameID = game.ID
		if err := tx.Create(p).Error; err != nil {
			return err
		}
	}
	for _, t := range game.Turns {
		if t == nil {
			continue
		}
		t.GameID = game.ID
		infos := t.PlayerTurnInfos
		t.PlayerTurnInfos = nil
		if err := tx.Create(t).Error; err != nil {
			return err
		}
		t.PlayerTurnInfos = infos
		for _, pti := range t.PlayerTurnInfos {
			if pti == nil {
				continue
			}
			pti.TurnID = t.ID
			if err := SavePlayerTurnInfoTx(tx, pti); err != nil {
				return err
			}
		}
	}
	return nil
}

// SaveTurnTx inserts a new turn (omit PTIs) or updates turn_status and turn_start_at only.
func SaveTurnTx(tx *gorm.DB, turn *dao.Turn) error {
	if turn.ID == 0 {
		infos := turn.PlayerTurnInfos
		turn.PlayerTurnInfos = nil
		err := tx.Create(turn).Error
		turn.PlayerTurnInfos = infos
		return err
	}
	return tx.Model(&dao.Turn{}).Where("id = ?", turn.ID).Updates(map[string]interface{}{
		"turn_status":   turn.TurnStatus,
		"turn_start_at": turn.TurnStartAt,
	}).Error
}

// SavePlayerTurnInfoTx creates or updates a player_turn_infos row (embedded submitted-card columns).
func SavePlayerTurnInfoTx(tx *gorm.DB, pti *dao.PlayerTurnInfo) error {
	if pti.ID == 0 {
		return tx.Session(&gorm.Session{FullSaveAssociations: false}).Create(pti).Error
	}
	return tx.Session(&gorm.Session{FullSaveAssociations: false}).Save(pti).Error
}

// SaveGameResultTreeTx inserts or updates game_results and nested battle_rewards / player_rewards.
// For insert (result.ID == 0), builds the tree explicitly to avoid broad association side effects.
func SaveGameResultTreeTx(tx *gorm.DB, gameID int64, result *dao.GameResult) error {
	result.GameID = gameID
	if result.ID != 0 {
		return tx.Session(&gorm.Session{FullSaveAssociations: true}).Save(result).Error
	}
	br := result.BattleReward
	result.BattleReward = nil
	if err := tx.Omit("BattleReward").Create(result).Error; err != nil {
		return err
	}
	if br == nil {
		result.BattleReward = nil
		return nil
	}
	br.GameResultID = result.ID
	prs := br.PlayerRewards
	br.PlayerRewards = nil
	if err := tx.Omit("PlayerRewards").Create(br).Error; err != nil {
		return err
	}
	for _, pr := range prs {
		if pr == nil {
			continue
		}
		pr.BattleRewardID = br.ID
		if err := tx.Create(pr).Error; err != nil {
			return err
		}
	}
	br.PlayerRewards = prs
	result.BattleReward = br
	return nil
}

// --- Convenience wrappers (single implicit transaction) ---

// InsertNewGameGraphCommit runs InsertNewGameGraph in a transaction.
func InsertNewGameGraphCommit(game *dao.Game) error {
	return GamePersistTx(func(tx *gorm.DB) error {
		return InsertNewGameGraph(tx, game)
	})
}

// UpdateGameFieldsCommit runs UpdateGameFieldsTx in a transaction.
func UpdateGameFieldsCommit(gameID int64, u GameFieldsUpdate) error {
	return GamePersistTx(func(tx *gorm.DB) error {
		return UpdateGameFieldsTx(tx, gameID, u)
	})
}

// SaveTurnCommit runs SaveTurnTx in a transaction.
func SaveTurnCommit(turn *dao.Turn) error {
	return GamePersistTx(func(tx *gorm.DB) error {
		return SaveTurnTx(tx, turn)
	})
}

// SavePlayerTurnInfoCommit runs SavePlayerTurnInfoTx in a transaction.
func SavePlayerTurnInfoCommit(pti *dao.PlayerTurnInfo) error {
	return GamePersistTx(func(tx *gorm.DB) error {
		return SavePlayerTurnInfoTx(tx, pti)
	})
}

// SaveGameResultTreeCommit runs SaveGameResultTreeTx in a transaction.
func SaveGameResultTreeCommit(gameID int64, result *dao.GameResult) error {
	return GamePersistTx(func(tx *gorm.DB) error {
		return SaveGameResultTreeTx(tx, gameID, result)
	})
}

// SaveFullGameGraph persists the entire dao.Game tree using GORM FullSaveAssociations.
//
// Intended for repair scripts, one-off migrations, tests, and admin tooling only.
// The room_server game worker must use the granular APIs above (InsertNewGameGraph, SaveTurnTx, …).
func SaveFullGameGraph(game *dao.Game) error {
	return Get().Session(&gorm.Session{FullSaveAssociations: true}).Save(game).Error
}

// GamePersistenceSnapshot is a compact view of persisted game shape for tests and diagnostics.
type GamePersistenceSnapshot struct {
	Status           proto.GameStatus
	TurnCount        int
	GamePlayerCount  int
	HasGameResult    bool
	GameResultID     uint
	LastTurnID       uint
	LastTurnPTICount int
}

// CaptureGamePersistenceSnapshot summarizes a loaded *dao.Game (nil-safe).
func CaptureGamePersistenceSnapshot(g *dao.Game) GamePersistenceSnapshot {
	if g == nil {
		return GamePersistenceSnapshot{}
	}
	s := GamePersistenceSnapshot{
		Status:          g.Status,
		TurnCount:       len(g.Turns),
		GamePlayerCount: len(g.Players),
	}
	if g.GameResult != nil {
		s.HasGameResult = true
		s.GameResultID = g.GameResult.ID
	}
	if len(g.Turns) > 0 {
		last := g.Turns[len(g.Turns)-1]
		if last != nil {
			s.LastTurnID = last.ID
			s.LastTurnPTICount = len(last.PlayerTurnInfos)
		}
	}
	return s
}
