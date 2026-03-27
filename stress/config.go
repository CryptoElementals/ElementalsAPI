package stress

// Config holds the configuration for stress testing
type Config struct {
	// Server configuration
	BaseURL string `json:"base_url"` // HTTP API base URL

	// Bot configuration
	NumBots    int    `json:"num_bots"`     // Number of bots to run
	BotInfoCSV string `json:"bot_info_csv"` // CSV file path to save bot info
}
