package ledgerserver

import (
	"context"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/CryptoElementals/common/db"
	"github.com/CryptoElementals/common/internal/tokenunits"
	"github.com/CryptoElementals/common/ledger_server/chainclient"
	dao "github.com/CryptoElementals/common/models"
	"github.com/CryptoElementals/common/pubsub"
	"github.com/CryptoElementals/common/rpc/proto"
)

// ChainWithdrawSubmitter submits withdraw transactions to chain-server.
type ChainWithdrawSubmitter interface {
	Withdraw(ctx context.Context, playerID int64, amountWei string, signature []byte) (*chainclient.WithdrawResult, error)
}

// Service applies on-chain token events to the ledger and user balances.
type Service struct {
	publisher                    pubsub.Publisher
	chain                        ChainWithdrawSubmitter
	chainID                      int64
	withdrawAuditThresholdTokens int32
}

func NewService(publisher pubsub.Publisher, chain ChainWithdrawSubmitter, chainID int64, withdrawAuditThresholdTokens int32) *Service {
	return &Service{
		publisher:                    publisher,
		chain:                        chain,
		chainID:                      chainID,
		withdrawAuditThresholdTokens: tokenunits.ResolveWithdrawAuditThreshold(withdrawAuditThresholdTokens),
	}
}

func (s *Service) SubmitChainEvents(ctx context.Context, req *proto.SubmitChainEventsRequest) (*proto.SubmitChainEventsResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("nil request")
	}
	resp := &proto.SubmitChainEventsResponse{
		Results: make([]*proto.EventApplyResult, 0, len(req.GetEvents())),
	}
	for _, ev := range req.GetEvents() {
		result, err := s.applyOne(ctx, ev)
		if err != nil {
			return nil, err
		}
		resp.Results = append(resp.Results, result)
	}
	return resp, nil
}

func (s *Service) RequestWithdraw(ctx context.Context, req *proto.RequestWithdrawRequest) (*proto.RequestWithdrawResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("nil request")
	}
	if req.GetPlayerId() <= 0 {
		return nil, fmt.Errorf("player_id is required")
	}
	if s.chain == nil {
		return nil, fmt.Errorf("chain client is not configured")
	}
	if s.chainID <= 0 {
		return nil, fmt.Errorf("chain_id is not configured")
	}

	if err := tokenunits.ValidateWithdrawTokenAmount(req.GetTokenAmount()); err != nil {
		return nil, err
	}
	amountWei, err := tokenunits.TokenToWei(req.GetTokenAmount())
	if err != nil {
		return nil, err
	}
	if len(req.GetSignature()) == 0 {
		return nil, fmt.Errorf("signature is required")
	}
	signature := req.GetSignature()
	sigHex := "0x" + hex.EncodeToString(signature)

	if tokenunits.RequiresWithdrawAudit(req.GetTokenAmount(), s.withdrawAuditThresholdTokens) {
		auditing, err := db.CreateAuditingWithdraw(ctx, db.PendingWithdrawInput{
			ChainID:   s.chainID,
			PlayerID:  req.GetPlayerId(),
			AmountWei: amountWei,
			Signature: sigHex,
		})
		if err != nil {
			if errors.Is(err, db.ErrInsufficientAvailableBalance) || errors.Is(err, db.ErrAuditingWithdrawInProgress) {
				return nil, fmt.Errorf("%w", err)
			}
			return nil, err
		}
		return &proto.RequestWithdrawResponse{
			RequestId: auditing.RequestID,
			LedgerId:  uint64(auditing.LedgerID),
			Status:    string(dao.ChainTokenLedgerStatusAuditing),
		}, nil
	}

	pending, err := db.CreatePendingWithdraw(ctx, db.PendingWithdrawInput{
		ChainID:   s.chainID,
		PlayerID:  req.GetPlayerId(),
		AmountWei: amountWei,
		Signature: sigHex,
	})
	if err != nil {
		if errors.Is(err, db.ErrInsufficientAvailableBalance) || errors.Is(err, db.ErrAuditingWithdrawInProgress) {
			return nil, fmt.Errorf("%w", err)
		}
		return nil, err
	}

	result, err := s.submitWithdrawToChain(ctx, pending.RequestID, req.GetPlayerId(), amountWei, signature)
	if err != nil {
		return nil, err
	}

	return &proto.RequestWithdrawResponse{
		RequestId:        pending.RequestID,
		TxHash:           result.TxHash,
		CollectorAddress: result.CollectorAddress,
		LedgerId:         uint64(pending.LedgerID),
		Status:           string(dao.ChainTokenLedgerStatusPending),
	}, nil
}

func (s *Service) submitWithdrawToChain(ctx context.Context, requestID string, playerID int64, amountWei string, signature []byte) (*chainclient.WithdrawResult, error) {
	submitCtx, cancel := context.WithTimeout(ctx, 60*time.Second)
	defer cancel()
	result, err := s.chain.Withdraw(submitCtx, playerID, amountWei, signature)
	if err != nil {
		_ = db.MarkPendingWithdrawFailed(ctx, requestID, chainTokenFailChainSubmit)
		return nil, fmt.Errorf("chain withdraw: %w", err)
	}
	if err := db.UpdatePendingWithdrawTxHash(ctx, requestID, result.TxHash, result.CollectorAddress); err != nil {
		return nil, err
	}
	return result, nil
}

