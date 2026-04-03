package config

import (
	"github.com/CryptoElementals/common/db"
	"github.com/CryptoElementals/common/log"
	"github.com/CryptoElementals/common/redis"
	"github.com/spf13/viper"
)

var RSGConf = RoomServerConfig{}

// RoomServerConfig represents the complete application configuration structure
type RoomServerConfig struct {
	LogCfg              log.Config   `mapstructure:"log"`
	RedisCfg            redis.Config `mapstructure:"redis"`
	DbCfg               db.Config    `mapstructure:"database"`
	ChainCfg            ChainConfig  `mapstructure:"chain"`
	WalletPaths         []string     `mapstructure:"wallet-paths"`
	ListenPort          int64        `mapstructure:"listen-port"`
	BotWaitTime         int64        `mapstructure:"bot-wait-time"`
	StatServiceEndpoint string       `mapstructure:"stat-service-endpoint"`
	ShouldRecoverGames  bool         `mapstructure:"should-recover-games"`
	// pool batch size for on-chain submissions
	PoolBatchSize int `mapstructure:"pool-batch-size"`

	// MinTokenToJoinQueue is the minimum token balance required to join matchmaking (API server uses its own game-params for HTTP checks).
	MinTokenToJoinQueue int32 `mapstructure:"min-token-to-join-queue"`
	// GameArgsID is the game_args row id used for new matches (must be non-zero; room server loads it at startup).
	GameArgsID uint `mapstructure:"game-args-id"`
}

func InitRSConfig(configPath string) error {
	viper.SetConfigFile(configPath)
	err := viper.ReadInConfig()
	if err != nil {
		return err
	}
	if err := viper.Unmarshal(&RSGConf); err != nil {
		return err
	}

	if RSGConf.MinTokenToJoinQueue == 0 {
		RSGConf.MinTokenToJoinQueue = 10000
	}

	// GameParams on room is for tests / non-DB defaults; match base stake lives on game_args.base_stake (copied per match) for lobby settlement.
	InitializeGameParams(&GameParamConfig{})
	if GameParams.BaseStake == 0 {
		GameParams.BaseStake = 1000
	}

	return nil
}
