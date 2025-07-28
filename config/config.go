package config

import (
	"fmt"

	"github.com/CryptoElementals/common/db"
	"github.com/CryptoElementals/common/log"
	"github.com/CryptoElementals/common/redis"
	"github.com/spf13/viper"
)

type NodeConfig struct {
	HttpRpc string `mapstructure:"http-rpc"` //https://mainnet.optimism.io
	WsRpc   string `mapstructure:"ws-rpc"`
}

type ContractConfig struct {
	RoomManagerAddress string `mapstructure:"room-manager-address"`
	PlayerStateAddress string `mapstructure:"player-state-address"`
}

type ChainConfig struct {
	NodeConfig     `mapstructure:"node"`
	ContractConfig `mapstructure:"contract"`
}

// InitConfig loads configuration from file and unmarshals into the provided struct
func InitConfig(configPath string, cfg any) error {
	viper.SetConfigFile(configPath)
	viper.SetConfigType("yaml") // 明确设置配置文件类型
	err := viper.ReadInConfig()
	if err != nil {
		return err
	}
	return viper.Unmarshal(cfg)
}

// validateLogConfig validates log configuration
func validateLogConfig(cfg *log.Config) error {
	if cfg.Level != "" {
		validLevels := []string{"debug", "info", "warn", "error", "fatal", "panic"}
		isValid := false
		for _, level := range validLevels {
			if cfg.Level == level {
				isValid = true
				break
			}
		}
		if !isValid {
			return fmt.Errorf("invalid log level: %s", cfg.Level)
		}
	}
	return nil
}

// validateRedisConfig validates Redis configuration
func validateRedisConfig(cfg *redis.Config) error {
	if cfg.Address == "" {
		return fmt.Errorf("redis address is required")
	}
	if cfg.Size <= 0 {
		return fmt.Errorf("redis pool size must be greater than 0")
	}
	return nil
}

// validateDatabaseConfig validates database configuration
func validateDatabaseConfig(cfg *db.Config) error {
	if cfg.Endpoint == "" {
		return fmt.Errorf("database endpoint is required")
	}
	if cfg.User == "" {
		return fmt.Errorf("database user is required")
	}
	if cfg.DbName == "" {
		return fmt.Errorf("database name is required")
	}
	return nil
}

// ServerConfig defines the configuration for the HTTP server
type ServerConfig struct {
	Port               int    `mapstructure:"port"`
	ServerMode         string `mapstructure:"server-mode"`
	SessionMaxAge      int    `mapstructure:"session-max-age"`
	RefreshTokenMaxAge int    `mapstructure:"refresh-token-max-age"`
	ServiceName        string `mapstructure:"service-name"`
}
