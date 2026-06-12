package db

import (
	"testing"

	dao "github.com/CryptoElementals/common/models"
	"github.com/stretchr/testify/require"
)

func TestUpsertTokenCollectorAddress_Idempotent(t *testing.T) {
	require.NoError(t, initMemDbSqlite())
	require.NoError(t, MigrateMemDb())

	row := dao.TokenCollectorAddress{
		ChainID: 97,
		Address: "0xCc49255a2639560171fc28b09DCd6CdC3b25597C2",
		Source:  dao.TokenCollectorSourceOnChainRefresh,
	}
	inserted, err := UpsertTokenCollectorAddress(row)
	require.NoError(t, err)
	require.True(t, inserted)

	inserted, err = UpsertTokenCollectorAddress(row)
	require.NoError(t, err)
	require.False(t, inserted)

	rows, err := ListTokenCollectorAddresses(97)
	require.NoError(t, err)
	require.Len(t, rows, 1)
	require.Equal(t, "0xcc49255a2639560171fc28b09dcd6cdc3b25597c2", rows[0].Address)
}
