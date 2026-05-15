package config

import (
	"testing"

	"github.com/CryptoElementals/common/db"
	"github.com/CryptoElementals/common/log"
	"github.com/CryptoElementals/common/redis"
)

func TestValidateApiServerConfigRequiresTrialAndNormal(t *testing.T) {
	cfg := &ApiServerConfig{
		LogCfg: log.Config{
			Level:        "debug",
			Dir:          "./logs",
			Prefix:       "test",
			Suffix:       "log",
			MaxAge:       1,
			RotationTime: 24,
		},
		DbCfg: db.Config{
			Endpoint: "localhost:3306",
			User:     "root",
			DbName:   "test",
		},
		RedisCfg: redis.Config{Address: "localhost:6379", Size: 10},
		ServerCfg: ServerConfig{
			Port:               8080,
			ServerMode:         "debug",
			SessionMaxAge:      180,
			RefreshTokenMaxAge: 300,
		},
		S3Config: S3Config{
			AccessKeyID:     "test",
			SecretAccessKey: "test",
			Region:          "us-east-1",
			Endpoint:        "http://localhost",
			Bucket:          "test",
			PresignExpire:   3600,
		},
		EnvironmentConfigs: []EnvironmentConfig{
			{
				Name:               ServerTypeTrial,
				RedisCfg:           redis.Config{Address: "localhost:6379", Size: 10},
				RoomServerAddress:  "r:1",
				LobbyServerAddress: "l:1",
			},
		},
	}
	if err := ValidateApiServerConfig(cfg); err == nil {
		t.Fatal("expected validation error when normal env is missing")
	}

	cfg.EnvironmentConfigs = append(cfg.EnvironmentConfigs, EnvironmentConfig{
		Name:               ServerTypeNormal,
		RedisCfg:           redis.Config{Address: "localhost:6380", Size: 10},
		RoomServerAddress:  "r:2",
		LobbyServerAddress: "l:2",
	})
	if err := ValidateApiServerConfig(cfg); err != nil {
		t.Fatalf("unexpected validation error: %v", err)
	}
}
