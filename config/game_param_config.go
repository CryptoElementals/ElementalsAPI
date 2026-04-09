package config

type GameParamConfig struct {
	SystemFeeRate     float64 `mapstructure:"system-fee-rate"`     // 系统抽水比例，例如 0.016 表示 1.6%
	WinnerPointRate   float64 `mapstructure:"winner-point-rate"`   // 获胜者积分倍率
	LoserPointRate    float64 `mapstructure:"loser-point-rate"`    // 失败者积分倍率
	TieTokenRate      float64 `mapstructure:"tie-token-rate"`      // 平局扣除赌注比例
	TiePointRate      float64 `mapstructure:"tie-point-rate"`      // 平局积分倍率
	TokenThreshold    int     `mapstructure:"token-threshold"`     // 加入匹配所需最低可用代币
	BaseStake         int     `mapstructure:"base-stake"`          // 计算奖励时使用的基准赌注
	DailyRewardTokens int     `mapstructure:"daily-reward-tokens"` // 每日奖励代币数量（已废弃，使用FirstTimeRewardTokens和DailyRewardTokensAfterFirst）
	KeygenPolicy      uint    `mapstructure:"keygen-policy"`       // 1-后端生成(调试)，2-前端生成(生产)
	// 每日奖励活动配置
	DailyRewardStartDate        string `mapstructure:"daily-reward-start-date"`         // 活动开始日期，格式：YYYY-MM-DD
	DailyRewardEndDate          string `mapstructure:"daily-reward-end-date"`           // 活动结束日期，格式：YYYY-MM-DD
	FirstTimeRewardTokens       int    `mapstructure:"first-time-reward-tokens"`        // 活动期间内第一次领取奖励代币数量
	DailyRewardTokensAfterFirst int    `mapstructure:"daily-reward-tokens-after-first"` // 活动后续每天奖励代币数量

	MaxRounds    int64 `mapstructure:"max-rounds"`
	InitialHP    int64 `mapstructure:"initial-hp"`
	MaxHPOneLine int64 `mapstructure:"max-hp-one-line"`
	// timeouts
	ConfirmationTimeout         int64 `mapstructure:"confirmation-timeout"`          // Timeout for game match and round confirmation
	CommitmentSubmissionTimeout int64 `mapstructure:"commitment-submission-timeout"` // Timeout for commitment submission
	CardSubmissionTimeout       int64 `mapstructure:"card-submission-timeout"`       // Timeout for card submission
	GameContinueTimeout         int64 `mapstructure:"game-continue-timeout"`         // Timeout for game continue
	// timeout redundancy
	ConfirmationTimeoutRedundancy         int64 `mapstructure:"confirmation-timeout-redundancy"`          // Redundancy for game match and round confirmation
	CommitmentSubmissionTimeoutRedundancy int64 `mapstructure:"commitment-submission-timeout-redundancy"` // Redundancy for commitment submission
	CardSubmissionTimeoutRedundancy       int64 `mapstructure:"card-submission-timeout-redundancy"`       // Redundancy for card submission
	GameContinueTimeoutRedundancy         int64 `mapstructure:"game-continue-timeout-redundancy"`         // Redundancy for game continue

	MaxTurnsPerRound int64 `mapstructure:"max-turns-per-round"`
}

// 全局可读的游戏参数
var GameParams = GameParamConfig{}

// InitializeGameParams 设置 GameParams 的默认值并赋值给全局变量
func InitializeGameParams(gameParams *GameParamConfig) {
	// 设置默认值
	if gameParams.InitialHP == 0 {
		gameParams.InitialHP = 6000
	}
	if gameParams.SystemFeeRate == 0 {
		gameParams.SystemFeeRate = 0.016
	}
	if gameParams.WinnerPointRate == 0 {
		gameParams.WinnerPointRate = 0.012
	}
	if gameParams.LoserPointRate == 0 {
		gameParams.LoserPointRate = 0.004
	}
	if gameParams.TieTokenRate == 0 {
		gameParams.TieTokenRate = 0.008
	}
	if gameParams.TiePointRate == 0 {
		gameParams.TiePointRate = 0.008
	}
	if gameParams.TokenThreshold == -1 {
		gameParams.TokenThreshold = 10000
	}
	if gameParams.BaseStake == -1 {
		gameParams.BaseStake = 1000
	}
	if gameParams.DailyRewardTokens == 0 {
		gameParams.DailyRewardTokens = 1000
	}
	if gameParams.FirstTimeRewardTokens == 0 {
		gameParams.FirstTimeRewardTokens = 10000
	}
	if gameParams.DailyRewardTokensAfterFirst == 0 {
		gameParams.DailyRewardTokensAfterFirst = 3000
	}
	if gameParams.KeygenPolicy == 0 {
		gameParams.KeygenPolicy = 2
	}

	if gameParams.MaxRounds == 0 {
		gameParams.MaxRounds = 3
	}
	if gameParams.InitialHP == 0 {
		gameParams.InitialHP = 3000
	}
	if gameParams.ConfirmationTimeout == 0 {
		gameParams.ConfirmationTimeout = 10
	}
	if gameParams.CommitmentSubmissionTimeout == 0 {
		gameParams.CommitmentSubmissionTimeout = 20
	}
	if gameParams.CardSubmissionTimeout == 0 {
		gameParams.CardSubmissionTimeout = 20
	}
	if gameParams.GameContinueTimeout == 0 {
		gameParams.GameContinueTimeout = 10
	}
	if gameParams.MaxTurnsPerRound == 0 {
		gameParams.MaxTurnsPerRound = 3
	}

	// 赋值给全局变量
	GameParams = *gameParams
}
