package config

import (
	"github.com/CryptoElementals/common/db"
	"github.com/CryptoElementals/common/log"
	"github.com/CryptoElementals/common/redis"
	"github.com/CryptoElementals/common/server"
	"github.com/spf13/viper"
)

var RSGConf = RoomServerConfig{}

// RoomServerConfig represents the complete application configuration structure
type RoomServerConfig struct {
	LogCfg        log.Config    `mapstructure:"log"`
	RedisCfg      redis.Config  `mapstructure:"redis"`
	DbCfg         db.Config     `mapstructure:"database"`
	ServerCfg     server.Config `mapstructure:"server"`
	ChainCfg      ChainConfig   `mapstructure:"chain"`
	WalletPath    string        `mapstructure:"wallet_path"`
	RoundTimeout  int64         `mapstructure:"round_timeout"`
	MaxRounds     int64         `mapstructure:"max_rounds"`
	GameInitialHP int64         `mapstructure:"game_initial_hp"`
	ListenPort    int64         `mapstructure:"listen_port"`
}

func InitRSConfig(configPath string) error {
	viper.SetConfigFile(configPath)
	err := viper.ReadInConfig()
	if err != nil {
		return err
	}
	return viper.Unmarshal(&RSGConf)
}
