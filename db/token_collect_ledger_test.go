package db

import (
	"testing"

	dao "github.com/CryptoElementals/common/models"
	"github.com/stretchr/testify/require"
)

func TestInsertTokenCollectLedger(t *testing.T) {
	require.NoError(t, initMemDbSqlite())
	require.NoError(t, MigrateMemDb())

	id, err := InsertTokenCollectLedger(&dao.TokenCollectLedger{
		PlayerID:         42,
		PlayerAddress:    "0xAbCdEf0000000000000000000000000000000001",
		WalletIndex:      3,
		CollectorAddress: "0xCc49255a2639560171fc28b09DCd6CdC3b25597C",
		TokenAmount:      "1000000000000000000",
		TxHash:           "0xABC123",
		ChainID:          97,
	})
	require.NoError(t, err)
	require.NotZero(t, id)

	var row dao.TokenCollectLedger
	require.NoError(t, Get().First(&row, id).Error)
	require.EqualValues(t, 42, row.PlayerID)
	require.Equal(t, "0xabcdef0000000000000000000000000000000001", row.PlayerAddress)
	require.Equal(t, "0xcc49255a2639560171fc28b09dcd6cdc3b25597c", row.CollectorAddress)
	require.Equal(t, "0xabc123", row.TxHash)
}
