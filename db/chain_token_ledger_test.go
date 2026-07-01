package db

import (
	"context"
	"testing"

	"github.com/CryptoElementals/common/internal/tokenunits"
	dao "github.com/CryptoElementals/common/models"
	"github.com/stretchr/testify/require"
)

func chainDepositEvent(chainID int64, txHash string, logIndex uint32, playerID int64, amountWei string) ChainTokenEventInput {
	return ChainTokenEventInput{
		ChainID:          chainID,
		BlockNumber:      100,
		BlockHash:        "0xabc",
		Timestamp:        1700000000,
		TxHash:           txHash,
		LogIndex:         logIndex,
		CollectorAddress: "0xcollector",
		EventType:        dao.ChainTokenLedgerEventDeposit,
		PlayerID:         playerID,
		AmountWei:        amountWei,
		FromAddress:      "0xfrom",
		NewCreditedWei:   amountWei,
	}
}

func chainWithdrawEvent(chainID int64, txHash string, logIndex uint32, playerID int64, amountWei string) ChainTokenEventInput {
	return ChainTokenEventInput{
		ChainID:          chainID,
		BlockNumber:      101,
		BlockHash:        "0xdef",
		Timestamp:        1700000001,
		TxHash:           txHash,
		LogIndex:         logIndex,
		CollectorAddress: "0xcollector",
		EventType:        dao.ChainTokenLedgerEventWithdraw,
		PlayerID:         playerID,
		AmountWei:        amountWei,
		Operator:         "0xop",
		ToAddress:        "0xto",
	}
}

func TestApplyChainTokenEvent_DepositCreditsUserToken(t *testing.T) {
	require.NoError(t, initMemDbSqlite())
	require.NoError(t, MigrateMemDb())

	const amountWei = "1000000000000000000" // 1000 game tokens
	ev := chainDepositEvent(97, "0xtx1", 1, 42, amountWei)

	result, err := ApplyChainTokenEvent(context.Background(), ev)
	require.NoError(t, err)
	require.Equal(t, ChainTokenEventApplyFinalized, result.Status)
	require.Equal(t, int32(1000), result.TokenDelta)
	require.Equal(t, int32(1000), result.NewBalance)
	require.Equal(t, "0xfrom", result.DepositAddress)

	dup, err := ApplyChainTokenEvent(context.Background(), ev)
	require.NoError(t, err)
	require.Equal(t, ChainTokenEventApplyDuplicate, dup.Status)

	token, err := EnsureUserTokenByPlayerID(42)
	require.NoError(t, err)
	require.Equal(t, int32(1000), token.TokenAmount)
}

func TestApplyChainTokenEvent_WithdrawDeductsUserToken(t *testing.T) {
	require.NoError(t, initMemDbSqlite())
	require.NoError(t, MigrateMemDb())

	const amountWei = "1000000000000000000"
	_, err := ApplyChainTokenEvent(context.Background(), chainDepositEvent(97, "0xtxdep", 1, 7, amountWei))
	require.NoError(t, err)

	withdrawWei := "500000000000000000"
	result, err := ApplyChainTokenEvent(context.Background(), chainWithdrawEvent(97, "0xtxwd", 2, 7, withdrawWei))
	require.NoError(t, err)
	require.Equal(t, ChainTokenEventApplyFinalized, result.Status)
	require.Equal(t, int32(-500), result.TokenDelta)
	require.Equal(t, int32(500), result.NewBalance)
}

func TestApplyChainTokenEvent_WithdrawInsufficientBalanceFailed(t *testing.T) {
	require.NoError(t, initMemDbSqlite())
	require.NoError(t, MigrateMemDb())

	depositWei := "500000000000000000" // 500 game tokens
	_, err := ApplyChainTokenEvent(context.Background(), chainDepositEvent(97, "0xtxdep2", 1, 9, depositWei))
	require.NoError(t, err)

	withdrawWei := "1000000000000000000" // 1000 game tokens
	result, err := ApplyChainTokenEvent(context.Background(), chainWithdrawEvent(97, "0xtxwd2", 2, 9, withdrawWei))
	require.NoError(t, err)
	require.Equal(t, ChainTokenEventApplyFailed, result.Status)
	require.Equal(t, chainTokenFailInsufficientBalance, result.Message)

	token, err := EnsureUserTokenByPlayerID(9)
	require.NoError(t, err)
	require.Equal(t, int32(500), token.TokenAmount)

	dup, err := ApplyChainTokenEvent(context.Background(), chainWithdrawEvent(97, "0xtxwd2", 2, 9, withdrawWei))
	require.NoError(t, err)
	require.Equal(t, ChainTokenEventApplyFailed, dup.Status)
}

