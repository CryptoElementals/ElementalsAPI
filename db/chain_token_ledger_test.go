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

	const amountWei = "1000000000000000000" // 1000 game tokens
	ev := chainDepositEvent(97, "0xtx1", 1, 42, amountWei)

	result, err := ApplyChainTokenEvent(context.Background(), ev)
	require.NoError(t, err)
	require.Equal(t, ChainTokenEventApplyApplied, result.Status)
	require.Equal(t, int32(1000), result.TokenDelta)
	require.Equal(t, int32(1000), result.NewBalance)

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
	require.Equal(t, ChainTokenEventApplyApplied, result.Status)
	require.Equal(t, int32(-500), result.TokenDelta)
	require.Equal(t, int32(500), result.NewBalance)
}

func TestApplyChainTokenEvent_WithdrawInsufficientBalanceRejected(t *testing.T) {
	require.NoError(t, initMemDbSqlite())
	require.NoError(t, MigrateMemDb())

	depositWei := "500000000000000000" // 500 game tokens
	_, err := ApplyChainTokenEvent(context.Background(), chainDepositEvent(97, "0xtxdep2", 1, 9, depositWei))
	require.NoError(t, err)

	withdrawWei := "1000000000000000000" // 1000 game tokens
	result, err := ApplyChainTokenEvent(context.Background(), chainWithdrawEvent(97, "0xtxwd2", 2, 9, withdrawWei))
	require.NoError(t, err)
	require.Equal(t, ChainTokenEventApplyRejected, result.Status)
	require.Equal(t, chainTokenRejectInsufficientBalance, result.Message)

	token, err := EnsureUserTokenByPlayerID(9)
	require.NoError(t, err)
	require.Equal(t, int32(500), token.TokenAmount)

	dup, err := ApplyChainTokenEvent(context.Background(), chainWithdrawEvent(97, "0xtxwd2", 2, 9, withdrawWei))
	require.NoError(t, err)
	require.Equal(t, ChainTokenEventApplyRejected, dup.Status)
}
