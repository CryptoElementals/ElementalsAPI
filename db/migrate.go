package db

func Migrate() error {
	migrates := []any{}
	err := db.Set("gorm:table_options", "ENGINE=InnoDB").AutoMigrate(migrates...)
	if err != nil {
		return err
	}

	return nil
}
