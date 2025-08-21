package config

import (
	"github.com/CryptoElementals/common/db"
	"github.com/CryptoElementals/common/log"
	"github.com/spf13/viper"
)

var StatGConf = StatConfig{}

// StatConfig represents the complete application configuration structure
type StatConfig struct {
	LogCfg     log.Config `mapstructure:"log"`
	DbCfg      db.Config  `mapstructure:"database"`
	ListenPort uint32     `mapstructure:"listen-port"`
}

func InitStatConfig(configPath string) error {
	viper.SetConfigFile(configPath)
	err := viper.ReadInConfig()
	if err != nil {
		return err
	}
	return viper.Unmarshal(&StatGConf)
}
