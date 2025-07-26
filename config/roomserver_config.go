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
	LogCfg        log.Config   `mapstructure:"log"`
	RedisCfg      redis.Config `mapstructure:"redis"`
	DbCfg         db.Config    `mapstructure:"database"`
	ChainCfg      ChainConfig  `mapstructure:"chain"`
	WalletPath    string       `mapstructure:"wallet-path"`
	RoundTimeout  int64        `mapstructure:"round-timeout"`
	MaxRounds     int64        `mapstructure:"max-rounds"`
	GameInitialHP int64        `mapstructure:"game-initial-hp"`
	ListenPort    int64        `mapstructure:"listen-port"`
}

func InitRSConfig(configPath string) error {
	viper.SetConfigFile(configPath)
	err := viper.ReadInConfig()
	if err != nil {
		return err
	}
	return viper.Unmarshal(&RSGConf)
}
