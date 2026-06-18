package db

import (
	"context"
	"testing"

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

	const amountWei = "1000000000000000000" // 100 game tokens
	ev := chainDepositEvent(97, "0xtx1", 1, 42, amountWei)

	result, err := ApplyChainTokenEvent(context.Background(), ev)
	require.NoError(t, err)
	require.Equal(t, ChainTokenEventApplyFinalized, result.Status)
	require.Equal(t, int32(100), result.TokenDelta)
	require.Equal(t, int32(100), result.NewBalance)

	dup, err := ApplyChainTokenEvent(context.Background(), ev)
	require.NoError(t, err)
	require.Equal(t, ChainTokenEventApplyDuplicate, dup.Status)

	token, err := EnsureUserTokenByPlayerID(42)
	require.NoError(t, err)
	require.Equal(t, int32(100), token.TokenAmount)
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
	require.Equal(t, int32(-50), result.TokenDelta)
	require.Equal(t, int32(50), result.NewBalance)
}

func TestApplyChainTokenEvent_WithdrawInsufficientBalanceFailed(t *testing.T) {
	require.NoError(t, initMemDbSqlite())
	require.NoError(t, MigrateMemDb())

	depositWei := "500000000000000000" // 50 game tokens
	_, err := ApplyChainTokenEvent(context.Background(), chainDepositEvent(97, "0xtxdep2", 1, 9, depositWei))
	require.NoError(t, err)

	withdrawWei := "1000000000000000000" // 100 game tokens
	result, err := ApplyChainTokenEvent(context.Background(), chainWithdrawEvent(97, "0xtxwd2", 2, 9, withdrawWei))
	require.NoError(t, err)
	require.Equal(t, ChainTokenEventApplyFailed, result.Status)
	require.Equal(t, chainTokenFailInsufficientBalance, result.Message)

	token, err := EnsureUserTokenByPlayerID(9)
	require.NoError(t, err)
	require.Equal(t, int32(50), token.TokenAmount)

	dup, err := ApplyChainTokenEvent(context.Background(), chainWithdrawEvent(97, "0xtxwd2", 2, 9, withdrawWei))
	require.NoError(t, err)
	require.Equal(t, ChainTokenEventApplyFailed, dup.Status)
}

func TestCreatePendingWithdrawAndFinalize(t *testing.T) {
	require.NoError(t, initMemDbSqlite())
	require.NoError(t, MigrateMemDb())

	const depositWei = "2000000000000000000" // 200 tokens
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
	require.Equal(t, int32(200), token.TokenAmount)

	require.NoError(t, UpdatePendingWithdrawTxHash(context.Background(), pending.RequestID, "0xtxpending", "0xcollector"))

	result, err := FinalizeChainTokenWithdraw(context.Background(), chainWithdrawEvent(97, "0xtxpending", 3, 11, withdrawWei))
	require.NoError(t, err)
	require.Equal(t, ChainTokenEventApplyFinalized, result.Status)
	require.Equal(t, int32(-50), result.TokenDelta)
	require.Equal(t, int32(150), result.NewBalance)

	token, err = EnsureUserTokenByPlayerID(11)
	require.NoError(t, err)
	require.Equal(t, int32(150), token.TokenAmount)
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
