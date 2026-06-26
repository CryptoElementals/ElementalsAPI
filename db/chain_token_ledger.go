package db

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/CryptoElementals/common/internal/tokenunits"
	"github.com/CryptoElementals/common/log"
	dao "github.com/CryptoElementals/common/models"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type ChainTokenEventApplyStatus string

const (
	ChainTokenEventApplyFinalized ChainTokenEventApplyStatus = "finalized"
	ChainTokenEventApplyDuplicate ChainTokenEventApplyStatus = "duplicate"
	ChainTokenEventApplyFailed    ChainTokenEventApplyStatus = "failed"
)

const (
	chainTokenFailInsufficientBalance = "insufficient_balance"
	chainTokenFailChainSubmit         = "chain_submit_failed"
)

var ErrInsufficientAvailableBalance = errors.New("insufficient available balance")

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

type PendingWithdrawInput struct {
	ChainID   int64
	PlayerID  int64
	AmountWei string
	Signature string
}

type PendingWithdrawResult struct {
	RequestID  string
	LedgerID   uint
	TxHash     string
	TokenDelta int32
}

type ChainTokenLedgerFilter struct {
	PlayerID  int64
	EventType string
	Status    string
	Limit     int
	Offset    int
}

type ChainTokenLedgerListResult struct {
	Records []*dao.ChainTokenLedger
	Total   int64
}

// ApplyChainTokenEvent records a finalized on-chain deposit or withdraw event.
func ApplyChainTokenEvent(ctx context.Context, ev ChainTokenEventInput) (*ChainTokenEventApplyResult, error) {
	if ev.EventType == dao.ChainTokenLedgerEventWithdraw {
		return FinalizeChainTokenWithdraw(ctx, ev)
	}
	var result *ChainTokenEventApplyResult
	err := Get().WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var err error
		result, err = applyChainDepositEventTx(tx, ev)
		return err
	})
	if err != nil {
		return nil, err
	}
	return result, nil
}

// FinalizeChainTokenWithdraw settles a withdraw from chain events, matching API pending rows when present.
func FinalizeChainTokenWithdraw(ctx context.Context, ev ChainTokenEventInput) (*ChainTokenEventApplyResult, error) {
	var result *ChainTokenEventApplyResult
	err := Get().WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var err error
		result, err = finalizeChainTokenWithdrawTx(tx, ev)
		return err
	})
	if err != nil {
		return nil, err
	}
	return result, nil
}

func applyChainDepositEventTx(tx *gorm.DB, ev ChainTokenEventInput) (*ChainTokenEventApplyResult, error) {
	if err := validateChainTokenEventInput(ev); err != nil {
		return nil, err
	}
	if ev.EventType != dao.ChainTokenLedgerEventDeposit {
		return nil, fmt.Errorf("applyChainDepositEventTx: expected deposit, got %s", ev.EventType)
	}

	normalizedTxHash := strings.ToLower(strings.TrimSpace(ev.TxHash))
	existing, err := findChainTokenLedger(tx, ev.ChainID, normalizedTxHash, ev.LogIndex)
	if err != nil {
		return nil, err
	}
	if existing != nil {
		return duplicateChainTokenResult(existing), nil
	}

	tokenDelta, err := tokenunits.WeiToToken(ev.AmountWei)
	if err != nil {
		return nil, fmt.Errorf("convert amount wei: %w", err)
	}
	if remainder, remErr := tokenunits.WeiToTokenRemainder(ev.AmountWei); remErr == nil && remainder.Sign() > 0 {
		log.Warnf("chain token event tx=%s log=%d has wei remainder %s after /10^15",
			normalizedTxHash, ev.LogIndex, remainder.String())
	}

	row := &dao.ChainTokenLedger{
		ChainID:          ev.ChainID,
		TxHash:           normalizedTxHash,
		LogIndex:         ev.LogIndex,
		BlockNumber:      ev.BlockNumber,
		BlockHash:        strings.ToLower(strings.TrimSpace(ev.BlockHash)),
		EventType:        dao.ChainTokenLedgerEventDeposit,
		PlayerID:         ev.PlayerID,
		CollectorAddress: strings.ToLower(strings.TrimSpace(ev.CollectorAddress)),
		AmountWei:        strings.TrimSpace(ev.AmountWei),
		TokenDelta:       tokenDelta,
		Status:           dao.ChainTokenLedgerStatusFinalized,
		FromAddress:      strings.ToLower(strings.TrimSpace(ev.FromAddress)),
		ToAddress:        strings.ToLower(strings.TrimSpace(ev.ToAddress)),
		Operator:         strings.ToLower(strings.TrimSpace(ev.Operator)),
		NewCreditedWei:   strings.TrimSpace(ev.NewCreditedWei),
	}

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
		Status:     ChainTokenEventApplyFinalized,
		TokenDelta: tokenDelta,
		NewBalance: newBalance,
	}, nil
}

