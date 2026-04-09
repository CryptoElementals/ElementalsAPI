package config

import (
	"github.com/CryptoElementals/common/db"
	"github.com/CryptoElementals/common/redis"
	"github.com/spf13/viper"
)

type ToolsConfig struct {
	DbCfg    db.Config    `mapstructure:"database"`
	RedisCfg redis.Config `mapstructure:"redis"`
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
