package db

import (
	"fmt"

	dao "github.com/CryptoElementals/common/models"
)

const (
	userTokenActivePlayerColumn = "active_player_id"
	userTokenActivePlayerIndex  = "ux_user_tokens_active_player"
)

// EnsureUserTokenActivePlayerUniqueIndex ensures at most one active (deleted_at IS NULL)
// row per player_id. Soft-deleted rows keep active_player_id NULL so multiples are allowed.
func EnsureUserTokenActivePlayerUniqueIndex() error {
	gdb := Get()
	m := gdb.Migrator()
	table := &dao.UserToken{}

	if !m.HasColumn(table, userTokenActivePlayerColumn) {
		var addColSQL string
		switch gdb.Dialector.Name() {
		case "mysql":
			addColSQL = `ALTER TABLE user_tokens ADD COLUMN active_player_id BIGINT
				GENERATED ALWAYS AS (IF(deleted_at IS NULL, player_id, NULL)) VIRTUAL`
		case "sqlite":
			addColSQL = `ALTER TABLE user_tokens ADD COLUMN active_player_id INTEGER
				GENERATED ALWAYS AS (CASE WHEN deleted_at IS NULL THEN player_id ELSE NULL END) VIRTUAL`
		default:
			return fmt.Errorf("unsupported dialect for user_token active unique index: %s", gdb.Dialector.Name())
		}
		if err := gdb.Exec(addColSQL).Error; err != nil {
			return fmt.Errorf("add active_player_id generated column: %w", err)
		}
	}

	if m.HasIndex(table, userTokenActivePlayerIndex) {
		return nil
	}
	createIdxSQL := fmt.Sprintf(
		"CREATE UNIQUE INDEX %s ON user_tokens (%s)",
		userTokenActivePlayerIndex,
		userTokenActivePlayerColumn,
	)
	if err := gdb.Exec(createIdxSQL).Error; err != nil {
		return fmt.Errorf("create %s: %w", userTokenActivePlayerIndex, err)
	}
	return nil
}
