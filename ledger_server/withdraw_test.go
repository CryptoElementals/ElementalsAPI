package ledgerserver

import (
	"context"
	"errors"
	"testing"

	"github.com/CryptoElementals/common/db"
	"github.com/CryptoElementals/common/ledger_server/chainclient"
	dao "github.com/CryptoElementals/common/models"
	"github.com/CryptoElementals/common/rpc/proto"
	"github.com/stretchr/testify/require"
)

type stubChainSubmitter struct {
	txHash    string
	collector string
	err       error
}

func (s *stubChainSubmitter) Withdraw(ctx context.Context, playerID int64, amountWei string, signature []byte) (*chainclient.WithdrawResult, error) {
	if s.err != nil {
		return nil, s.err
	}
	return &chainclient.WithdrawResult{
		TxHash:           s.txHash,
		CollectorAddress: s.collector,
	}, nil
}

func TestRequestWithdrawCreatesPendingAndUpdatesTxHash(t *testing.T) {
	require.NoError(t, db.Init(&db.Config{Development: true}))
	require.NoError(t, db.MigrateMemDb())

	const depositWei = "2000000000000000000"
	_, err := db.ApplyChainTokenEvent(context.Background(), db.ChainTokenEventInput{
		ChainID: 97, BlockNumber: 1, BlockHash: "0xabc", TxHash: "0xdep", LogIndex: 1,
		CollectorAddress: "0xcol", EventType: dao.ChainTokenLedgerEventDeposit,
		PlayerID: 50, AmountWei: depositWei, FromAddress: "0xfrom", NewCreditedWei: depositWei,
	})
	require.NoError(t, err)

	svc := NewService(nil, &stubChainSubmitter{
		txHash:    "0xwithdrawtx",
		collector: "0xcollector",
	}, 97)

	resp, err := svc.RequestWithdraw(context.Background(), &proto.RequestWithdrawRequest{
		PlayerId:    50,
		TokenAmount: 50,
		Signature:   []byte("abcdef"),
	})
	require.NoError(t, err)
	require.Equal(t, string(dao.ChainTokenLedgerStatusPending), resp.GetStatus())
	require.Equal(t, "0xwithdrawtx", resp.GetTxHash())
	require.NotEmpty(t, resp.GetRequestId())

	list, err := svc.ListChainTokenLedgers(context.Background(), &proto.ListChainTokenLedgersRequest{
		PlayerId: 50,
		Status:   string(dao.ChainTokenLedgerStatusPending),
	})
	require.NoError(t, err)
	require.Equal(t, int64(1), list.GetTotal())
	require.Equal(t, "0xwithdrawtx", list.GetRecords()[0].GetTxHash())
}

func TestRequestWithdrawMarksFailedOnChainError(t *testing.T) {
	require.NoError(t, db.Init(&db.Config{Development: true}))
	require.NoError(t, db.MigrateMemDb())

	const depositWei = "1000000000000000000"
	_, err := db.ApplyChainTokenEvent(context.Background(), db.ChainTokenEventInput{
		ChainID: 97, BlockNumber: 1, BlockHash: "0xabc", TxHash: "0xdep2", LogIndex: 1,
		CollectorAddress: "0xcol", EventType: dao.ChainTokenLedgerEventDeposit,
		PlayerID: 51, AmountWei: depositWei, FromAddress: "0xfrom", NewCreditedWei: depositWei,
	})
	require.NoError(t, err)

	svc := NewService(nil, &stubChainSubmitter{err: errors.New("chain down")}, 97)
	_, err = svc.RequestWithdraw(context.Background(), &proto.RequestWithdrawRequest{
		PlayerId:    51,
		TokenAmount: 10,
		Signature:   []byte("abcdef"),
	})
	require.Error(t, err)

	list, err := svc.ListChainTokenLedgers(context.Background(), &proto.ListChainTokenLedgersRequest{
		PlayerId: 51,
		Status:   string(dao.ChainTokenLedgerStatusFailed),
	})
	require.NoError(t, err)
	require.Equal(t, int64(1), list.GetTotal())
	require.Equal(t, "chain_submit_failed", list.GetRecords()[0].GetFailReason())
}
