package db

import (
	dao "github.com/CryptoElementals/common/models"
)

// BackfillUserProfileServerTypes sets legacy empty types to normal and renames server_segment column if present.
func BackfillUserProfileServerTypes() error {
	gdb := Get()
	if gdb.Migrator().HasColumn(&dao.UserProfile{}, "server_segment") {
		if err := gdb.Migrator().RenameColumn(&dao.UserProfile{}, "server_segment", "server_type"); err != nil {
			return err
		}
	}
	return gdb.Model(&dao.UserProfile{}).
		Where("server_type = '' OR server_type IS NULL").
		Update("server_type", dao.DefaultServerTypeForExistingUser).Error
}
