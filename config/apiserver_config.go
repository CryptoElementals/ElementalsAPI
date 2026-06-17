package config

import (
	"fmt"

	"github.com/CryptoElementals/common/db"
	"github.com/CryptoElementals/common/log"
	"github.com/CryptoElementals/common/redis"
)

var RoomServerAddress string
var GConf ApiServerConfig

// EnvironmentConfig is one logical game shard: its own named Redis pool (event streams),
// room gRPC, lobby gRPC, and ledger gRPC.
type EnvironmentConfig struct {
	Name                string       `mapstructure:"name"`
	RedisCfg            redis.Config `mapstructure:"redis"`
	RoomServerAddress   string       `mapstructure:"room-server-address"`
	LobbyServerAddress  string       `mapstructure:"lobby-server-address"`
	LedgerServerAddress string       `mapstructure:"ledger-server-address"`

	DailyRewardStartDate        string `mapstructure:"daily-reward-start-date"`
	DailyRewardEndDate          string `mapstructure:"daily-reward-end-date"`
	FirstTimeRewardTokens       int    `mapstructure:"first-time-reward-tokens"`
	DailyRewardTokensAfterFirst int    `mapstructure:"daily-reward-tokens-after-first"`
	EnableDailyReward           bool   `mapstructure:"enable-daily-reward"`

	NewUserRewardTokens int  `mapstructure:"new-user-reward-tokens"`
	EnableNewUserReward bool `mapstructure:"enable-new-user-reward"`
}

// ApiServerConfig represents the complete application configuration structure
type ApiServerConfig struct {
	LogCfg             log.Config          `mapstructure:"log"`
	DbCfg              db.Config           `mapstructure:"database"`
	RedisCfg           redis.Config        `mapstructure:"redis"` // API server default Redis (sessions, refresh-token cache)
	Snowflake          SnowflakeConfig     `mapstructure:"snowflake"`
	ServerCfg          ServerConfig        `mapstructure:"server"`
	S3Config           S3Config            `mapstructure:"s3"`
	EnvironmentConfigs []EnvironmentConfig `mapstructure:"environments"`
}

// EnvironmentForServerType returns the environment for trial/normal server type.
func (cfg *ApiServerConfig) EnvironmentForServerType(serverType string) (EnvironmentConfig, bool) {
	return cfg.EnvironmentByName(NormalizeServerType(serverType))
}

// EnvironmentByName returns the environment with the given name and whether it exists.
func (cfg *ApiServerConfig) EnvironmentByName(name string) (EnvironmentConfig, bool) {
	if name == "" || cfg == nil {
		return EnvironmentConfig{}, false
	}
	for i := range cfg.EnvironmentConfigs {
		if cfg.EnvironmentConfigs[i].Name == name {
			return cfg.EnvironmentConfigs[i], true
		}
	}
	return EnvironmentConfig{}, false
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

	// 将主环境房间服地址写入全局变量
	if len(cfg.EnvironmentConfigs) > 0 {
		RoomServerAddress = cfg.EnvironmentConfigs[0].RoomServerAddress
	}

	// 设置全局配置
	GConf = *cfg

	return cfg, nil
}

// ValidateApiServerConfig validates the application configuration
func ValidateApiServerConfig(cfg *ApiServerConfig) error {
	// Validate log configuration
	if err := validateLogConfig(&cfg.LogCfg); err != nil {
		return fmt.Errorf("log config validation failed: %w", err)
	}

	if len(cfg.EnvironmentConfigs) < 1 {
		return fmt.Errorf("at least one environment is required in environments")
	}

	seen := make(map[string]struct{}, len(cfg.EnvironmentConfigs))
	requiredEnvs := make(map[string]struct{}, len(RequiredEnvironmentNames()))
	for _, name := range RequiredEnvironmentNames() {
		requiredEnvs[name] = struct{}{}
	}
	for i, env := range cfg.EnvironmentConfigs {
		if env.Name == "" {
			return fmt.Errorf("environment[%d]: name cannot be empty", i)
		}
		if env.Name == "default" {
			return fmt.Errorf("environment[%d]: name %q is reserved", i, env.Name)
		}
		if _, dup := seen[env.Name]; dup {
			return fmt.Errorf("duplicate environment name: %q", env.Name)
		}
		seen[env.Name] = struct{}{}
		delete(requiredEnvs, env.Name)

		if err := validateRedisConfig(&env.RedisCfg); err != nil {
			return fmt.Errorf("environment %q redis: %w", env.Name, err)
		}
		if env.RoomServerAddress == "" {
			return fmt.Errorf("environment %q: room server address cannot be empty", env.Name)
		}
		if env.LobbyServerAddress == "" {
			return fmt.Errorf("environment %q: lobby server address cannot be empty", env.Name)
		}
	}
	for name := range requiredEnvs {
		return fmt.Errorf("missing required environment %q (trial and normal backends must be configured)", name)
	}

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

	// Validate S3 configuration
	if err := validateS3Config(&cfg.S3Config); err != nil {
		return fmt.Errorf("s3 config validation failed: %w", err)
	}

	return nil
}

// validateServerConfig validates server configuration
func validateServerConfig(cfg *ServerConfig) error {
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

func applyEnvironmentRewardDefaults(env *EnvironmentConfig) {
	if env.FirstTimeRewardTokens == 0 {
		env.FirstTimeRewardTokens = 10000
	}
	if env.DailyRewardTokensAfterFirst == 0 {
		env.DailyRewardTokensAfterFirst = 3000
	}
	if env.NewUserRewardTokens == 0 {
		env.NewUserRewardTokens = 5000
	}
}

func applyRedisDefaults(rc *redis.Config) {
	if rc.Address == "" {
		rc.Address = "localhost:6379"
	}
	if rc.Size == 0 {
		rc.Size = 10
	}
	if rc.SessionExpire == 0 {
		rc.SessionExpire = 43200 // 12小时
	}
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

	applyRedisDefaults(&cfg.RedisCfg)
	for i := range cfg.EnvironmentConfigs {
		applyRedisDefaults(&cfg.EnvironmentConfigs[i].RedisCfg)
		if cfg.EnvironmentConfigs[i].RoomServerAddress == "" {
			cfg.EnvironmentConfigs[i].RoomServerAddress = "127.0.0.1:50051"
		}
		if cfg.EnvironmentConfigs[i].LobbyServerAddress == "" {
			cfg.EnvironmentConfigs[i].LobbyServerAddress = "127.0.0.1:50052"
		}
		applyEnvironmentRewardDefaults(&cfg.EnvironmentConfigs[i])
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

	// Set default S3 configuration
	if cfg.S3Config.PresignExpire == 0 {
		cfg.S3Config.PresignExpire = 3600 // 1小时过期
	}

}