const chainTokenFailChainSubmit = "chain_submit_failed"

func (s *Service) AuditWithdraw(ctx context.Context, req *proto.AuditWithdrawRequest) (*proto.AuditWithdrawResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("nil request")
	}
	requestID := strings.TrimSpace(req.GetRequestId())
	if requestID == "" {
		return nil, fmt.Errorf("request_id is required")
	}
	if s.chain == nil {
		return nil, fmt.Errorf("chain client is not configured")
	}

	switch req.GetDecision() {
	case proto.WithdrawAuditDecision_WITHDRAW_AUDIT_DECISION_REJECT:
		if strings.TrimSpace(req.GetFailReason()) == "" {
			return nil, fmt.Errorf("fail_reason is required")
		}
		if err := db.RejectAuditingWithdraw(ctx, requestID, req.GetFailReason()); err != nil {
			return nil, err
		}
		return &proto.AuditWithdrawResponse{
			RequestId: requestID,
			Status:    string(dao.ChainTokenLedgerStatusFailed),
		}, nil
	case proto.WithdrawAuditDecision_WITHDRAW_AUDIT_DECISION_APPROVE:
		row, err := db.ApproveAuditingWithdraw(ctx, requestID)
		if err != nil {
			return nil, err
		}
		sigBytes, err := decodeHexSignature(row.Signature)
		if err != nil {
			return nil, fmt.Errorf("invalid stored signature: %w", err)
		}
		result, err := s.submitWithdrawToChain(ctx, row.RequestID, row.PlayerID, row.AmountWei, sigBytes)
		if err != nil {
			return nil, err
		}
		return &proto.AuditWithdrawResponse{
			RequestId:        row.RequestID,
			TxHash:           result.TxHash,
			CollectorAddress: result.CollectorAddress,
			LedgerId:         uint64(row.LedgerID),
			Status:           string(dao.ChainTokenLedgerStatusPending),
		}, nil
	default:
		return nil, fmt.Errorf("decision is required")
	}
}

func decodeHexSignature(sig string) ([]byte, error) {
	raw := strings.TrimSpace(sig)
	raw = strings.TrimPrefix(raw, "0x")
	if raw == "" {
		return nil, fmt.Errorf("signature is empty")
	}
	b, err := hex.DecodeString(raw)
	if err != nil {
		return nil, fmt.Errorf("invalid signature hex")
	}
	return b, nil
}

func (s *Service) ListChainTokenLedgers(ctx context.Context, req *proto.ListChainTokenLedgersRequest) (*proto.ListChainTokenLedgersResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("nil request")
	}
	list, err := db.ListChainTokenLedgers(ctx, db.ChainTokenLedgerFilter{
		PlayerID:  req.GetPlayerId(),
		EventType: req.GetEventType(),
		Status:    req.GetStatus(),
		Limit:     int(req.GetLimit()),
		Offset:    int(req.GetOffset()),
	})
	if err != nil {
		return nil, err
	}
	out := &proto.ListChainTokenLedgersResponse{Total: list.Total}
	for _, row := range list.Records {
		out.Records = append(out.Records, chainTokenLedgerToProto(row))
	}
	return out, nil
}

func (s *Service) GetWithdrawableTokenAmount(ctx context.Context, req *proto.GetWithdrawableTokenAmountRequest) (*proto.GetWithdrawableTokenAmountResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("nil request")
	}
	if req.GetPlayerId() <= 0 {
		return nil, fmt.Errorf("player_id is required")
	}
	breakdown, err := db.GetWithdrawableTokenAmount(ctx, req.GetPlayerId())
	if err != nil {
		return nil, err
	}
	return &proto.GetWithdrawableTokenAmountResponse{
		WithdrawableTokenAmount:    breakdown.WithdrawableTokenAmount,
		TokenAmount:                breakdown.TokenAmount,
		LockedTokens:               breakdown.LockedTokens,
		PendingWithdrawTokenAmount: breakdown.PendingWithdrawTokenAmount,
	}, nil
}

func (s *Service) GetTokenUnitRates() *proto.GetTokenUnitRatesResponse {
	return tokenUnitRatesToProto(tokenunits.DefaultSpec.Rates())
}

func (s *Service) ConvertTokenAmount(req *proto.ConvertTokenAmountRequest) (*proto.ConvertTokenAmountResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("nil request")
	}
	from := tokenunits.ProtoUnit(int32(req.GetFromUnit()))
	to := tokenunits.ProtoUnit(int32(req.GetToUnit()))
	if from == tokenunits.UnitUnspecified || to == tokenunits.UnitUnspecified {
		return nil, fmt.Errorf("from_unit and to_unit are required")
	}
	amount, remainder, err := tokenunits.Convert(from, to, req.GetAmount())
	if err != nil {
		return nil, err
	}
	return &proto.ConvertTokenAmountResponse{
		Amount:    amount,
		Remainder: remainder,
	}, nil
}

