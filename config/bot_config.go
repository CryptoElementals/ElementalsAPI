package config

import (
	"fmt"
	"strings"

	"github.com/CryptoElementals/common/db"
	"github.com/CryptoElementals/common/log"
	"github.com/CryptoElementals/common/redis"
	"github.com/spf13/viper"
)

type WalletInfo struct {
	PlayerId        int64  `mapstructure:"player-id"`
	TemporaryWallet string `mapstructure:"temporary-wallet"`
}

type BotConfig struct {
	LogCfg                          log.Config   `mapstructure:"log"`
	ChainCfg                        ChainConfig  `mapstructure:"chain"`
	DbCfg                           db.Config    `mapstructure:"database"`
	RedisCfg                        redis.Config `mapstructure:"redis"`
	RoomServerEndpoint              string       `mapstructure:"room-server-endpoint"`
	LobbyServerEndpoint             string       `mapstructure:"lobby-server-endpoint"`
	ApiServerEndpoint               string       `mapstructure:"api-server-endpoint"`
	NumBots                         int          `mapstructure:"num-bots"`
	GameClientMode                  string       `mapstructure:"game-client-mode"` // grpc | http
	BotRegistryHeartbeatIntervalSec int          `mapstructure:"bot-registry-heartbeat-interval-sec"`
	WalletInfos                     []WalletInfo `mapstructure:"wallet-infos"` // legacy fallback
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
	if BotCfg.LobbyServerEndpoint == "" {
		BotCfg.LobbyServerEndpoint = "127.0.0.1:50052"
	}
	if BotCfg.GameClientMode == "" {
		BotCfg.GameClientMode = "grpc"
	}
	BotCfg.GameClientMode = strings.ToLower(strings.TrimSpace(BotCfg.GameClientMode))
	if BotCfg.NumBots <= 0 && len(BotCfg.WalletInfos) == 0 {
		return fmt.Errorf("num-bots must be > 0 (or provide legacy wallet-infos)")
	}
	if BotCfg.ApiServerEndpoint == "" {
		return fmt.Errorf("api-server-endpoint is required (used for bot account provisioning)")
	}
	if BotCfg.GameClientMode != "grpc" && BotCfg.GameClientMode != "http" {
		return fmt.Errorf("game-client-mode must be either grpc or http")
	}
	if BotCfg.RedisCfg.Address == "" {
		return fmt.Errorf("redis.address is required for redis-backed bot registry")
	}
	if BotCfg.BotRegistryHeartbeatIntervalSec <= 0 {
		BotCfg.BotRegistryHeartbeatIntervalSec = 8
	}

	return nil
}
