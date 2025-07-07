package test

import (
	"testing"

	"github.com/CryptoElementals/common/config"
	"github.com/CryptoElementals/common/db"
	"github.com/CryptoElementals/common/log"
)

func TestMain(m *testing.M) {
	// 加载完整配置
	cfg, err := config.LoadAppConfig("../config.yaml")
	if err != nil {
		panic(err)
	}

	log.InitGlobalLogger(&cfg.LogCfg)
	db.Init(&cfg.DbCfg)
	m.Run()
}

// 测试数据库连接
func TestDatabaseConnection(t *testing.T) {
	dbInstance := db.Get()
	if dbInstance == nil {
		t.Fatal("Database connection is nil")
	}

	// 测试数据库连接是否正常
	sqlDB, err := dbInstance.DB()
	if err != nil {
		t.Fatalf("Failed to get underlying sql.DB: %v", err)
	}

	err = sqlDB.Ping()
	if err != nil {
		t.Fatalf("Failed to ping database: %v", err)
	}

	t.Log("Database connection test passed")
}
