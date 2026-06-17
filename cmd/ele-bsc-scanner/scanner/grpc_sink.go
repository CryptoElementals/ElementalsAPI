package scanner

import (
	"context"
	"fmt"
	"time"

	"github.com/CryptoElementals/common/config"
	"github.com/CryptoElementals/common/log"
	"github.com/CryptoElementals/common/rpc/proto"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

const defaultLedgerSubmitTimeout = 3 * time.Second

func defaultGrpcDialOptions() []grpc.DialOption {
	return []grpc.DialOption{
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithDefaultCallOptions(
			grpc.MaxCallRecvMsgSize(4*1024*1024),
			grpc.MaxCallSendMsgSize(4*1024*1024),
		),
	}
}

// GrpcSink submits chain events to ledger-server.
type GrpcSink struct {
	client  proto.LedgerServiceClient
	conn    *grpc.ClientConn
	timeout time.Duration
}

func NewGrpcSink(conn *grpc.ClientConn, timeout time.Duration) *GrpcSink {
	if timeout <= 0 {
		timeout = defaultLedgerSubmitTimeout
	}
	return &GrpcSink{
		client:  proto.NewLedgerServiceClient(conn),
		conn:    conn,
		timeout: timeout,
	}
}

func DialLedgerSink(parent context.Context, addr string, timeout time.Duration) (*GrpcSink, error) {
	if addr == "" {
		return nil, fmt.Errorf("ledger-server address is empty")
	}
	dialCtx, cancel := context.WithTimeout(parent, 30*time.Second)
	defer cancel()
	conn, err := grpc.DialContext(dialCtx, addr, defaultGrpcDialOptions()...)
	if err != nil {
		return nil, fmt.Errorf("dial ledger-server %s: %w", addr, err)
	}
	return NewGrpcSink(conn, timeout), nil
}

func (s *GrpcSink) EmitBlock(ctx context.Context, block *BlockData, events []TokenCollectorEvent) error {
	if s == nil || s.client == nil {
		return fmt.Errorf("grpc sink is not initialized")
	}
	if len(events) == 0 {
		return nil
	}
	req := &proto.SubmitChainEventsRequest{
		BlockNumber: block.BlockNumber,
		Events:      tokenCollectorEventsToProto(events),
	}
	submitCtx, cancel := context.WithTimeout(ctx, s.timeout)
	defer cancel()
	resp, err := s.client.SubmitChainEvents(submitCtx, req)
	if err != nil {
		log.Errorf("SubmitChainEvents failed: block=%d events=%d err=%v", block.BlockNumber, len(events), err)
		return fmt.Errorf("SubmitChainEvents block %d: %w", block.BlockNumber, err)
	}
	log.Infof("SubmitChainEvents success: block=%d events=%d results=%d", block.BlockNumber, len(events), len(resp.GetResults()))
	for _, result := range resp.GetResults() {
		if result.GetStatus() == proto.EventApplyStatus_EVENT_APPLY_STATUS_REJECTED {
			log.Warnf("ledger rejected chain event tx=%s log=%d: %s",
				result.GetTxHash(), result.GetLogIndex(), result.GetMessage())
		} else {
			log.Infof("ledger apply result: block=%d tx=%s log=%d status=%s delta=%d balance=%d msg=%s",
				block.BlockNumber, result.GetTxHash(), result.GetLogIndex(), result.GetStatus().String(),
				result.GetTokenDelta(), result.GetNewBalance(), result.GetMessage())
		}
	}
	return nil
}

func (s *GrpcSink) Close() error {
	if s == nil || s.conn == nil {
		return nil
	}
	return s.conn.Close()
}

func tokenCollectorEventsToProto(events []TokenCollectorEvent) []*proto.ChainTokenEvent {
	out := make([]*proto.ChainTokenEvent, 0, len(events))
	for _, ev := range events {
		out = append(out, tokenCollectorEventToProto(ev))
	}
	return out
}

func tokenCollectorEventToProto(ev TokenCollectorEvent) *proto.ChainTokenEvent {
	msg := &proto.ChainTokenEvent{
		ChainId:          ev.ChainID,
		BlockNumber:      ev.BlockNumber,
		BlockHash:        ev.BlockHash,
		Timestamp:        ev.Timestamp,
		TxHash:           ev.TxHash,
		LogIndex:         ev.LogIndex,
		CollectorAddress: ev.CollectorAddress,
		EventType:        ev.EventType,
	}
	if ev.Deposit != nil {
		msg.Payload = &proto.ChainTokenEvent_Deposit{
			Deposit: &proto.ChainDepositEvent{
				PlayerId:       ev.Deposit.PlayerID,
				FromAddress:    ev.Deposit.FromAddress,
				AmountWei:      ev.Deposit.Amount,
				NewCreditedWei: ev.Deposit.NewCredited,
			},
		}
	}
	if ev.Withdraw != nil {
		msg.Payload = &proto.ChainTokenEvent_Withdraw{
			Withdraw: &proto.ChainWithdrawEvent{
				PlayerId:  ev.Withdraw.PlayerID,
				Operator:  ev.Withdraw.Operator,
				ToAddress: ev.Withdraw.ToAddress,
				AmountWei: ev.Withdraw.Amount,
			},
		}
	}
	return msg
}

// NewEventSink selects log or grpc sink from bsc-scanner config.
func NewEventSink(parent context.Context) (EventSink, error) {
	cfg := config.BscScannerGConf
	if cfg.LedgerServerMocked || cfg.LedgerServer == "" {
		if cfg.LedgerServer == "" {
			log.Info("ledger-server not configured, using log sink")
		}
		return NewLogSink(), nil
	}
	timeout := cfg.LedgerServerTimeout
	if timeout <= 0 {
		timeout = defaultLedgerSubmitTimeout
	}
	return DialLedgerSink(parent, cfg.LedgerServer, timeout)
}
