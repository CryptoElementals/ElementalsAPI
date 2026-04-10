package config

import (
	"github.com/CryptoElementals/common/db"
	"github.com/CryptoElementals/common/redis"
	"github.com/spf13/viper"
)

type ToolsConfig struct {
	DbCfg       db.Config              `mapstructure:"database"`
	RedisCfg    redis.Config           `mapstructure:"redis"`
	Game        ToolsGameConfig        `mapstructure:"game"`
	BotRegistry ToolsBotRegistryConfig `mapstructure:"bot-registry"`
	Queue       ToolsQueueConfig       `mapstructure:"queue"`
}

type ToolsGameConfig struct {
	ClientMode          string             `mapstructure:"client-mode"`
	RoomServerEndpoint  string             `mapstructure:"room-server-endpoint"`
	LobbyServerEndpoint string             `mapstructure:"lobby-server-endpoint"`
	ApiServerEndpoint   string             `mapstructure:"api-server-endpoint"`
	PlayerID            int64              `mapstructure:"player-id"`
	TempWalletPath      string             `mapstructure:"temp-wallet-path"`
	Get                 ToolsGameGetConfig `mapstructure:"get"`
}

type ToolsGameGetConfig struct {
	GameID int64 `mapstructure:"game-id"`
}

type ToolsBotRegistryConfig struct {
	Namespace    string `mapstructure:"namespace"`
	FreshnessSec int64  `mapstructure:"freshness-sec"`
}

type ToolsQueueConfig struct {
	Namespace string `mapstructure:"namespace"`
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
