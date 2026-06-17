package config

import (
	"github.com/CryptoElementals/common/db"
	"github.com/CryptoElementals/common/log"
)

var LedgerGConf = LedgerServerConfig{}

type LedgerServerConfig struct {
	LogCfg     log.Config `mapstructure:"log"`
	DbCfg      db.Config  `mapstructure:"database"`
	ListenPort int64      `mapstructure:"listen-port"`
}

func InitLedgerServerConfig(configPath string) error {
	return InitConfig(configPath, &LedgerGConf)
}
