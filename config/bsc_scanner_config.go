package config

import (
	"time"

	"github.com/CryptoElementals/common/db"
	"github.com/CryptoElementals/common/log"
	"github.com/spf13/viper"
)

var BscScannerGConf = BscScannerConfig{}

type BscScannerConfig struct {
	LogCfg                        log.Config     `mapstructure:"log"`
	DbCfg                         db.Config      `mapstructure:"database"`
	ChainCfg                      BscChainConfig `mapstructure:"chain"`
	WorkerCount                   int            `mapstructure:"worker-count"`
	WalletRegistryRefreshInterval time.Duration  `mapstructure:"wallet-registry-refresh-interval"`
}

type BscChainConfig struct {
	NodeConfig           `mapstructure:"node"`
	WalletManagerAddress string `mapstructure:"wallet-manager-address"`
}

func (c BscChainConfig) SyncType() string {
	return "finalized"
}

func InitBscScannerConfig(configPath string) error {
	viper.SetConfigFile(configPath)
	if err := viper.ReadInConfig(); err != nil {
		return err
	}
	return viper.Unmarshal(&BscScannerGConf)
}