func (s *Service) applyOne(ctx context.Context, ev *proto.ChainTokenEvent) (*proto.EventApplyResult, error) {
	if ev == nil {
		return nil, fmt.Errorf("nil event")
	}
	input, err := protoToChainTokenEventInput(ev)
	if err != nil {
		return nil, fmt.Errorf("invalid event: %w", err)
	}
	var applyResult *db.ChainTokenEventApplyResult
	if input.EventType == dao.ChainTokenLedgerEventWithdraw {
		applyResult, err = db.FinalizeChainTokenWithdraw(ctx, input)
	} else {
		applyResult, err = db.ApplyChainTokenEvent(ctx, input)
	}
	if err != nil {
		return nil, err
	}
	result := dbApplyResultToProto(ev.GetTxHash(), ev.GetLogIndex(), applyResult)
	s.publishTokenUpdated(ctx, ev, applyResult)
	return result, nil
}

func chainTokenLedgerToProto(row *dao.ChainTokenLedger) *proto.ChainTokenLedgerRecord {
	if row == nil {
		return nil
	}
	rec := &proto.ChainTokenLedgerRecord{
		Id:               uint64(row.ID),
		ChainId:          row.ChainID,
		TxHash:           row.TxHash,
		LogIndex:         row.LogIndex,
		BlockNumber:      row.BlockNumber,
		BlockHash:        row.BlockHash,
		EventType:        string(row.EventType),
		PlayerId:         row.PlayerID,
		CollectorAddress: row.CollectorAddress,
		AmountWei:        row.AmountWei,
		TokenDelta:       row.TokenDelta,
		Status:           string(row.Status),
		FailReason:       row.FailReason,
		Signature:        row.Signature,
		FromAddress:      row.FromAddress,
		ToAddress:        row.ToAddress,
		Operator:         row.Operator,
		NewCreditedWei:   row.NewCreditedWei,
	}
	if row.RequestID != nil {
		rec.RequestId = *row.RequestID
	}
	if !row.CreatedAt.IsZero() {
		rec.CreatedAt = row.CreatedAt.Unix()
	}
	return rec
}

func protoToChainTokenEventInput(ev *proto.ChainTokenEvent) (db.ChainTokenEventInput, error) {
	eventType := dao.ChainTokenLedgerEventType(ev.GetEventType())
	input := db.ChainTokenEventInput{
		ChainID:          int64(ev.GetChainId()),
		BlockNumber:      ev.GetBlockNumber(),
		BlockHash:        ev.GetBlockHash(),
		Timestamp:        ev.GetTimestamp(),
		TxHash:           ev.GetTxHash(),
		LogIndex:         ev.GetLogIndex(),
		CollectorAddress: ev.GetCollectorAddress(),
		EventType:        eventType,
	}
	switch eventType {
	case dao.ChainTokenLedgerEventDeposit:
		dep := ev.GetDeposit()
		if dep == nil {
			return input, fmt.Errorf("deposit payload required for event_type deposit")
		}
		input.PlayerID = dep.GetPlayerId()
		input.AmountWei = dep.GetAmountWei()
		input.FromAddress = dep.GetFromAddress()
		input.NewCreditedWei = dep.GetNewCreditedWei()
	case dao.ChainTokenLedgerEventWithdraw:
		wd := ev.GetWithdraw()
		if wd == nil {
			return input, fmt.Errorf("withdraw payload required for event_type withdraw")
		}
		input.PlayerID = wd.GetPlayerId()
		input.AmountWei = wd.GetAmountWei()
		input.Operator = wd.GetOperator()
		input.ToAddress = wd.GetToAddress()
	default:
		return input, fmt.Errorf("unsupported event_type: %s", ev.GetEventType())
	}
	return input, nil
}

func dbApplyResultToProto(txHash string, logIndex uint32, r *db.ChainTokenEventApplyResult) *proto.EventApplyResult {
	if r == nil {
		return &proto.EventApplyResult{
			TxHash:   txHash,
			LogIndex: logIndex,
		}
	}
	out := &proto.EventApplyResult{
		TxHash:     txHash,
		LogIndex:   logIndex,
		Message:    r.Message,
		TokenDelta: r.TokenDelta,
		NewBalance: r.NewBalance,
	}
	switch r.Status {
	case db.ChainTokenEventApplyFinalized:
		out.Status = proto.EventApplyStatus_EVENT_APPLY_STATUS_FINALIZED
	case db.ChainTokenEventApplyDuplicate:
		out.Status = proto.EventApplyStatus_EVENT_APPLY_STATUS_DUPLICATE
	case db.ChainTokenEventApplyFailed:
		out.Status = proto.EventApplyStatus_EVENT_APPLY_STATUS_FAILED
	default:
		out.Status = proto.EventApplyStatus_EVENT_APPLY_STATUS_UNSPECIFIED
	}
	return out
}