func finalizeChainTokenWithdrawTx(tx *gorm.DB, ev ChainTokenEventInput) (*ChainTokenEventApplyResult, error) {
	if err := validateChainTokenEventInput(ev); err != nil {
		return nil, err
	}
	if ev.EventType != dao.ChainTokenLedgerEventWithdraw {
		return nil, fmt.Errorf("finalizeChainTokenWithdrawTx: expected withdraw, got %s", ev.EventType)
	}

	normalizedTxHash := strings.ToLower(strings.TrimSpace(ev.TxHash))
	existing, err := findChainTokenLedger(tx, ev.ChainID, normalizedTxHash, ev.LogIndex)
	if err != nil {
		return nil, err
	}
	if existing != nil {
		return duplicateChainTokenResult(existing), nil
	}

	tokenDelta, err := tokenunits.WeiToToken(ev.AmountWei)
	if err != nil {
		return nil, fmt.Errorf("convert amount wei: %w", err)
	}

	pending, err := findPendingWithdrawByTx(tx, ev.ChainID, normalizedTxHash, ev.PlayerID)
	if err != nil {
		return nil, err
	}

	if pending != nil {
		return finalizePendingWithdrawTx(tx, pending, ev, tokenDelta)
	}

	row := &dao.ChainTokenLedger{
		ChainID:          ev.ChainID,
		TxHash:           normalizedTxHash,
		LogIndex:         ev.LogIndex,
		BlockNumber:      ev.BlockNumber,
		BlockHash:        strings.ToLower(strings.TrimSpace(ev.BlockHash)),
		EventType:        dao.ChainTokenLedgerEventWithdraw,
		PlayerID:         ev.PlayerID,
		CollectorAddress: strings.ToLower(strings.TrimSpace(ev.CollectorAddress)),
		AmountWei:        strings.TrimSpace(ev.AmountWei),
		TokenDelta:       tokenDelta,
		Status:           dao.ChainTokenLedgerStatusFinalized,
		Operator:         strings.ToLower(strings.TrimSpace(ev.Operator)),
		ToAddress:        strings.ToLower(strings.TrimSpace(ev.ToAddress)),
	}
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
			"status":      dao.ChainTokenLedgerStatusFailed,
			"fail_reason": chainTokenFailInsufficientBalance,
		}).Error; err != nil {
			return nil, err
		}
		return &ChainTokenEventApplyResult{
			Status:     ChainTokenEventApplyFailed,
			Message:    chainTokenFailInsufficientBalance,
			TokenDelta: -tokenDelta,
		}, nil
	}
	return &ChainTokenEventApplyResult{
		Status:     ChainTokenEventApplyFinalized,
		TokenDelta: -tokenDelta,
		NewBalance: newBalance,
	}, nil
}

func finalizePendingWithdrawTx(tx *gorm.DB, pending *dao.ChainTokenLedger, ev ChainTokenEventInput, tokenDelta int32) (*ChainTokenEventApplyResult, error) {
	updates := map[string]any{
		"log_index":         ev.LogIndex,
		"block_number":      ev.BlockNumber,
		"block_hash":        strings.ToLower(strings.TrimSpace(ev.BlockHash)),
		"collector_address": strings.ToLower(strings.TrimSpace(ev.CollectorAddress)),
		"amount_wei":        strings.TrimSpace(ev.AmountWei),
		"token_delta":       tokenDelta,
		"operator":          strings.ToLower(strings.TrimSpace(ev.Operator)),
		"to_address":        strings.ToLower(strings.TrimSpace(ev.ToAddress)),
		"status":            dao.ChainTokenLedgerStatusFinalized,
	}
	if err := tx.Model(pending).Updates(updates).Error; err != nil {
		return nil, err
	}

	newBalance, deducted, err := deductUserTokenTx(tx, ev.PlayerID, tokenDelta)
	if err != nil {
		return nil, err
	}
	if !deducted {
		if err := tx.Model(pending).Updates(map[string]any{
			"status":      dao.ChainTokenLedgerStatusFailed,
			"fail_reason": chainTokenFailInsufficientBalance,
		}).Error; err != nil {
			return nil, err
		}
		return &ChainTokenEventApplyResult{
			Status:     ChainTokenEventApplyFailed,
			Message:    chainTokenFailInsufficientBalance,
			TokenDelta: -tokenDelta,
		}, nil
	}
	return &ChainTokenEventApplyResult{
		Status:     ChainTokenEventApplyFinalized,
		TokenDelta: -tokenDelta,
		NewBalance: newBalance,
	}, nil
}

