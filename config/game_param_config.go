package config

type GameParamConfig struct {
	MaxHP             int     `mapstructure:"max-hp"`
	InitialMultiplier int     `mapstructure:"initial-multiplier"`
	SystemFeeRate     float64 `mapstructure:"system-fee-rate"`     // 系统抽水比例，例如 0.016 表示 1.6%
	WinnerPointRate   float64 `mapstructure:"winner-point-rate"`   // 获胜者积分倍率
	LoserPointRate    float64 `mapstructure:"loser-point-rate"`    // 失败者积分倍率
	TieTokenRate      float64 `mapstructure:"tie-token-rate"`      // 平局扣除赌注比例
	TiePointRate      float64 `mapstructure:"tie-point-rate"`      // 平局积分倍率
	TokenThreshold    int     `mapstructure:"token-threshold"`     // 加入匹配所需最低可用代币
	BaseStake         int     `mapstructure:"base-stake"`          // 计算奖励时使用的基准赌注
	DailyRewardTokens int     `mapstructure:"daily-reward-tokens"` // 每日奖励代币数量
}

// 全局可读的游戏参数
var GameParams = GameParamConfig{}

// InitializeGameParams 设置 GameParams 的默认值并赋值给全局变量
func InitializeGameParams(gameParams *GameParamConfig) {
	// 设置默认值
	if gameParams.MaxHP == 0 {
		gameParams.MaxHP = 3000
	}
	if gameParams.InitialMultiplier == 0 {
		gameParams.InitialMultiplier = 1
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

	// 赋值给全局变量
	GameParams = *gameParams
}
