package config

import (
	"github.com/CryptoElementals/common/log"
	"github.com/spf13/viper"
)

type WalletPath struct {
	AccountWallet   string `mapstructure:"account-wallet"`
	TemporaryWallet string `mapstructure:"temporary-wallet"`
}

type BotConfig struct {
	LogCfg             log.Config   `mapstructure:"log"`
	ChainCfg           ChainConfig  `mapstructure:"chain"`
	RoomServerEndpoint string       `mapstructure:"room-server-endpoint"`
	WalletPaths        []WalletPath `mapstructure:"wallet-paths"`
}

var BotCfg BotConfig

func InitBotConfig(configPath string) error {
	viper.SetConfigFile(configPath)
	err := viper.ReadInConfig()
	if err != nil {
		return err
	}
	if err := viper.Unmarshal(&BotCfg); err != nil {
		return err
	}

	return nil
}
