package config

import (
	"github.com/CryptoElementals/common/db"
	"github.com/CryptoElementals/common/log"
)

var CSGConf = ChainServerConfig{}

type ChainEntry struct {
	ChainID int64 `mapstructure:"chain-id"`
	NodeConfig     `mapstructure:"node"`
	ContractConfig `mapstructure:"contract"`
}

type WalletChainConfig struct {
	ChainID int64 `mapstructure:"chain-id"`
	NodeConfig `mapstructure:"node"`
	WalletManagerAddress string `mapstructure:"wallet-manager-address"`
}

type ChainServerConfig struct {
	LogCfg log.Config `mapstructure:"log"`
	DbCfg  db.Config  `mapstructure:"database"`

	ListenPort int64 `mapstructure:"listen-port"`

	WalletPaths []string `mapstructure:"wallet-paths"`

	PoolBatchSize          int `mapstructure:"pool-batch-size"`
	PoolProcessingInterval int `mapstructure:"pool-processing-interval"`
	PoolClaimTimeoutSeconds int `mapstructure:"pool-claim-timeout-seconds"`

	Chains []ChainEntry `mapstructure:"chains"`

	WalletChain *WalletChainConfig `mapstructure:"wallet-chain"`
}

func (c *ChainServerConfig) EffectiveChains() []ChainEntry {
	return c.Chains
}

func InitCSConfig(configPath string) error {
	return InitConfig(configPath, &CSGConf)
}
