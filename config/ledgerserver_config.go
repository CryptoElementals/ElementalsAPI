package config

import (
	"github.com/CryptoElementals/common/db"
	"github.com/CryptoElementals/common/log"
	"github.com/CryptoElementals/common/redis"
)

var LedgerGConf = LedgerServerConfig{}

type LedgerServerConfig struct {
	LogCfg     log.Config   `mapstructure:"log"`
	DbCfg      db.Config    `mapstructure:"database"`
	RedisCfg   redis.Config `mapstructure:"redis"`
	UChainCfg  UChainConfig `mapstructure:"u-chain"`
	ListenPort int64        `mapstructure:"listen-port"`
}

type UChainConfig struct {
	ChainServerAddress string `mapstructure:"chain-server-address"`
	ChainID            int64  `mapstructure:"chain-id"`
}

func InitLedgerServerConfig(configPath string) error {
	return InitConfig(configPath, &LedgerGConf)
}
