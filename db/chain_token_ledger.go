package db

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/CryptoElementals/common/internal/chainamount"
	"github.com/CryptoElementals/common/log"
	dao "github.com/CryptoElementals/common/models"
	"gorm.io/gorm"
)

type ChainTokenEventApplyStatus string

const (
	ChainTokenEventApplyApplied   ChainTokenEventApplyStatus = "applied"
	ChainTokenEventApplyDuplicate ChainTokenEventApplyStatus = "duplicate"
	ChainTokenEventApplyRejected  ChainTokenEventApplyStatus = "rejected"
)

const chainTokenRejectInsufficientBalance = "insufficient_balance"

// ChainTokenEventInput is a parsed on-chain deposit or withdraw event.
type ChainTokenEventInput struct {
	ChainID          int64
	BlockNumber      uint64
	BlockHash        string
	Timestamp        uint64
	TxHash           string
	LogIndex         uint32
	CollectorAddress string
	EventType        dao.ChainTokenLedgerEventType
	PlayerID         int64
	AmountWei        string
	FromAddress      string
	ToAddress        string
	Operator         string
	NewCreditedWei   string
}

type ChainTokenEventApplyResult struct {
	Status     ChainTokenEventApplyStatus
	Message    string
	TokenDelta int32
	NewBalance int32
}

// ApplyChainTokenEvent records the event and updates user_tokens idempotently.
func ApplyChainTokenEvent(ctx context.Context, ev ChainTokenEventInput) (*ChainTokenEventApplyResult, error) {
	var result *ChainTokenEventApplyResult
	err := Get().WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var err error
		result, err = applyChainTokenEventTx(tx, ev)
		return err
	})
	if err != nil {
		return nil, err
	}
	return result, nil
}

func applyChainTokenEventTx(tx *gorm.DB, ev ChainTokenEventInput) (*ChainTokenEventApplyResult, error) {
	if err := validateChainTokenEventInput(ev); err != nil {
		return nil, err
	}

	normalizedTxHash := strings.ToLower(strings.TrimSpace(ev.TxHash))
	existing, err := findChainTokenLedger(tx, ev.ChainID, normalizedTxHash, ev.LogIndex)
	if err != nil {
		return nil, err
	}
	if existing != nil {
		return duplicateChainTokenResult(existing), nil
	}

	tokenDelta, err := chainamount.WeiToGameToken(ev.AmountWei)
	if err != nil {
		return nil, fmt.Errorf("convert amount wei: %w", err)
	}
	if remainder, remErr := chainamount.WeiToGameTokenRemainder(ev.AmountWei); remErr == nil && remainder.Sign() > 0 {
		log.Warnf("chain token event tx=%s log=%d has wei remainder %s after /10^15",
			normalizedTxHash, ev.LogIndex, remainder.String())
	}

	row := &dao.ChainTokenLedger{
		ChainID:          ev.ChainID,
		TxHash:           normalizedTxHash,
		LogIndex:         ev.LogIndex,
		BlockNumber:      ev.BlockNumber,
		BlockHash:        strings.ToLower(strings.TrimSpace(ev.BlockHash)),
		EventType:        ev.EventType,
		PlayerID:         ev.PlayerID,
		CollectorAddress: strings.ToLower(strings.TrimSpace(ev.CollectorAddress)),
		AmountWei:        strings.TrimSpace(ev.AmountWei),
		TokenDelta:       tokenDelta,
		Status:           dao.ChainTokenLedgerStatusApplied,
		FromAddress:      strings.ToLower(strings.TrimSpace(ev.FromAddress)),
		ToAddress:        strings.ToLower(strings.TrimSpace(ev.ToAddress)),
		Operator:         strings.ToLower(strings.TrimSpace(ev.Operator)),
		NewCreditedWei:   strings.TrimSpace(ev.NewCreditedWei),
	}

	switch ev.EventType {
	case dao.ChainTokenLedgerEventDeposit:
		if err := tx.Create(row).Error; err != nil {
			if isChainTokenLedgerDuplicateErr(err) {
				existing, findErr := findChainTokenLedger(tx, ev.ChainID, normalizedTxHash, ev.LogIndex)
				if findErr != nil {
					return nil, findErr
				}
				if existing != nil {
					return duplicateChainTokenResult(existing), nil
				}
			}
			return nil, err
		}
		newBalance, err := creditUserTokenTx(tx, ev.PlayerID, tokenDelta)
		if err != nil {
			return nil, err
		}
		return &ChainTokenEventApplyResult{
			Status:     ChainTokenEventApplyApplied,
			TokenDelta: tokenDelta,
			NewBalance: newBalance,
		}, nil

	case dao.ChainTokenLedgerEventWithdraw:
		if err := tx.Create(row).Error; err != nil {
			if isChainTokenLedgerDuplicateErr(err) {
				existing, findErr := findChainTokenLedger(tx, ev.ChainID, normalizedTxHash, ev.LogIndex)
				if findErr != nil {
					return nil, findErr
				}
				if existing != nil {
					return duplicateChainTokenResult(existing), nil
				}
			}
			return nil, err
		}
		newBalance, deducted, err := deductUserTokenTx(tx, ev.PlayerID, tokenDelta)
		if err != nil {
			return nil, err
		}
		if !deducted {
			if err := tx.Model(row).Updates(map[string]any{
				"status":        dao.ChainTokenLedgerStatusRejected,
				"reject_reason": chainTokenRejectInsufficientBalance,
			}).Error; err != nil {
				return nil, err
			}
			return &ChainTokenEventApplyResult{
				Status:     ChainTokenEventApplyRejected,
				Message:    chainTokenRejectInsufficientBalance,
				TokenDelta: -tokenDelta,
			}, nil
		}
		return &ChainTokenEventApplyResult{
			Status:     ChainTokenEventApplyApplied,
			TokenDelta: -tokenDelta,
			NewBalance: newBalance,
		}, nil

	default:
		return nil, fmt.Errorf("unsupported event type: %s", ev.EventType)
	}
}

