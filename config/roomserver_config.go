package config

import (
	"github.com/CryptoElementals/common/db"
	"github.com/CryptoElementals/common/log"
	"github.com/CryptoElementals/common/redis"
	"github.com/spf13/viper"
)

var RSGConf = RoomServerConfig{}

// RoomServerConfig represents the complete application configuration structure
type RoomServerConfig struct {
	LogCfg      log.Config   `mapstructure:"log"`
	RedisCfg    redis.Config `mapstructure:"redis"`
	DbCfg       db.Config    `mapstructure:"database"`
	ChainCfg    ChainConfig  `mapstructure:"chain"`
	WalletPaths []string     `mapstructure:"wallet-paths"`
	ListenPort  int64        `mapstructure:"listen-port"`

	// pool batch size for on-chain submissions
	PoolBatchSize int `mapstructure:"pool-batch-size"`

	PoolProcessingInterval int `mapstructure:"pool-processing-interval"`
	// GameArgsID is the game_args row id used for new matches (must be non-zero; room server loads it at startup).
	GameArgsID uint `mapstructure:"game-args-id"`
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