func TestCreatePendingWithdrawAndFinalize(t *testing.T) {
	require.NoError(t, initMemDbSqlite())
	require.NoError(t, MigrateMemDb())

	const depositWei = "2000000000000000000" // 2000 tokens
	_, err := ApplyChainTokenEvent(context.Background(), chainDepositEvent(97, "0xdep3", 1, 11, depositWei))
	require.NoError(t, err)

	const withdrawWei = "500000000000000000"
	pending, err := CreatePendingWithdraw(context.Background(), PendingWithdrawInput{
		ChainID:   97,
		PlayerID:  11,
		AmountWei: withdrawWei,
		Signature: "0xabc",
	})
	require.NoError(t, err)
	require.NotEmpty(t, pending.RequestID)

	token, err := EnsureUserTokenByPlayerID(11)
	require.NoError(t, err)
	require.Equal(t, int32(2000), token.TokenAmount)

	require.NoError(t, UpdatePendingWithdrawTxHash(context.Background(), pending.RequestID, "0xtxpending", "0xcollector"))

	result, err := FinalizeChainTokenWithdraw(context.Background(), chainWithdrawEvent(97, "0xtxpending", 3, 11, withdrawWei))
	require.NoError(t, err)
	require.Equal(t, ChainTokenEventApplyFinalized, result.Status)
	require.Equal(t, int32(-500), result.TokenDelta)
	require.Equal(t, int32(1500), result.NewBalance)

	token, err = EnsureUserTokenByPlayerID(11)
	require.NoError(t, err)
	require.Equal(t, int32(1500), token.TokenAmount)
}

func TestCreatePendingWithdrawInsufficientAvailableBalance(t *testing.T) {
	require.NoError(t, initMemDbSqlite())
	require.NoError(t, MigrateMemDb())

	const depositWei = "500000000000000000"
	_, err := ApplyChainTokenEvent(context.Background(), chainDepositEvent(97, "0xdep4", 1, 12, depositWei))
	require.NoError(t, err)

	_, err = CreatePendingWithdraw(context.Background(), PendingWithdrawInput{
		ChainID:   97,
		PlayerID:  12,
		AmountWei: "400000000000000000",
		Signature: "0xabc",
	})
	require.NoError(t, err)

	_, err = CreatePendingWithdraw(context.Background(), PendingWithdrawInput{
		ChainID:   97,
		PlayerID:  12,
		AmountWei: "300000000000000000",
		Signature: "0xdef",
	})
	require.ErrorIs(t, err, ErrInsufficientAvailableBalance)
}

func TestGetWithdrawableTokenAmountFullBalance(t *testing.T) {
	require.NoError(t, initMemDbSqlite())
	require.NoError(t, MigrateMemDb())

	const depositWei = "1000000000000000000" // 1000 tokens
	_, err := ApplyChainTokenEvent(context.Background(), chainDepositEvent(97, "0xwd-full", 1, 30, depositWei))
	require.NoError(t, err)

	got, err := GetWithdrawableTokenAmount(context.Background(), 30)
	require.NoError(t, err)
	require.Equal(t, int32(1000), got.WithdrawableTokenAmount)
	require.Equal(t, int32(1000), got.TokenAmount)
	require.Equal(t, int32(0), got.LockedTokens)
	require.Equal(t, int32(0), got.PendingWithdrawTokenAmount)
}

