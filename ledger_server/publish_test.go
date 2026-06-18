package ledgerserver

import (
	"context"
	"testing"

	"github.com/CryptoElementals/common/db"
	dao "github.com/CryptoElementals/common/models"
	"github.com/CryptoElementals/common/rpc/proto"
	"github.com/stretchr/testify/require"
)

func TestChainEventPlayerID(t *testing.T) {
	dep := &proto.ChainTokenEvent{
		EventType: string(dao.ChainTokenLedgerEventDeposit),
		Payload: &proto.ChainTokenEvent_Deposit{
			Deposit: &proto.ChainDepositEvent{
				PlayerId: 42,
			},
		},
	}
	require.Equal(t, int64(42), chainEventPlayerID(dep))

	wd := &proto.ChainTokenEvent{
		EventType: string(dao.ChainTokenLedgerEventWithdraw),
		Payload: &proto.ChainTokenEvent_Withdraw{
			Withdraw: &proto.ChainWithdrawEvent{
				PlayerId: 7,
			},
		},
	}
	require.Equal(t, int64(7), chainEventPlayerID(wd))
}

func TestSumLockedTokens(t *testing.T) {
	total := sumLockedTokens(&dao.UserToken{
		LockedTokens: []*dao.LockedUserToken{
			{TokenAmount: 100},
			{TokenAmount: 200},
		},
	})
	require.Equal(t, int32(300), total)
}

func TestPublishTokenUpdatedSkipsNonFinalized(t *testing.T) {
	svc := NewService(nil, nil, 0)
	ev := &proto.ChainTokenEvent{
		EventType: string(dao.ChainTokenLedgerEventDeposit),
		Payload: &proto.ChainTokenEvent_Deposit{
			Deposit: &proto.ChainDepositEvent{PlayerId: 1},
		},
	}
	svc.publishTokenUpdated(context.Background(), ev, &db.ChainTokenEventApplyResult{
		Status: db.ChainTokenEventApplyDuplicate,
	})
}
