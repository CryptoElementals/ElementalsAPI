package db

import (
	"errors"
	"fmt"
	"strings"

	dao "github.com/CryptoElementals/common/models"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// ErrChainTxPoolDuplicate is returned when inserting a row that violates the unique natural key.
var ErrChainTxPoolDuplicate = errors.New("chain tx pool: duplicate key")

func isChainTxPoolDuplicateErr(err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, gorm.ErrDuplicatedKey) {
		return true
	}
	s := err.Error()
	return strings.Contains(strings.ToLower(s), "duplicate") ||
		strings.Contains(s, "UNIQUE constraint failed")
}

// InsertChainTxPoolItem inserts a row. Returns ErrChainTxPoolDuplicate on unique violation.
func InsertChainTxPoolItem(row *dao.ChainTxPoolItem) error {
	if row == nil {
		return fmt.Errorf("nil item")
	}
	err := Get().Create(row).Error
	if err != nil {
		if isChainTxPoolDuplicateErr(err) {
			return ErrChainTxPoolDuplicate
		}
		return err
	}
	return nil
}

// ChainTxPoolPendingRow is one queued item for a chain.
type ChainTxPoolPendingRow struct {
	ID                  uint
	Kind                uint8
	GameID              int64
	PlayerTemporaryAddr string
	RoundNumber         uint32
	TurnNumber          uint32
	Payload             []byte
}

func chainItemsToPendingRows(items []dao.ChainTxPoolItem) []ChainTxPoolPendingRow {
	var out []ChainTxPoolPendingRow
	for i := range items {
		k := items[i].Kind
		if k < 1 || k > 4 {
			continue
		}
		out = append(out, ChainTxPoolPendingRow{
			ID:                  items[i].ID,
			Kind:                items[i].Kind,
			GameID:              items[i].GameID,
			PlayerTemporaryAddr: items[i].PlayerTemporaryAddr,
			RoundNumber:         items[i].RoundNumber,
			TurnNumber:          items[i].TurnNumber,
			Payload:             append([]byte(nil), items[i].Payload...),
		})
	}
	return out
}

// ListChainTxPoolPendingForChain returns pending items for a chain.
func ListChainTxPoolPendingForChain(chainID int64) ([]ChainTxPoolPendingRow, error) {
	var items []dao.ChainTxPoolItem
	if err := Get().Where("chain_id = ?", chainID).Find(&items).Error; err != nil {
		return nil, err
	}
	return chainItemsToPendingRows(items), nil
}

// ListChainTxPoolPendingByChain loads all non-deleted pool items in one query, grouped by chain_id.
func ListChainTxPoolPendingByChain() (map[int64][]ChainTxPoolPendingRow, error) {
	var all []dao.ChainTxPoolItem
	if err := Get().Find(&all).Error; err != nil {
		return nil, err
	}
	byItems := make(map[int64][]dao.ChainTxPoolItem)
	for i := range all {
		cid := all[i].ChainID
		byItems[cid] = append(byItems[cid], all[i])
	}
	out := make(map[int64][]ChainTxPoolPendingRow, len(byItems))
	for cid, items := range byItems {
		out[cid] = chainItemsToPendingRows(items)
	}
	return out, nil
}

// PopChainTxPoolBatchForChain atomically locks, loads, deletes, and returns one batch for a chain.
func PopChainTxPoolBatchForChain(chainID int64, limit int) ([]ChainTxPoolPendingRow, error) {
	if chainID == 0 {
		return nil, nil
	}
	if limit <= 0 {
		limit = 1
	}

	var popped []ChainTxPoolPendingRow
	err := Get().Transaction(func(tx *gorm.DB) error {
		var items []dao.ChainTxPoolItem
		if err := tx.
			Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("chain_id = ?", chainID).
			Order("id ASC").
			Limit(limit).
			Find(&items).Error; err != nil {
			return err
		}
		if len(items) == 0 {
			return nil
		}

		popped = chainItemsToPendingRows(items)
		if len(popped) == 0 {
			return nil
		}

		ids := make([]uint, 0, len(popped))
		for i := range popped {
			ids = append(ids, popped[i].ID)
		}
		return tx.Where("id IN ?", ids).Delete(&dao.ChainTxPoolItem{}).Error
	})
	if err != nil {
		return nil, err
	}
	return popped, nil
}

// DeleteChainTxPoolItemsByIDs removes submitted pool rows.
func DeleteChainTxPoolItemsByIDs(ids []uint) error {
	if len(ids) == 0 {
		return nil
	}
	return Get().Where("id IN ?", ids).Delete(&dao.ChainTxPoolItem{}).Error
}

// DeleteChainTxPoolItemsForGame soft-deletes all pool items for a game.
func DeleteChainTxPoolItemsForGame(gameID int64) error {
	if gameID == 0 {
		return nil
	}
	return Get().Where("game_id = ?", gameID).Delete(&dao.ChainTxPoolItem{}).Error
}
