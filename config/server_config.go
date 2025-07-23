package config

// ServerConfig defines the configuration for the HTTP server
type ServerConfig struct {
	Port               int    `mapstructure:"port"`
	ServerMode         string `mapstructure:"server-mode"`
	SessionMaxAge      int    `mapstructure:"session-max-age"`
	RefreshTokenMaxAge int    `mapstructure:"refresh-token-max-age"`
	ServiceName        string `mapstructure:"service-name"`
}