func validateChainTokenEventInput(ev ChainTokenEventInput) error {
	if ev.ChainID <= 0 {
		return fmt.Errorf("chain_id is required")
	}
	if strings.TrimSpace(ev.TxHash) == "" {
		return fmt.Errorf("tx_hash is required")
	}
	if ev.EventType != dao.ChainTokenLedgerEventDeposit && ev.EventType != dao.ChainTokenLedgerEventWithdraw {
		return fmt.Errorf("invalid event_type: %s", ev.EventType)
	}
	if ev.PlayerID < 0 {
		return fmt.Errorf("player_id must be non-negative")
	}
	return nil
}

func findChainTokenLedger(tx *gorm.DB, chainID int64, txHash string, logIndex uint32) (*dao.ChainTokenLedger, error) {
	var row dao.ChainTokenLedger
	err := tx.Where("chain_id = ? AND tx_hash = ? AND log_index = ?", chainID, txHash, logIndex).
		First(&row).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &row, nil
}

func duplicateChainTokenResult(existing *dao.ChainTokenLedger) *ChainTokenEventApplyResult {
	delta := existing.TokenDelta
	if existing.EventType == dao.ChainTokenLedgerEventWithdraw && existing.Status == dao.ChainTokenLedgerStatusApplied {
		delta = -existing.TokenDelta
	}
	msg := string(existing.Status)
	if existing.RejectReason != "" {
		msg = existing.RejectReason
	}
	status := ChainTokenEventApplyDuplicate
	if existing.Status == dao.ChainTokenLedgerStatusRejected {
		status = ChainTokenEventApplyRejected
	}
	return &ChainTokenEventApplyResult{
		Status:     status,
		Message:    msg,
		TokenDelta: delta,
	}
}

func creditUserTokenTx(tx *gorm.DB, playerID int64, delta int32) (int32, error) {
	token, err := EnsureUserTokenByPlayerIDTx(tx, playerID)
	if err != nil {
		return 0, err
	}
	res := tx.Model(&dao.UserToken{}).
		Where("id = ?", token.ID).
		Update("token_amount", gorm.Expr("token_amount + ?", delta))
	if res.Error != nil {
		return 0, res.Error
	}
	if res.RowsAffected == 0 {
		return 0, errors.New("user token not found")
	}
	updated, err := EnsureUserTokenByPlayerIDTx(tx, playerID)
	if err != nil {
		return 0, err
	}
	return updated.TokenAmount, nil
}

func deductUserTokenTx(tx *gorm.DB, playerID int64, delta int32) (newBalance int32, ok bool, err error) {
	token, err := EnsureUserTokenByPlayerIDTx(tx, playerID)
	if err != nil {
		return 0, false, err
	}
	res := tx.Model(&dao.UserToken{}).
		Where("id = ? AND token_amount >= ?", token.ID, delta).
		Update("token_amount", gorm.Expr("token_amount - ?", delta))
	if res.Error != nil {
		return 0, false, res.Error
	}
	if res.RowsAffected == 0 {
		return 0, false, nil
	}
	updated, err := EnsureUserTokenByPlayerIDTx(tx, playerID)
	if err != nil {
		return 0, false, err
	}
	return updated.TokenAmount, true, nil
}

func isChainTokenLedgerDuplicateErr(err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, gorm.ErrDuplicatedKey) {
		return true
	}
	errStr := strings.ToLower(err.Error())
	return strings.Contains(errStr, "duplicate") ||
		strings.Contains(errStr, "1062") ||
		strings.Contains(errStr, "unique constraint")
}
