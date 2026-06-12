package db

import (
	"errors"
	"fmt"
	"strings"
	"time"

	dao "github.com/CryptoElementals/common/models"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

const DefaultChainTxPoolClaimTimeout = 2 * time.Second

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

func chainTxPoolNaturalKeyQuery(tx *gorm.DB, row *dao.ChainTxPoolItem) *gorm.DB {
	return tx.Model(&dao.ChainTxPoolItem{}).
		Where("game_id = ? AND kind = ? AND player_temporary_addr = ? AND round_number = ? AND turn_number = ?",
			row.GameID, row.Kind, row.PlayerTemporaryAddr, row.RoundNumber, row.TurnNumber)
}

func InsertChainTxPoolItem(row *dao.ChainTxPoolItem) error {
	if row == nil {
		return fmt.Errorf("nil item")
	}
	var existing dao.ChainTxPoolItem
	err := chainTxPoolNaturalKeyQuery(Get(), row).Limit(1).Take(&existing).Error
	if err == nil {
		return ErrChainTxPoolDuplicate
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return err
	}
	err = Get().Create(row).Error
	if err != nil {
		if isChainTxPoolDuplicateErr(err) {
			return ErrChainTxPoolDuplicate
		}
		return err
	}
	return nil
}

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

func ListChainTxPoolPendingForChain(chainID int64) ([]ChainTxPoolPendingRow, error) {
	var items []dao.ChainTxPoolItem
	if err := Get().Where("chain_id = ? AND claimed_at IS NULL", chainID).Find(&items).Error; err != nil {
		return nil, err
	}
	return chainItemsToPendingRows(items), nil
}

func ListChainTxPoolPendingByChain() (map[int64][]ChainTxPoolPendingRow, error) {
	var all []dao.ChainTxPoolItem
	if err := Get().Where("claimed_at IS NULL").Find(&all).Error; err != nil {
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

func ClaimChainTxPoolBatchForChain(chainID int64, limit int, claimTimeout time.Duration) ([]ChainTxPoolPendingRow, error) {
	if chainID == 0 {
		return nil, nil
	}
	if limit <= 0 {
		limit = 1
	}
	if claimTimeout <= 0 {
		claimTimeout = DefaultChainTxPoolClaimTimeout
	}
	staleBefore := time.Now().Add(-claimTimeout)

	var claimed []ChainTxPoolPendingRow
	err := Get().Transaction(func(tx *gorm.DB) error {
		var items []dao.ChainTxPoolItem
		if err := tx.
			Clauses(clause.Locking{Strength: "UPDATE", Options: "SKIP LOCKED"}).
			Where("chain_id = ? AND (claimed_at IS NULL OR claimed_at < ?)", chainID, staleBefore).
			Order("id ASC").
			Limit(limit).
			Find(&items).Error; err != nil {
			return err
		}
		if len(items) == 0 {
			return nil
		}

		now := time.Now()
		ids := make([]uint, 0, len(items))
		for i := range items {
			ids = append(ids, items[i].ID)
			items[i].ClaimedAt = &now
		}
		if err := tx.Model(&dao.ChainTxPoolItem{}).
			Where("id IN ?", ids).
			Update("claimed_at", now).Error; err != nil {
			return err
		}

		claimed = chainItemsToPendingRows(items)
		return nil
	})
	if err != nil {
		return nil, err
	}
	return claimed, nil
}

func DeleteChainTxPoolItemsByIDs(ids []uint) error {
	if len(ids) == 0 {
		return nil
	}
	return Get().Where("id IN ?", ids).Delete(&dao.ChainTxPoolItem{}).Error
}

func ReleaseChainTxPoolClaims(ids []uint) error {
	if len(ids) == 0 {
		return nil
	}
	return Get().Model(&dao.ChainTxPoolItem{}).
		Where("id IN ?", ids).
		Update("claimed_at", nil).Error
}

func ReleaseStaleChainTxPoolClaims(cutoff time.Time) error {
	return Get().Model(&dao.ChainTxPoolItem{}).
		Where("claimed_at IS NOT NULL AND claimed_at < ?", cutoff).
		Update("claimed_at", nil).Error
}

func DeleteChainTxPoolItemsForGame(gameID int64) error {
	if gameID == 0 {
		return nil
	}
	return Get().Where("game_id = ?", gameID).Delete(&dao.ChainTxPoolItem{}).Error
}

func DropLegacyChainTxPoolIndexes() error {
	m := Get().Migrator()
	item := &dao.ChainTxPoolItem{}
	if m.HasIndex(item, "ux_chain_tx_pool_natural") {
		if err := m.DropIndex(item, "ux_chain_tx_pool_natural"); err != nil {
			return err
		}
	}
	if m.HasIndex(item, "idx_chain_tx_pool_items_chain_id") {
		if err := m.DropIndex(item, "idx_chain_tx_pool_items_chain_id"); err != nil {
			return err
		}
	}
	return nil
}
