package config

import (
	"github.com/CryptoElementals/common/db"
	"github.com/CryptoElementals/common/log"
	"github.com/CryptoElementals/common/redis"
	"github.com/spf13/viper"
)

// LSGConf is the global lobby server config after InitLSConfig.
var LSGConf = LobbyServerConfig{}

// LobbyServerConfig is loaded by ele-lobbyserver.
type LobbyServerConfig struct {
	LogCfg              log.Config    `mapstructure:"log"`
	RedisCfg            redis.Config  `mapstructure:"redis"`
	DbCfg               db.Config     `mapstructure:"database"`
	ListenPort          int64         `mapstructure:"listen-port"`
	RoomServerAddress   string        `mapstructure:"room-server-address"`
	MinTokenToJoinQueue int32         `mapstructure:"min-token-to-join-queue"`
	GameArgsID          uint          `mapstructure:"game-args-id"`
	BotWaitTime         int64         `mapstructure:"bot-wait-time"`
	StatServiceEndpoint string        `mapstructure:"stat-service-endpoint"`
	IsDevelop           bool          `mapstructure:"is-develop"`
}

// InitLSConfig loads lobby server config from a YAML file (viper).
func InitLSConfig(configPath string) error {
	viper.SetConfigFile(configPath)
	if err := viper.ReadInConfig(); err != nil {
		return err
	}
	if err := viper.Unmarshal(&LSGConf); err != nil {
		return err
	}
	if LSGConf.MinTokenToJoinQueue == 0 {
		LSGConf.MinTokenToJoinQueue = 10000
	}
	return nil
}
