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
	LogCfg          log.Config      `mapstructure:"log"`
	RedisCfg        redis.Config    `mapstructure:"redis"`
	DbCfg           db.Config       `mapstructure:"database"`
	ChainCfg        ChainConfig     `mapstructure:"chain"`
	GameParams      GameParamConfig `mapstructure:"game-params"`
	WalletPaths     []string        `mapstructure:"wallet-paths"`
	RoundTimeout    int64           `mapstructure:"round-timeout"`
	ContinueTimeout int64           `mapstructure:"continue-timeout"`
	MaxRounds       int64           `mapstructure:"max-rounds"`
	GameInitialHP   int64           `mapstructure:"game-initial-hp"`
	ListenPort      int64           `mapstructure:"listen-port"`
	BotWaitTime     int64           `mapstructure:"bot-wait-time"`
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
