package config

type GameParamConfig struct {
	MaxHP             int     `mapstructure:"max-hp"`
	InitialMultiplier int     `mapstructure:"initial-multiplier"`
	SystemFeeRate     float64 `mapstructure:"system-fee-rate"`   // 系统抽水比例，例如 0.016 表示 1.6%
	WinnerPointRate   float64 `mapstructure:"winner-point-rate"` // 获胜者积分倍率
	LoserPointRate    float64 `mapstructure:"loser-point-rate"`  // 失败者积分倍率
	TieTokenRate      float64 `mapstructure:"tie-token-rate"`    // 平局扣除赌注比例
	TiePointRate      float64 `mapstructure:"tie-point-rate"`    // 平局积分倍率
	TokenThreshold    int     `mapstructure:"token-threshold"`   // 加入匹配所需最低可用代币
	BaseStake         int     `mapstructure:"base-stake"`        // 计算奖励时使用的基准赌注
}

// 全局可读的游戏参数
var GameParams = GameParamConfig{}