// CreatePendingWithdraw records an API-initiated withdraw before chain submission.
func CreatePendingWithdraw(ctx context.Context, input PendingWithdrawInput) (*PendingWithdrawResult, error) {
	if input.ChainID <= 0 {
		return nil, fmt.Errorf("chain_id is required")
	}
	if input.PlayerID <= 0 {
		return nil, fmt.Errorf("player_id is required")
	}
	if strings.TrimSpace(input.AmountWei) == "" {
		return nil, fmt.Errorf("amount_wei is required")
	}
	if strings.TrimSpace(input.Signature) == "" {
		return nil, fmt.Errorf("signature is required")
	}

	tokenDelta, err := tokenunits.WeiToToken(input.AmountWei)
	if err != nil {
		return nil, fmt.Errorf("convert amount wei: %w", err)
	}

	var result *PendingWithdrawResult
	err = Get().WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		available, err := availableTokenBalanceTx(tx, input.PlayerID)
		if err != nil {
			return err
		}
		if available < tokenDelta {
			return ErrInsufficientAvailableBalance
		}

		requestID := uuid.NewString()
		placeholderTx := pendingWithdrawTxHash(requestID)
		reqID := requestID
		row := &dao.ChainTokenLedger{
			RequestID:        &reqID,
			ChainID:          input.ChainID,
			TxHash:           placeholderTx,
			LogIndex:         0,
			BlockNumber:      0,
			BlockHash:        "0x0",
			EventType:        dao.ChainTokenLedgerEventWithdraw,
			PlayerID:         input.PlayerID,
			CollectorAddress: "0x0",
			AmountWei:        strings.TrimSpace(input.AmountWei),
			TokenDelta:       tokenDelta,
			Status:           dao.ChainTokenLedgerStatusPending,
			Signature:        strings.TrimSpace(input.Signature),
		}
		if err := tx.Create(row).Error; err != nil {
			return err
		}
		result = &PendingWithdrawResult{
			RequestID:  requestID,
			LedgerID:   row.ID,
			TxHash:     placeholderTx,
			TokenDelta: tokenDelta,
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return result, nil
}

// UpdatePendingWithdrawTxHash sets the on-chain tx hash after chain-server submission.
func UpdatePendingWithdrawTxHash(ctx context.Context, requestID, txHash, collectorAddress string) error {
	requestID = strings.TrimSpace(requestID)
	txHash = strings.ToLower(strings.TrimSpace(txHash))
	collectorAddress = strings.ToLower(strings.TrimSpace(collectorAddress))
	if requestID == "" || txHash == "" {
		return fmt.Errorf("request_id and tx_hash are required")
	}
	res := Get().WithContext(ctx).Model(&dao.ChainTokenLedger{}).
		Where("request_id = ? AND status = ?", requestID, dao.ChainTokenLedgerStatusPending).
		Updates(map[string]any{
			"tx_hash":           txHash,
			"collector_address": collectorAddress,
		})
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return fmt.Errorf("pending withdraw not found for request_id %s", requestID)
	}
	return nil
}

// MarkPendingWithdrawFailed marks a pending withdraw as failed.
func MarkPendingWithdrawFailed(ctx context.Context, requestID, failReason string) error {
	requestID = strings.TrimSpace(requestID)
	if requestID == "" {
		return fmt.Errorf("request_id is required")
	}
	res := Get().WithContext(ctx).Model(&dao.ChainTokenLedger{}).
		Where("request_id = ? AND status = ?", requestID, dao.ChainTokenLedgerStatusPending).
		Updates(map[string]any{
			"status":      dao.ChainTokenLedgerStatusFailed,
			"fail_reason": strings.TrimSpace(failReason),
		})
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return fmt.Errorf("pending withdraw not found for request_id %s", requestID)
	}
	return nil
}

// SumPendingWithdrawTokens returns the total game-token amount reserved by pending withdraws.
func SumPendingWithdrawTokens(ctx context.Context, playerID int64) (int32, error) {
	var total int64
	err := Get().WithContext(ctx).Model(&dao.ChainTokenLedger{}).
		Where("player_id = ? AND event_type = ? AND status = ?",
			playerID, dao.ChainTokenLedgerEventWithdraw, dao.ChainTokenLedgerStatusPending).
		Select("COALESCE(SUM(token_delta), 0)").
		Scan(&total).Error
	if err != nil {
		return 0, err
	}
	if total > int64(^uint32(0)>>1) {
		return 0, fmt.Errorf("pending withdraw sum overflows int32")
	}
	return int32(total), nil
}

