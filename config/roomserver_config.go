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
	LogCfg              log.Config      `mapstructure:"log"`
	RedisCfg            redis.Config    `mapstructure:"redis"`
	DbCfg               db.Config       `mapstructure:"database"`
	ChainCfg            ChainConfig     `mapstructure:"chain"`
	GameParams          GameParamConfig `mapstructure:"game-params"`
	WalletPaths         []string        `mapstructure:"wallet-paths"`
	ListenPort          int64           `mapstructure:"listen-port"`
	BotWaitTime         int64           `mapstructure:"bot-wait-time"`
	StatServiceEndpoint string          `mapstructure:"stat-service-endpoint"`
	ShouldRecoverGames  bool            `mapstructure:"should-recover-games"`
	// pool batch size for on-chain submissions
	PoolBatchSize int `mapstructure:"pool-batch-size"`
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
