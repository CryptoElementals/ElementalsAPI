package ledgerserver

import (
	"context"
	"encoding/hex"
	"errors"
	"fmt"
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
	publisher pubsub.Publisher
	chain     ChainWithdrawSubmitter
	chainID   int64
}

func NewService(publisher pubsub.Publisher, chain ChainWithdrawSubmitter, chainID int64) *Service {
	return &Service{
		publisher: publisher,
		chain:     chain,
		chainID:   chainID,
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

	if req.GetTokenAmount() <= 0 {
		return nil, fmt.Errorf("token_amount is required")
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

	pending, err := db.CreatePendingWithdraw(ctx, db.PendingWithdrawInput{
		ChainID:   s.chainID,
		PlayerID:  req.GetPlayerId(),
		AmountWei: amountWei,
		Signature: sigHex,
	})
	if err != nil {
		if errors.Is(err, db.ErrInsufficientAvailableBalance) {
			return nil, fmt.Errorf("%w", err)
		}
		return nil, err
	}

	submitCtx, cancel := context.WithTimeout(ctx, 60*time.Second)
	defer cancel()
	result, err := s.chain.Withdraw(submitCtx, req.GetPlayerId(), amountWei, signature)
	if err != nil {
		_ = db.MarkPendingWithdrawFailed(ctx, pending.RequestID, "chain_submit_failed")
		return nil, fmt.Errorf("chain withdraw: %w", err)
	}
	if err := db.UpdatePendingWithdrawTxHash(ctx, pending.RequestID, result.TxHash, result.CollectorAddress); err != nil {
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

func (s *Service) ListChainTokenLedgers(ctx context.Context, req *proto.ListChainTokenLedgersRequest) (*proto.ListChainTokenLedgersResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("nil request")
	}
	if req.GetPlayerId() <= 0 {
		return nil, fmt.Errorf("player_id is required")
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
