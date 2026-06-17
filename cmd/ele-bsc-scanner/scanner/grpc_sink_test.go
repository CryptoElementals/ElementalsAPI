package scanner

import (
	"context"
	"errors"
	"testing"

	"github.com/CryptoElementals/common/rpc/proto"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
)

type mockLedgerClient struct {
	submit func(ctx context.Context, in *proto.SubmitChainEventsRequest, opts ...grpc.CallOption) (*proto.SubmitChainEventsResponse, error)
}

func (m *mockLedgerClient) SubmitChainEvents(ctx context.Context, in *proto.SubmitChainEventsRequest, opts ...grpc.CallOption) (*proto.SubmitChainEventsResponse, error) {
	return m.submit(ctx, in, opts...)
}

func TestGrpcSinkEmitBlockRejectedDoesNotError(t *testing.T) {
	sink := &GrpcSink{
		client: &mockLedgerClient{
			submit: func(ctx context.Context, in *proto.SubmitChainEventsRequest, opts ...grpc.CallOption) (*proto.SubmitChainEventsResponse, error) {
				return &proto.SubmitChainEventsResponse{
					Results: []*proto.EventApplyResult{
						{
							TxHash:   "0xtx",
							LogIndex: 1,
							Status:   proto.EventApplyStatus_EVENT_APPLY_STATUS_REJECTED,
							Message:  "insufficient_balance",
						},
					},
				}, nil
			},
		},
		timeout: defaultLedgerSubmitTimeout,
	}

	block := &BlockData{BlockNumber: 100}
	events := []TokenCollectorEvent{
		{
			ChainID:   97,
			TxHash:    "0xtx",
			LogIndex:  1,
			EventType: eventTypeWithdraw,
			Withdraw: &WithdrawPayload{
				PlayerID: 1,
				Amount:   "1000000000000000000",
			},
		},
	}
	require.NoError(t, sink.EmitBlock(context.Background(), block, events))
}

func TestGrpcSinkEmitBlockTransportError(t *testing.T) {
	sink := &GrpcSink{
		client: &mockLedgerClient{
			submit: func(ctx context.Context, in *proto.SubmitChainEventsRequest, opts ...grpc.CallOption) (*proto.SubmitChainEventsResponse, error) {
				return nil, errors.New("connection refused")
			},
		},
		timeout: defaultLedgerSubmitTimeout,
	}

	block := &BlockData{BlockNumber: 100}
	events := []TokenCollectorEvent{{ChainID: 97, TxHash: "0xtx", LogIndex: 1, EventType: eventTypeDeposit, Deposit: &DepositPayload{Amount: "1"}}}
	require.Error(t, sink.EmitBlock(context.Background(), block, events))
}

func TestTokenCollectorEventToProto(t *testing.T) {
	ev := TokenCollectorEvent{
		ChainID:          97,
		BlockNumber:      100,
		BlockHash:        "0xabc",
		Timestamp:        1700000000,
		TxHash:           "0xtx",
		LogIndex:         2,
		CollectorAddress: "0xcollector",
		EventType:        eventTypeDeposit,
		Deposit: &DepositPayload{
			PlayerID:    42,
			FromAddress: "0xfrom",
			Amount:      "1000",
			NewCredited: "5000",
		},
	}
	msg := tokenCollectorEventToProto(ev)
	require.Equal(t, uint64(97), msg.GetChainId())
	require.Equal(t, int64(42), msg.GetDeposit().GetPlayerId())
	require.Equal(t, "1000", msg.GetDeposit().GetAmountWei())
}