func TestGetWithdrawableTokenAmountPendingWithdraw(t *testing.T) {
	require.NoError(t, initMemDbSqlite())
	require.NoError(t, MigrateMemDb())

	const depositWei = "500000000000000000" // 500 tokens
	_, err := ApplyChainTokenEvent(context.Background(), chainDepositEvent(97, "0xwd-pend", 1, 31, depositWei))
	require.NoError(t, err)

	_, err = CreatePendingWithdraw(context.Background(), PendingWithdrawInput{
		ChainID: 97, PlayerID: 31, AmountWei: "400000000000000000", Signature: "0xabc",
	})
	require.NoError(t, err)

	got, err := GetWithdrawableTokenAmount(context.Background(), 31)
	require.NoError(t, err)
	require.Equal(t, int32(100), got.WithdrawableTokenAmount)
	require.Equal(t, int32(500), got.TokenAmount)
	require.Equal(t, int32(0), got.LockedTokens)
	require.Equal(t, int32(400), got.PendingWithdrawTokenAmount)
}

func TestGetWithdrawableTokenAmountLockedTokens(t *testing.T) {
	require.NoError(t, initMemDbSqlite())
	require.NoError(t, MigrateMemDb())

	const depositWei = "1000000000000000000" // 1000 tokens
	_, err := ApplyChainTokenEvent(context.Background(), chainDepositEvent(97, "0xwd-lock", 1, 32, depositWei))
	require.NoError(t, err)

	require.NoError(t, LockUserToken(context.Background(), 32, "0xtemp", 30, ""))

	got, err := GetWithdrawableTokenAmount(context.Background(), 32)
	require.NoError(t, err)
	require.Equal(t, int32(970), got.WithdrawableTokenAmount)
	require.Equal(t, int32(1000), got.TokenAmount)
	require.Equal(t, int32(30), got.LockedTokens)
	require.Equal(t, int32(0), got.PendingWithdrawTokenAmount)
}

func TestListChainTokenLedgersFilter(t *testing.T) {
	require.NoError(t, initMemDbSqlite())
	require.NoError(t, MigrateMemDb())

	_, err := ApplyChainTokenEvent(context.Background(), chainDepositEvent(97, "0xlist1", 1, 20, "1000000000000000000"))
	require.NoError(t, err)

	list, err := ListChainTokenLedgers(context.Background(), ChainTokenLedgerFilter{
		PlayerID:  20,
		EventType: string(dao.ChainTokenLedgerEventDeposit),
		Status:    string(dao.ChainTokenLedgerStatusFinalized),
		Limit:     10,
	})
	require.NoError(t, err)
	require.Equal(t, int64(1), list.Total)
	require.Len(t, list.Records, 1)
	require.Equal(t, dao.ChainTokenLedgerStatusFinalized, list.Records[0].Status)
}

func TestCreateAuditingWithdrawReservesBalance(t *testing.T) {
	require.NoError(t, initMemDbSqlite())
	require.NoError(t, MigrateMemDb())

	depositWei, err := tokenunits.TokenToWei(200_000)
	require.NoError(t, err)
	_, err = ApplyChainTokenEvent(context.Background(), chainDepositEvent(97, "0xdep-audit", 1, 40, depositWei))
	require.NoError(t, err)

	withdrawWei, err := tokenunits.TokenToWei(100_001)
	require.NoError(t, err)
	_, err = CreateAuditingWithdraw(context.Background(), PendingWithdrawInput{
		ChainID: 97, PlayerID: 40, AmountWei: withdrawWei, Signature: "0xabc",
	})
	require.NoError(t, err)

	got, err := GetWithdrawableTokenAmount(context.Background(), 40)
	require.NoError(t, err)
	require.Equal(t, int32(99_999), got.WithdrawableTokenAmount)
	require.Equal(t, int32(100_001), got.PendingWithdrawTokenAmount)
}

func TestRejectAuditingWithdrawReleasesBalance(t *testing.T) {
	require.NoError(t, initMemDbSqlite())
	require.NoError(t, MigrateMemDb())

	depositWei, err := tokenunits.TokenToWei(200_000)
	require.NoError(t, err)
	_, err = ApplyChainTokenEvent(context.Background(), chainDepositEvent(97, "0xdep-rej", 1, 41, depositWei))
	require.NoError(t, err)

	withdrawWei, err := tokenunits.TokenToWei(100_001)
	require.NoError(t, err)
	pending, err := CreateAuditingWithdraw(context.Background(), PendingWithdrawInput{
		ChainID: 97, PlayerID: 41, AmountWei: withdrawWei, Signature: "0xabc",
	})
	require.NoError(t, err)

	require.NoError(t, RejectAuditingWithdraw(context.Background(), pending.RequestID, "manual_reject"))

	got, err := GetWithdrawableTokenAmount(context.Background(), 41)
	require.NoError(t, err)
	require.Equal(t, int32(200_000), got.WithdrawableTokenAmount)
	require.Equal(t, int32(0), got.PendingWithdrawTokenAmount)
}

