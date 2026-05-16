package config

// GameParamConfig holds API-server-only settings (rewards, keygen).
// Match rules and timeouts live in models.GameArgs (DB), not here.
type GameParamConfig struct {
	KeygenPolicy uint `mapstructure:"keygen-policy"` // 1=backend temp keys (debug), 2=client-generated (production)

	DailyRewardStartDate        string `mapstructure:"daily-reward-start-date"`
	DailyRewardEndDate          string `mapstructure:"daily-reward-end-date"`
	FirstTimeRewardTokens       int    `mapstructure:"first-time-reward-tokens"`
	DailyRewardTokensAfterFirst int    `mapstructure:"daily-reward-tokens-after-first"`
	EnableDailyReward           bool   `mapstructure:"enable-daily-reward"`

	NewUserRewardTokens int  `mapstructure:"new-user-reward-tokens"`
	EnableNewUserReward bool `mapstructure:"enable-new-user-reward"`
}

// GameParams is the global copy loaded from apiserver config.
var GameParams = GameParamConfig{}

// InitializeGameParams applies defaults and copies into GameParams.
func InitializeGameParams(gameParams *GameParamConfig) {
	if gameParams.FirstTimeRewardTokens == 0 {
		gameParams.FirstTimeRewardTokens = 10000
	}
	if gameParams.DailyRewardTokensAfterFirst == 0 {
		gameParams.DailyRewardTokensAfterFirst = 3000
	}
	if gameParams.NewUserRewardTokens == 0 {
		gameParams.NewUserRewardTokens = 5000
	}
	if gameParams.KeygenPolicy == 0 {
		gameParams.KeygenPolicy = 2
	}
	GameParams = *gameParams
}
