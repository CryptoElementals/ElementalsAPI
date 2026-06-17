package ledgerserver

import (
	"context"
	"fmt"

	"github.com/CryptoElementals/common/db"
	dao "github.com/CryptoElementals/common/models"
	"github.com/CryptoElementals/common/rpc/proto"
)

// Service applies on-chain token events to the ledger and user balances.
type Service struct{}

func NewService() *Service {
	return &Service{}
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

func (s *Service) applyOne(ctx context.Context, ev *proto.ChainTokenEvent) (*proto.EventApplyResult, error) {
	if ev == nil {
		return nil, fmt.Errorf("nil event")
	}
	input, err := protoToChainTokenEventInput(ev)
	if err != nil {
		return nil, fmt.Errorf("invalid event: %w", err)
	}
	applyResult, err := db.ApplyChainTokenEvent(ctx, input)
	if err != nil {
		return nil, err
	}
	return dbApplyResultToProto(ev.GetTxHash(), ev.GetLogIndex(), applyResult), nil
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
	case db.ChainTokenEventApplyApplied:
		out.Status = proto.EventApplyStatus_EVENT_APPLY_STATUS_APPLIED
	case db.ChainTokenEventApplyDuplicate:
		out.Status = proto.EventApplyStatus_EVENT_APPLY_STATUS_DUPLICATE
	case db.ChainTokenEventApplyRejected:
		out.Status = proto.EventApplyStatus_EVENT_APPLY_STATUS_REJECTED
	default:
		out.Status = proto.EventApplyStatus_EVENT_APPLY_STATUS_UNSPECIFIED
	}
	return out
}