// ListChainTokenLedgers returns ledger rows filtered by player and optional event_type/status.
func ListChainTokenLedgers(ctx context.Context, filter ChainTokenLedgerFilter) (*ChainTokenLedgerListResult, error) {
	if filter.PlayerID <= 0 {
		return nil, fmt.Errorf("player_id is required")
	}
	limit := filter.Limit
	if limit <= 0 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}
	offset := filter.Offset
	if offset < 0 {
		offset = 0
	}

	q := Get().WithContext(ctx).Model(&dao.ChainTokenLedger{}).
		Where("player_id = ?", filter.PlayerID)
	if strings.TrimSpace(filter.EventType) != "" {
		q = q.Where("event_type = ?", strings.TrimSpace(filter.EventType))
	}
	if strings.TrimSpace(filter.Status) != "" {
		q = q.Where("status = ?", strings.TrimSpace(filter.Status))
	}

	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, err
	}

	var records []*dao.ChainTokenLedger
	if err := q.Order("id DESC").Limit(limit).Offset(offset).Find(&records).Error; err != nil {
		return nil, err
	}
	return &ChainTokenLedgerListResult{Records: records, Total: total}, nil
}

// WithdrawableTokenAmountBreakdown is the withdrawable token amount and how it is derived for a player.
type WithdrawableTokenAmountBreakdown struct {
	WithdrawableTokenAmount    int32
	TokenAmount                int32
	LockedTokens               int32
	PendingWithdrawTokenAmount int32
}

// GetWithdrawableTokenAmount returns how many game tokens the player can withdraw now.
func GetWithdrawableTokenAmount(ctx context.Context, playerID int64) (*WithdrawableTokenAmountBreakdown, error) {
	if playerID <= 0 {
		return nil, fmt.Errorf("player_id is required")
	}
	var breakdown WithdrawableTokenAmountBreakdown
	err := Get().WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var err error
		breakdown, err = withdrawableTokenAmountTx(tx, playerID)
		return err
	})
	if err != nil {
		return nil, err
	}
	return &breakdown, nil
}

func withdrawableTokenAmountTx(tx *gorm.DB, playerID int64) (WithdrawableTokenAmountBreakdown, error) {
	token, err := EnsureUserTokenByPlayerIDTx(tx, playerID)
	if err != nil {
		return WithdrawableTokenAmountBreakdown{}, err
	}
	locked, err := sumActiveLockedTokensTx(tx, token.ID)
	if err != nil {
		return WithdrawableTokenAmountBreakdown{}, err
	}
	var pendingSum int64
	if err := tx.Model(&dao.ChainTokenLedger{}).
		Where("player_id = ? AND event_type = ? AND status = ?",
			playerID, dao.ChainTokenLedgerEventWithdraw, dao.ChainTokenLedgerStatusPending).
		Select("COALESCE(SUM(token_delta), 0)").
		Scan(&pendingSum).Error; err != nil {
		return WithdrawableTokenAmountBreakdown{}, err
	}
	if pendingSum > int64(^uint32(0)>>1) {
		return WithdrawableTokenAmountBreakdown{}, fmt.Errorf("pending withdraw sum overflows int32")
	}
	pending := int32(pendingSum)
	available := int64(token.TokenAmount) - int64(locked) - int64(pending)
	withdrawable := int32(0)
	if available > 0 {
		if available > int64(^uint32(0)>>1) {
			withdrawable = int32(^uint32(0) >> 1)
		} else {
			withdrawable = int32(available)
		}
	}
	return WithdrawableTokenAmountBreakdown{
		WithdrawableTokenAmount:    withdrawable,
		TokenAmount:                token.TokenAmount,
		LockedTokens:               locked,
		PendingWithdrawTokenAmount: pending,
	}, nil
}

func availableTokenBalanceTx(tx *gorm.DB, playerID int64) (int32, error) {
	breakdown, err := withdrawableTokenAmountTx(tx, playerID)
	if err != nil {
		return 0, err
	}
	return breakdown.WithdrawableTokenAmount, nil
}

func pendingWithdrawTxHash(requestID string) string {
	return "pending:" + requestID
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

func findPendingWithdrawByTx(tx *gorm.DB, chainID int64, txHash string, playerID int64) (*dao.ChainTokenLedger, error) {
	var row dao.ChainTokenLedger
	err := tx.Where("chain_id = ? AND tx_hash = ? AND player_id = ? AND status = ?",
		chainID, txHash, playerID, dao.ChainTokenLedgerStatusPending).
		Order("id ASC").
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
	if existing.EventType == dao.ChainTokenLedgerEventWithdraw && existing.Status == dao.ChainTokenLedgerStatusFinalized {
		delta = -existing.TokenDelta
	}
	msg := string(existing.Status)
	if existing.FailReason != "" {
		msg = existing.FailReason
	}
	status := ChainTokenEventApplyDuplicate
	if existing.Status == dao.ChainTokenLedgerStatusFailed {
		status = ChainTokenEventApplyFailed
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
