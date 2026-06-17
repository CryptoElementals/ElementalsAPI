package config

import (
	"github.com/CryptoElementals/common/db"
	"github.com/CryptoElementals/common/log"
	"github.com/CryptoElementals/common/redis"
)

var LedgerGConf = LedgerServerConfig{}

type LedgerServerConfig struct {
	LogCfg     log.Config     `mapstructure:"log"`
	DbCfg      db.Config      `mapstructure:"database"`
	RedisCfg   redis.Config   `mapstructure:"redis"`
	ListenPort int64          `mapstructure:"listen-port"`
}

func InitLedgerServerConfig(configPath string) error {
	return InitConfig(configPath, &LedgerGConf)
}
