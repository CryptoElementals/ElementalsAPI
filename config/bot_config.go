package config

import (
	"github.com/CryptoElementals/common/log"
	"github.com/spf13/viper"
)

type BotConfig struct {
	LogCfg             log.Config  `mapstructure:"log"`
	ChainCfg           ChainConfig `mapstructure:"chain"`
	RoomServerEndpoint string
	WalletPaths        []string
	ListenPort         int
}

var BotCfg BotConfig

func InitBotConfig(configPath string) error {
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
