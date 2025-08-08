package db

import (
	"errors"
	"fmt"
	"time"

	"gorm.io/driver/mysql"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	gorm_logger "gorm.io/gorm/logger"
)

var ErrNotFound error = errors.New("not found")

var DEFAULT_TOKEN_TARGET uint64 = 36000
var DEFAULT_REWARD_FEE_RATE uint8 = 20
var ANNUAL_PERCENTAGE_RATE uint8 = 20
var MIN_STAKED_AMOUNT uint64 = 100

var db *gorm.DB

type Config struct {
	Endpoint    string `mapstructure:"endpoint"`
	User        string `mapstructure:"user"`
	Password    string `mapstructure:"password"`
	DbName      string `mapstructure:"db-name"`
	Development bool   `mapstructure:"development"`
}

func Init(cfg *Config) error {
	if cfg.Development {
		return initMemDbSqlite()
	}
	return initMysql(cfg)
}

func initMysql(cfg *Config) error {
	dsn := fmt.Sprintf("%s:%s@tcp(%s)/%s?multiStatements=true&charset=utf8mb4&parseTime=true", cfg.User,
		cfg.Password, cfg.Endpoint, cfg.DbName)
	ldb, err := gorm.Open(mysql.Open(dsn), &gorm.Config{
		Logger: gorm_logger.Default.LogMode(gorm_logger.Error),
	})
	if err != nil {
		return err
	}
	db = ldb

	// set connection pool parameters
	sqlDB, err := db.DB()
	if err != nil {
		return err
	}
	sqlDB.SetMaxIdleConns(50)
	sqlDB.SetMaxOpenConns(100)
	sqlDB.SetConnMaxLifetime(time.Hour)

	return nil
}

func initMemDbSqlite() error {
	ldb, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{
		Logger:                                   gorm_logger.Default.LogMode(gorm_logger.Info),
		DisableForeignKeyConstraintWhenMigrating: true,
	})
	if err != nil {
		return err
	}
	db = ldb
	sqlDB, err := db.DB()
	if err != nil {
		return err
	}
	sqlDB.SetMaxOpenConns(1)
	return nil
}

func Get() *gorm.DB {
	return db
}
