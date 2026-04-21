package config

import (
	"github.com/CryptoElementals/common/db"
	"github.com/CryptoElementals/common/log"
	"github.com/CryptoElementals/common/redis"
	"github.com/spf13/viper"
)

var RSGConf = RoomServerConfig{}

// RoomChainEntry is one chain the room server may place games on.
// ChainID may be 0: the worker resolves it from the RPC at dial time.
type RoomChainEntry struct {
	ChainID int64 `mapstructure:"chain-id"`
	NodeConfig     `mapstructure:"node"`
	ContractConfig `mapstructure:"contract"`
}

// RoomServerConfig represents the complete application configuration structure
type RoomServerConfig struct {
	LogCfg      log.Config      `mapstructure:"log"`
	RedisCfg    redis.Config  `mapstructure:"redis"`
	DbCfg       db.Config     `mapstructure:"database"`
	Snowflake   SnowflakeConfig `mapstructure:"snowflake"`
	ChainCfg    ChainConfig  `mapstructure:"chain"`
	// Chains is the preferred multi-chain list (http-rpc + room contract per entry).
	// If empty, legacy ChainCfg is treated as a single entry (see EffectiveChains).
	Chains []RoomChainEntry `mapstructure:"chains"`
	WalletPaths []string     `mapstructure:"wallet-paths"`
	ListenPort  int64        `mapstructure:"listen-port"`

	// pool batch size for on-chain submissions
	PoolBatchSize int `mapstructure:"pool-batch-size"`

	PoolProcessingInterval int `mapstructure:"pool-processing-interval"`
	// GameArgsID is the game_args row id used for new matches (must be non-zero; room server loads it at startup).
	GameArgsID uint `mapstructure:"game-args-id"`
}

// EffectiveChains returns the configured chain list: explicit Chains, or one synthetic entry from legacy ChainCfg.
func (c *RoomServerConfig) EffectiveChains() []RoomChainEntry {
	if len(c.Chains) > 0 {
		return c.Chains
	}
	if c.ChainCfg.HttpRpc == "" && c.ChainCfg.RoomV3ContractAddress == "" {
		return nil
	}
	return []RoomChainEntry{{
		ChainID:        0,
		NodeConfig:     c.ChainCfg.NodeConfig,
		ContractConfig: c.ChainCfg.ContractConfig,
	}}
}

func InitRSConfig(configPath string) error {
	viper.SetConfigFile(configPath)
	err := viper.ReadInConfig()
	if err != nil {
		return err
	}
	if err := viper.Unmarshal(&RSGConf); err != nil {
		return err
	}

	return nil
}
