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
	LogCfg     log.Config      `mapstructure:"log"`
	RedisCfg   redis.Config    `mapstructure:"redis"`
	DbCfg      db.Config       `mapstructure:"database"`
	ServerCfg  ServerConfig    `mapstructure:"server"`
	ChainCfg   ChainConfig     `mapstructure:"chain"`
	GameParams GameParamConfig `mapstructure:"game-params"`
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

	// 初始化游戏参数
	InitializeGameParams(&RSGConf.GameParams)

	return nil
}
