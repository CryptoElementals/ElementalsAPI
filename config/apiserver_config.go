package config

import (
	"fmt"

	"github.com/CryptoElementals/common/db"
	"github.com/CryptoElementals/common/log"
	"github.com/CryptoElementals/common/redis"
	"github.com/CryptoElementals/common/server"
)

// ApiServerConfig represents the complete application configuration structure
type ApiServerConfig struct {
	LogCfg    log.Config    `mapstructure:"log"`
	RedisCfg  redis.Config  `mapstructure:"redis"`
	DbCfg     db.Config     `mapstructure:"database"`
	ServerCfg server.Config `mapstructure:"server"`
}

// LoadApiServerConfig loads the complete application configuration from file
func LoadApiServerConfig(configPath string) (*ApiServerConfig, error) {
	cfg := &ApiServerConfig{}
	err := InitConfig(configPath, cfg)
	if err != nil {
		return nil, err
	}

	// Set default values
	setDefaultValues(cfg)

	return cfg, nil
}

// ValidateApiServerConfig validates the application configuration
func ValidateApiServerConfig(cfg *ApiServerConfig) error {
	// Validate log configuration
	if err := validateLogConfig(&cfg.LogCfg); err != nil {
		return fmt.Errorf("log config validation failed: %w", err)
	}

	// Validate Redis configuration
	if err := validateRedisConfig(&cfg.RedisCfg); err != nil {
		return fmt.Errorf("redis config validation failed: %w", err)
	}

	// Validate database configuration
	if err := validateDatabaseConfig(&cfg.DbCfg); err != nil {
		return fmt.Errorf("database config validation failed: %w", err)
	}

	// Validate server configuration
	if err := validateServerConfig(&cfg.ServerCfg); err != nil {
		return fmt.Errorf("server config validation failed: %w", err)
	}

	return nil
}

// setDefaultValues sets default values for configuration fields
func setDefaultValues(cfg *ApiServerConfig) {
	// Set default log configuration
	if cfg.LogCfg.Level == "" {
		cfg.LogCfg.Level = "debug"
	}
	if cfg.LogCfg.Dir == "" {
		cfg.LogCfg.Dir = "./logs"
	}
	if cfg.LogCfg.Prefix == "" {
		cfg.LogCfg.Prefix = "beast-royale"
	}
	if cfg.LogCfg.Suffix == "" {
		cfg.LogCfg.Suffix = "log"
	}
	if cfg.LogCfg.MaxAge == 0 {
		cfg.LogCfg.MaxAge = 7
	}
	if cfg.LogCfg.RotationTime == 0 {
		cfg.LogCfg.RotationTime = 24
	}

	// Set default Redis configuration
	if cfg.RedisCfg.Address == "" {
		cfg.RedisCfg.Address = "localhost:6379"
	}
	if cfg.RedisCfg.Size == 0 {
		cfg.RedisCfg.Size = 10
	}
	// Set default Redis session expire time
	if cfg.RedisCfg.SessionExpire == 0 {
		cfg.RedisCfg.SessionExpire = 43200 // 12小时
	}

	// Set default database configuration
	if cfg.DbCfg.Endpoint == "" {
		cfg.DbCfg.Endpoint = "localhost:3306"
	}
	if cfg.DbCfg.User == "" {
		cfg.DbCfg.User = "root"
	}
	if cfg.DbCfg.DbName == "" {
		cfg.DbCfg.DbName = "beast_royale"
	}

	// Set default server configuration
	if cfg.ServerCfg.Port == 0 {
		cfg.ServerCfg.Port = 8080
	}
	if cfg.ServerCfg.ServerMode == "" {
		cfg.ServerCfg.ServerMode = "debug"
	}
	if cfg.ServerCfg.SessionMaxAge == 0 {
		cfg.ServerCfg.SessionMaxAge = 180
	}
	if cfg.ServerCfg.RefreshTokenMaxAge == 0 {
		cfg.ServerCfg.RefreshTokenMaxAge = 300
	}
	if cfg.ServerCfg.ServiceName == "" {
		cfg.ServerCfg.ServiceName = "DILL"
	}
}
