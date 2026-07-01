package ledgerserver

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/CryptoElementals/common/db"
	"github.com/CryptoElementals/common/internal/tokenunits"
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
	}, 97, 0)

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

	svc := NewService(nil, &stubChainSubmitter{err: errors.New("chain down")}, 97, 0)
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

func TestRequestWithdrawExceedsMaxTokenAmount(t *testing.T) {
	svc := NewService(nil, &stubChainSubmitter{}, 97, 0)
	_, err := svc.RequestWithdraw(context.Background(), &proto.RequestWithdrawRequest{
		PlayerId:    1,
		TokenAmount: tokenunits.MaxWithdrawTokenAmount + 1,
		Signature:   []byte("abcdef"),
	})
	require.Error(t, err)
	require.Contains(t, err.Error(), "max withdraw limit")
}

func TestGetWithdrawableTokenAmountService(t *testing.T) {
	require.NoError(t, db.Init(&db.Config{Development: true}))
	require.NoError(t, db.MigrateMemDb())

	const depositWei = "1000000000000000000"
	_, err := db.ApplyChainTokenEvent(context.Background(), db.ChainTokenEventInput{
		ChainID: 97, BlockNumber: 1, BlockHash: "0xabc", TxHash: "0xdep-wd", LogIndex: 1,
		CollectorAddress: "0xcol", EventType: dao.ChainTokenLedgerEventDeposit,
		PlayerID: 60, AmountWei: depositWei, FromAddress: "0xfrom", NewCreditedWei: depositWei,
	})
	require.NoError(t, err)

	svc := NewService(nil, nil, 97, 0)
	resp, err := svc.GetWithdrawableTokenAmount(context.Background(), &proto.GetWithdrawableTokenAmountRequest{
		PlayerId: 60,
	})
	require.NoError(t, err)
	require.Equal(t, int32(1000), resp.GetWithdrawableTokenAmount())
	require.Equal(t, int32(1000), resp.GetTokenAmount())
	require.Equal(t, int32(0), resp.GetLockedTokens())
	require.Equal(t, int32(0), resp.GetPendingWithdrawTokenAmount())
}

func depositTokens(t *testing.T, playerID int64, tokenAmount int32) {
	t.Helper()
	amountWei, err := tokenunits.TokenToWei(tokenAmount)
	require.NoError(t, err)
	_, err = db.ApplyChainTokenEvent(context.Background(), db.ChainTokenEventInput{
		ChainID: 97, BlockNumber: 1, BlockHash: "0xabc", TxHash: fmt.Sprintf("0xdep-%d", playerID),
		LogIndex: 1, CollectorAddress: "0xcol", EventType: dao.ChainTokenLedgerEventDeposit,
		PlayerID: playerID, AmountWei: amountWei, FromAddress: "0xfrom", NewCreditedWei: amountWei,
	})
	require.NoError(t, err)
}

func TestRequestWithdrawLargeAmountCreatesAuditing(t *testing.T) {
	require.NoError(t, db.Init(&db.Config{Development: true}))
	require.NoError(t, db.MigrateMemDb())

	depositTokens(t, 70, 200_000)
	chain := &stubChainSubmitter{txHash: "0xshould-not-call", collector: "0xcol"}
	svc := NewService(nil, chain, 97, 0)

	resp, err := svc.RequestWithdraw(context.Background(), &proto.RequestWithdrawRequest{
		PlayerId:    70,
		TokenAmount: 100_001,
		Signature:   []byte("abcdef"),
	})
	require.NoError(t, err)
	require.Equal(t, string(dao.ChainTokenLedgerStatusAuditing), resp.GetStatus())
	require.Empty(t, resp.GetTxHash())
	require.NotEmpty(t, resp.GetRequestId())

	list, err := svc.ListChainTokenLedgers(context.Background(), &proto.ListChainTokenLedgersRequest{
		PlayerId: 0,
		Status:   string(dao.ChainTokenLedgerStatusAuditing),
	})
	require.NoError(t, err)
	require.Equal(t, int64(1), list.GetTotal())
}

func TestRequestWithdrawAtThresholdStillSubmitsChain(t *testing.T) {
	require.NoError(t, db.Init(&db.Config{Development: true}))
	require.NoError(t, db.MigrateMemDb())

	depositTokens(t, 71, 200_000)
	chain := &stubChainSubmitter{txHash: "0xsmallwd", collector: "0xcol"}
	svc := NewService(nil, chain, 97, 0)

	resp, err := svc.RequestWithdraw(context.Background(), &proto.RequestWithdrawRequest{
		PlayerId:    71,
		TokenAmount: 100_000,
		Signature:   []byte("abcdef"),
	})
	require.NoError(t, err)
	require.Equal(t, string(dao.ChainTokenLedgerStatusPending), resp.GetStatus())
	require.Equal(t, "0xsmallwd", resp.GetTxHash())
}

