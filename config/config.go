package config

import (
	"fmt"

	"github.com/CryptoElementals/common/db"
	"github.com/CryptoElementals/common/log"
	"github.com/CryptoElementals/common/redis"
	"github.com/CryptoElementals/common/server"
	"github.com/spf13/viper"
)

// InitConfig loads configuration from file and unmarshals into the provided struct
func InitConfig(configPath string, cfg any) error {
	viper.SetConfigFile(configPath)
	err := viper.ReadInConfig()
	if err != nil {
		return err
	}
	return viper.Unmarshal(cfg)
}

// AppConfig represents the complete application configuration structure
type AppConfig struct {
	LogCfg    log.Config    `mapstructure:"log"`
	RedisCfg  redis.Config  `mapstructure:"redis"`
	DbCfg     db.Config     `mapstructure:"database"`
	ServerCfg server.Config `mapstructure:"server"`
}

// LoadAppConfig loads the complete application configuration from file
func LoadAppConfig(configPath string) (*AppConfig, error) {
	cfg := &AppConfig{}
	err := InitConfig(configPath, cfg)
	if err != nil {
		return nil, err
	}

	// Set default values
	setDefaultValues(cfg)

	return cfg, nil
}

// ValidateAppConfig validates the application configuration
func ValidateAppConfig(cfg *AppConfig) error {
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
func setDefaultValues(cfg *AppConfig) {
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

// validateServerConfig validates server configuration
func validateServerConfig(cfg *server.Config) error {
	if cfg.Port <= 0 || cfg.Port > 65535 {
		return fmt.Errorf("invalid server port: %d", cfg.Port)
	}

	validModes := []string{"debug", "release", "test"}
	isValidMode := false
	for _, mode := range validModes {
		if cfg.ServerMode == mode {
			isValidMode = true
			break
		}
	}
	if !isValidMode {
		return fmt.Errorf("invalid server mode: %s", cfg.ServerMode)
	}

	if cfg.SessionMaxAge <= 0 {
		return fmt.Errorf("session max age must be greater than 0")
	}

	if cfg.RefreshTokenMaxAge <= 0 {
		return fmt.Errorf("refresh token max age must be greater than 0")
	}

	return nil
}
