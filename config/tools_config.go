package config

import (
	"github.com/CryptoElementals/common/db"
	"github.com/spf13/viper"
)

type ToolsConfig struct {
	DbCfg db.Config `mapstructure:"database"`
}

var ToolsGConf = ToolsConfig{}

func InitToolsConfig(configPath string) error {
	viper.SetConfigFile(configPath)
	err := viper.ReadInConfig()
	if err != nil {
		return err
	}
	if err := viper.Unmarshal(&ToolsGConf); err != nil {
		return err
	}
	return nil
}
