package db

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestMigrations(t *testing.T) {
	require.NoError(t, initMemDbSqlite())
	require.NoError(t, MigrateMemDb())
}