func TestAuditingWithdrawBlocksNewWithdraw(t *testing.T) {
	require.NoError(t, initMemDbSqlite())
	require.NoError(t, MigrateMemDb())

	depositWei, err := tokenunits.TokenToWei(300_000)
	require.NoError(t, err)
	_, err = ApplyChainTokenEvent(context.Background(), chainDepositEvent(97, "0xdep-block", 1, 42, depositWei))
	require.NoError(t, err)

	withdrawWei, err := tokenunits.TokenToWei(100_001)
	require.NoError(t, err)
	_, err = CreateAuditingWithdraw(context.Background(), PendingWithdrawInput{
		ChainID: 97, PlayerID: 42, AmountWei: withdrawWei, Signature: "0xabc",
	})
	require.NoError(t, err)

	smallWei, err := tokenunits.TokenToWei(1_000)
	require.NoError(t, err)
	_, err = CreatePendingWithdraw(context.Background(), PendingWithdrawInput{
		ChainID: 97, PlayerID: 42, AmountWei: smallWei, Signature: "0xdef",
	})
	require.ErrorIs(t, err, ErrAuditingWithdrawInProgress)
}

func TestApproveAuditingWithdrawTransitionsToPending(t *testing.T) {
	require.NoError(t, initMemDbSqlite())
	require.NoError(t, MigrateMemDb())

	depositWei, err := tokenunits.TokenToWei(200_000)
	require.NoError(t, err)
	_, err = ApplyChainTokenEvent(context.Background(), chainDepositEvent(97, "0xdep-appr", 1, 43, depositWei))
	require.NoError(t, err)

	withdrawWei, err := tokenunits.TokenToWei(100_001)
	require.NoError(t, err)
	pending, err := CreateAuditingWithdraw(context.Background(), PendingWithdrawInput{
		ChainID: 97, PlayerID: 43, AmountWei: withdrawWei, Signature: "0xabc",
	})
	require.NoError(t, err)

	row, err := ApproveAuditingWithdraw(context.Background(), pending.RequestID)
	require.NoError(t, err)
	require.Equal(t, int64(43), row.PlayerID)

	var ledger dao.ChainTokenLedger
	require.NoError(t, Get().First(&ledger, pending.LedgerID).Error)
	require.Equal(t, dao.ChainTokenLedgerStatusPending, ledger.Status)
}

func TestListChainTokenLedgersCrossPlayer(t *testing.T) {
	require.NoError(t, initMemDbSqlite())
	require.NoError(t, MigrateMemDb())

	depositWei, err := tokenunits.TokenToWei(200_000)
	require.NoError(t, err)
	_, err = ApplyChainTokenEvent(context.Background(), chainDepositEvent(97, "0xlist-a", 1, 21, depositWei))
	require.NoError(t, err)
	_, err = ApplyChainTokenEvent(context.Background(), chainDepositEvent(97, "0xlist-b", 1, 22, depositWei))
	require.NoError(t, err)

	withdrawWei, err := tokenunits.TokenToWei(100_001)
	require.NoError(t, err)
	_, err = CreateAuditingWithdraw(context.Background(), PendingWithdrawInput{
		ChainID: 97, PlayerID: 21, AmountWei: withdrawWei, Signature: "0xabc",
	})
	require.NoError(t, err)

	list, err := ListChainTokenLedgers(context.Background(), ChainTokenLedgerFilter{
		PlayerID:  0,
		Status:    string(dao.ChainTokenLedgerStatusAuditing),
		EventType: string(dao.ChainTokenLedgerEventWithdraw),
		Limit:     20,
	})
	require.NoError(t, err)
	require.Equal(t, int64(1), list.Total)
	require.Equal(t, int64(21), list.Records[0].PlayerID)
}
