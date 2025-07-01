package config

import "github.com/spf13/viper"

func InitConfig(configPath string, cfg any) error {
	viper.SetConfigFile(configPath)
	err := viper.ReadInConfig()
	if err != nil {
		return err
	}
	return viper.Unmarshal(cfg)
}
