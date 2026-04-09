package game

import (
	"github.com/CryptoElementals/common/db"
	"github.com/CryptoElementals/common/log"
	dao "github.com/CryptoElementals/common/models"
	"github.com/CryptoElementals/common/rpc/proto"
	"gorm.io/gorm"
)

// runGamePersist runs fn with either the outer mutation tx (executeOnGame) or a new transaction.
func (g *Game) runGamePersist(fn func(tx *gorm.DB) error) error {
	if g.mutateTx != nil {
		return fn(g.mutateTx)
	}
	return db.GamePersistTx(fn)
}

// persistInsertNewGameGraph inserts a brand-new match graph (Phase 2 granular persistence).
func (g *Game) persistInsertNewGameGraph() error {
	if err := db.InsertNewGameGraphCommit(g.gameInfo); err != nil {
		log.Errorw("persistInsertNewGameGraph failed", "err", err)
		return err
	}
	return nil
}

// persistCurrentTurn saves only the current turn row (turn_status, turn_start_at).
func (g *Game) persistCurrentTurn() error {
	t := g.currentRound.getCurrentTurn()
	if t == nil {
		return nil
	}
	if err := g.runGamePersist(func(tx *gorm.DB) error {
		return db.SaveTurnTx(tx, t)
	}); err != nil {
		log.Errorw("persistCurrentTurn failed", "err", err, "game id", g.gameInfo.ID)
		return err
	}
	return nil
}

// persistPlayerTurnInfo saves a single PTI row.
func (g *Game) persistPlayerTurnInfo(pti *dao.PlayerTurnInfo) error {
	if err := g.runGamePersist(func(tx *gorm.DB) error {
		return db.SavePlayerTurnInfoTx(tx, pti)
	}); err != nil {
		log.Errorw("persistPlayerTurnInfo failed", "err", err, "game id", g.gameInfo.ID)
		return err
	}
	return nil
}

// persistCommitmentStep saves one PTI and optionally updates the current turn to waiting cards.
func (g *Game) persistCommitmentStep(pti *dao.PlayerTurnInfo, transitionTurnToWaitingCards bool) error {
	return g.runGamePersist(func(tx *gorm.DB) error {
		if err := db.SavePlayerTurnInfoTx(tx, pti); err != nil {
			return err
		}
		if !transitionTurnToWaitingCards {
			return nil
		}
		cur := g.currentRound.getCurrentTurn()
		if cur == nil {
			return nil
		}
		return db.SaveTurnTx(tx, cur)
	})
}

// persistConfirmBattleAllReady saves game status (first round only) and current turn phase.
func (g *Game) persistConfirmBattleAllReady() error {
	return g.runGamePersist(func(tx *gorm.DB) error {
		if g.currentRound.roundNumber == 1 && g.currentRound.getCurrentTurnNumber() == 1 {
			st := proto.GameStatus_GAME_RUNNING
			if err := db.UpdateGameFieldsTx(tx, g.gameInfo.ID, db.GameFieldsUpdate{Status: &st}); err != nil {
				return err
			}
		}
		cur := g.currentRound.getCurrentTurn()
		if cur == nil {
			return nil
		}
		return db.SaveTurnTx(tx, cur)
	})
}

// persistTurnEndGameOver saves final turn + PTIs, game status, and result tree in one transaction.
func (g *Game) persistTurnEndGameOver() error {
	turn := g.currentRound.getCurrentTurn()
	if turn == nil {
		return nil
	}
	st := proto.GameStatus_GAME_END
	err := g.runGamePersist(func(tx *gorm.DB) error {
		for _, pti := range turn.PlayerTurnInfos {
			if pti == nil {
				continue
			}
			if err := db.SavePlayerTurnInfoTx(tx, pti); err != nil {
				return err
			}
		}
		if err := db.SaveTurnTx(tx, turn); err != nil {
			return err
		}
		if err := db.UpdateGameFieldsTx(tx, g.gameInfo.ID, db.GameFieldsUpdate{Status: &st}); err != nil {
			return err
		}
		if g.gameInfo.GameResult != nil {
			if err := db.SaveGameResultTreeTx(tx, g.gameInfo.ID, g.gameInfo.GameResult); err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		log.Errorw("persistTurnEndGameOver failed", "err", err, "game id", g.gameInfo.ID)
	}
	return err
}

// persistCompletedTurnAndNewTurn saves a finished turn (with PTIs), then inserts the next turn and its PTIs.
func (g *Game) persistCompletedTurnAndNewTurn(completed, fresh *dao.Turn) error {
	if completed == nil || fresh == nil {
		return nil
	}
	return g.runGamePersist(func(tx *gorm.DB) error {
		for _, pti := range completed.PlayerTurnInfos {
			if pti == nil {
				continue
			}
			if err := db.SavePlayerTurnInfoTx(tx, pti); err != nil {
				return err
			}
		}
		if err := db.SaveTurnTx(tx, completed); err != nil {
			return err
		}
		if err := db.SaveTurnTx(tx, fresh); err != nil {
			return err
		}
		for _, pti := range fresh.PlayerTurnInfos {
			if pti == nil {
				continue
			}
			pti.TurnID = fresh.ID
			if err := db.SavePlayerTurnInfoTx(tx, pti); err != nil {
				return err
			}
		}
		return nil
	})
}

// persistAbortInit sets game to aborted and stores the result tree (no turn row update).
func (g *Game) persistAbortInit() error {
	st := proto.GameStatus_GAME_ABORTED
	return g.runGamePersist(func(tx *gorm.DB) error {
		if err := db.UpdateGameFieldsTx(tx, g.gameInfo.ID, db.GameFieldsUpdate{Status: &st}); err != nil {
			return err
		}
		if g.gameInfo.GameResult != nil {
			return db.SaveGameResultTreeTx(tx, g.gameInfo.ID, g.gameInfo.GameResult)
		}
		return nil
	})
}

// persistAbortInternal saves current turn, game aborted status, and result tree.
func (g *Game) persistAbortInternal() error {
	st := proto.GameStatus_GAME_ABORTED
	return g.runGamePersist(func(tx *gorm.DB) error {
		cur := g.currentRound.getCurrentTurn()
		if cur != nil {
			if err := db.SaveTurnTx(tx, cur); err != nil {
				return err
			}
		}
		if err := db.UpdateGameFieldsTx(tx, g.gameInfo.ID, db.GameFieldsUpdate{Status: &st}); err != nil {
			return err
		}
		if g.gameInfo.GameResult != nil {
			return db.SaveGameResultTreeTx(tx, g.gameInfo.ID, g.gameInfo.GameResult)
		}
		return nil
	})
}
