package config

import (
	"github.com/CryptoElementals/common/db"
	"github.com/CryptoElementals/common/log"
	"github.com/spf13/viper"
)

var ScannerGConf = ScannerConfig{}

// ScannerConfig represents the complete application configuration structure
type ScannerConfig struct {
	LogCfg            log.Config  `mapstructure:"log"`
	DbCfg             db.Config   `mapstructure:"database"`
	ChainCfg          ChainConfig `mapstructure:"chain"`
	RoomServerHttpRpc string      `mapstructure:"room-server"`
	RoomServerMocked  bool        `mapstructure:"room-server-mocked"`
}

func InitScannerConfig(configPath string) error {
	viper.SetConfigFile(configPath)
	err := viper.ReadInConfig()
	if err != nil {
		return err
	}
	return viper.Unmarshal(&ScannerGConf)
}
