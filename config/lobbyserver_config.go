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
	LogCfg                  log.Config       `mapstructure:"log"`
	RedisCfg                redis.Config     `mapstructure:"redis"`
	DbCfg                   db.Config        `mapstructure:"database"`
	Snowflake               SnowflakeConfig  `mapstructure:"snowflake"`
	TournamentCfg           TournamentConfig `mapstructure:"tournament"`
	ListenPort              int64            `mapstructure:"listen-port"`
	RoomServerAddress       string           `mapstructure:"room-server-address"`
	MinTokenToJoinQueue     int32            `mapstructure:"min-token-to-join-queue"`
	GameArgsID              uint             `mapstructure:"game-args-id"`
	BotWaitTime             int64            `mapstructure:"bot-wait-time"`
	BotRegistryFreshnessSec int64            `mapstructure:"bot-registry-freshness-sec"`
	StatServiceEndpoint     string           `mapstructure:"stat-service-endpoint"`
	IsDevelop               bool             `mapstructure:"is-develop"`
}

type TournamentConfig struct {
	EntryFee                int32   `mapstructure:"entry-fee"`
	MinPlayersRequired      uint32  `mapstructure:"min-players-required"`
	IntervalSeconds         uint32  `mapstructure:"interval-seconds"`
	BeforeStartSeconds      uint32  `mapstructure:"before-start-seconds"`
	BotFillWindowSeconds    uint32  `mapstructure:"bot-fill-window-seconds"`
	BotFillIntervalSeconds  uint32  `mapstructure:"bot-fill-interval-seconds"`
	// TopFourPrizeTokens is rank 1..4 fixed prize amounts (tokens) shown to clients; pool is entry_fee * participants.
	TopFourPrizeTokens       []int32 `mapstructure:"top-four-prize-tokens"`
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
	if LSGConf.BotRegistryFreshnessSec <= 0 {
		LSGConf.BotRegistryFreshnessSec = 10
	}
	if LSGConf.TournamentCfg.BotFillWindowSeconds == 0 {
		LSGConf.TournamentCfg.BotFillWindowSeconds = 180
	}
	if LSGConf.TournamentCfg.BotFillIntervalSeconds == 0 {
		LSGConf.TournamentCfg.BotFillIntervalSeconds = 15
	}
	return nil
}
