package db

import (
	"errors"
	"fmt"
	"sort"
	"strings"

	dao "github.com/CryptoElementals/common/models"
	"gorm.io/gorm"
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

// ChainTxPoolPendingRow is one queued item for a chain, in flush order.
type ChainTxPoolPendingRow struct {
	ID      uint
	Kind    uint8
	Payload []byte
}

// chainItemsToPendingRowsInFlushOrder orders items for one chain: create → set turn → commitment →
// card, with id order within each kind (matches the legacy in-memory pool).
func chainItemsToPendingRowsInFlushOrder(items []dao.ChainTxPoolItem) []ChainTxPoolPendingRow {
	var byKind [5][]dao.ChainTxPoolItem
	for i := range items {
		k := items[i].Kind
		if k < 1 || k > 4 {
			continue
		}
		byKind[k] = append(byKind[k], items[i])
	}
	for k := 1; k <= 4; k++ {
		s := byKind[k]
		sort.Slice(s, func(i, j int) bool { return s[i].ID < s[j].ID })
	}
	var out []ChainTxPoolPendingRow
	for _, k := range []uint8{
		dao.ChainTxPoolKindCreateRoom,
		dao.ChainTxPoolKindSetTurnReady,
		dao.ChainTxPoolKindCommitment,
		dao.ChainTxPoolKindCard,
	} {
		for j := range byKind[k] {
			r := &byKind[k][j]
			out = append(out, ChainTxPoolPendingRow{ID: r.ID, Kind: r.Kind, Payload: append([]byte(nil), r.Payload...)})
		}
	}
	return out
}

// ListChainTxPoolPendingForChain returns pending items for a chain in flush order.
func ListChainTxPoolPendingForChain(chainID int64) ([]ChainTxPoolPendingRow, error) {
	var items []dao.ChainTxPoolItem
	if err := Get().Where("chain_id = ?", chainID).Find(&items).Error; err != nil {
		return nil, err
	}
	return chainItemsToPendingRowsInFlushOrder(items), nil
}

// ListChainTxPoolPendingByChain loads all non-deleted pool items in one query, classifies by chain_id,
// and returns each chain's rows in the same flush order as ListChainTxPoolPendingForChain.
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
		out[cid] = chainItemsToPendingRowsInFlushOrder(items)
	}
	return out, nil
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
