package config

import (
	"github.com/CryptoElementals/common/log"
	"github.com/spf13/viper"
)

var StressCfg StressConfig

// StressConfig represents the stress test configuration structure
type StressConfig struct {
	LogCfg     log.Config `mapstructure:"log"`
	BaseURL    string     `mapstructure:"base-url"`
	NumBots    int        `mapstructure:"num-bots"`
	BotInfoCSV string     `mapstructure:"bot-info-csv"`
}

func InitStressConfig(configPath string) error {
	viper.SetConfigFile(configPath)
	err := viper.ReadInConfig()
	if err != nil {
		return err
	}
	if err := viper.Unmarshal(&StressCfg); err != nil {
		return err
	}

	return nil
}