func TestRequestWithdrawBlockedWhileAuditingInProgress(t *testing.T) {
	require.NoError(t, db.Init(&db.Config{Development: true}))
	require.NoError(t, db.MigrateMemDb())

	depositTokens(t, 72, 300_000)
	svc := NewService(nil, &stubChainSubmitter{txHash: "0xwd", collector: "0xcol"}, 97, 0)

	_, err := svc.RequestWithdraw(context.Background(), &proto.RequestWithdrawRequest{
		PlayerId: 72, TokenAmount: 100_001, Signature: []byte("abcdef"),
	})
	require.NoError(t, err)

	_, err = svc.RequestWithdraw(context.Background(), &proto.RequestWithdrawRequest{
		PlayerId: 72, TokenAmount: 10, Signature: []byte("abcdef"),
	})
	require.ErrorIs(t, err, db.ErrAuditingWithdrawInProgress)
}

func TestRequestWithdrawCustomAuditThreshold(t *testing.T) {
	require.NoError(t, db.Init(&db.Config{Development: true}))
	require.NoError(t, db.MigrateMemDb())

	depositTokens(t, 73, 100_000)
	svc := NewService(nil, &stubChainSubmitter{txHash: "0xwd73", collector: "0xcol"}, 97, 50_000)

	resp, err := svc.RequestWithdraw(context.Background(), &proto.RequestWithdrawRequest{
		PlayerId: 73, TokenAmount: 50_001, Signature: []byte("abcdef"),
	})
	require.NoError(t, err)
	require.Equal(t, string(dao.ChainTokenLedgerStatusAuditing), resp.GetStatus())

	depositTokens(t, 74, 100_000)
	resp, err = svc.RequestWithdraw(context.Background(), &proto.RequestWithdrawRequest{
		PlayerId: 74, TokenAmount: 50_000, Signature: []byte("abcdef"),
	})
	require.NoError(t, err)
	require.Equal(t, string(dao.ChainTokenLedgerStatusPending), resp.GetStatus())
	require.Equal(t, "0xwd73", resp.GetTxHash())
}

func TestAuditWithdrawApproveAndReject(t *testing.T) {
	require.NoError(t, db.Init(&db.Config{Development: true}))
	require.NoError(t, db.MigrateMemDb())

	depositTokens(t, 80, 200_000)
	chain := &stubChainSubmitter{txHash: "0xapproved", collector: "0xcollector"}
	svc := NewService(nil, chain, 97, 0)

	createResp, err := svc.RequestWithdraw(context.Background(), &proto.RequestWithdrawRequest{
		PlayerId: 80, TokenAmount: 100_001, Signature: []byte("abcdef"),
	})
	require.NoError(t, err)

	approveResp, err := svc.AuditWithdraw(context.Background(), &proto.AuditWithdrawRequest{
		RequestId: createResp.GetRequestId(),
		Decision:  proto.WithdrawAuditDecision_WITHDRAW_AUDIT_DECISION_APPROVE,
	})
	require.NoError(t, err)
	require.Equal(t, string(dao.ChainTokenLedgerStatusPending), approveResp.GetStatus())
	require.Equal(t, "0xapproved", approveResp.GetTxHash())

	depositTokens(t, 81, 200_000)
	createResp, err = svc.RequestWithdraw(context.Background(), &proto.RequestWithdrawRequest{
		PlayerId: 81, TokenAmount: 100_001, Signature: []byte("abcdef"),
	})
	require.NoError(t, err)

	rejectResp, err := svc.AuditWithdraw(context.Background(), &proto.AuditWithdrawRequest{
		RequestId:  createResp.GetRequestId(),
		Decision:   proto.WithdrawAuditDecision_WITHDRAW_AUDIT_DECISION_REJECT,
		FailReason: "manual_reject",
	})
	require.NoError(t, err)
	require.Equal(t, string(dao.ChainTokenLedgerStatusFailed), rejectResp.GetStatus())

	_, err = svc.AuditWithdraw(context.Background(), &proto.AuditWithdrawRequest{
		RequestId:  createResp.GetRequestId(),
		Decision:   proto.WithdrawAuditDecision_WITHDRAW_AUDIT_DECISION_REJECT,
		FailReason: "again",
	})
	require.Error(t, err)
}
