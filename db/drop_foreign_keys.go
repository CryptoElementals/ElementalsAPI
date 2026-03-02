package db

import (
	"fmt"
)

// fkRow holds one row from INFORMATION_SCHEMA.TABLE_CONSTRAINTS for a foreign key.
type fkRow struct {
	TableName      string `gorm:"column:table_name"`
	ConstraintName string `gorm:"column:constraint_name"`
}

// DropAllForeignKeyConstraints discovers and drops all foreign key constraints
// in the current database. Only supported for MySQL; no-op for SQLite (development).
// Call this once if you previously migrated with FKs enabled and want to remove them.
func DropAllForeignKeyConstraints() error {
	dialector := Get().Dialector.Name()
	if dialector != "mysql" {
		return nil
	}

	var rows []fkRow
	// TABLE_SCHEMA = DATABASE() uses the current connection's database
	err := Get().Raw(`
		SELECT TABLE_NAME AS table_name, CONSTRAINT_NAME AS constraint_name
		FROM information_schema.TABLE_CONSTRAINTS
		WHERE CONSTRAINT_TYPE = 'FOREIGN KEY' AND TABLE_SCHEMA = DATABASE()
	`).Scan(&rows).Error
	if err != nil {
		return fmt.Errorf("list foreign keys: %w", err)
	}

	if len(rows) == 0 {
		return nil
	}

	for _, r := range rows {
		// Identifier quoting: backticks for MySQL table and constraint names
		sql := fmt.Sprintf("ALTER TABLE `%s` DROP FOREIGN KEY `%s`", r.TableName, r.ConstraintName)
		if err := Get().Exec(sql).Error; err != nil {
			return fmt.Errorf("drop FK %s on %s: %w", r.ConstraintName, r.TableName, err)
		}
	}

	return nil
}
