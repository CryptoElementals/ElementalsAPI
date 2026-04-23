package config

// SnowflakeConfig configures the process-wide snowflake worker id (see [github.com/bwmarrin/snowflake]).
type SnowflakeConfig struct {
	// NodeID is the worker id (1–1023). Zero (default / omitted) means a random id is chosen at startup.
	NodeID int64 `mapstructure:"node-id"`
}
