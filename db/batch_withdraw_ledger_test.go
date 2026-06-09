package db

import (
	"testing"

	dao "github.com/CryptoElementals/common/models"
	"github.com/stretchr/testify/require"
)

func TestInsertBatchWithdrawLedger(t *testing.T) {
	require.NoError(t, initMemDbSqlite())
	require.NoError(t, MigrateMemDb())

	id, err := InsertBatchWithdrawLedger(&dao.BatchWithdrawLedger{
		PlayerID:         42,
		Amount:           1_000_000_000_000_000_000,
		Signature:        "0xAB01",
		CollectorAddress: "0xCc49255a2639560171fc28b09DCd6CdC3b25597C",
		ChainID:          97,
		TxHash:           "0xABC123",
	})
	require.NoError(t, err)
	require.NotZero(t, id)

	var row dao.BatchWithdrawLedger
	require.NoError(t, Get().First(&row, id).Error)
	require.EqualValues(t, 42, row.PlayerID)
	require.EqualValues(t, 1_000_000_000_000_000_000, row.Amount)
	require.Equal(t, "0xab01", row.Signature)
	require.Equal(t, "0xcc49255a2639560171fc28b09dcd6cdc3b25597c", row.CollectorAddress)
	require.Equal(t, "0xabc123", row.TxHash)
}
