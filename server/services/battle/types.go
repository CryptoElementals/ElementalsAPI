package battle

// Card 卡牌结构
type Card struct {
	Symbol    string `json:"symbol"`     // 卡牌符号 (J, M, S, H, T)
	SubType   int    `json:"sub_type"`   // 小型号 (0, 1, 2, 3)
	Level     string `json:"level"`      // 卡牌等级 (legendary, epic, rare, normal)
	LifeForce int    `json:"life_force"` // 生命力
	Attack    int    `json:"attack"`     // 攻击力
	Defense   int    `json:"defense"`    // 防御力
}

// Player 玩家结构
type Player struct {
	Address    string  `json:"address"`    // 玩家地址
	Cards      []Card  `json:"cards"`      // 玩家的卡牌
	HP         int     `json:"hp"`         // 血量
	Multiplier float64 `json:"multiplier"` // 积分倍率
	IsMyself   bool    `json:"is_myself"`  // 是否是我自己
}

// StageResult 阶段结果
type StageResult struct {
	Stage             int                `json:"stage"`              // 阶段编号 (0, 1, 2)
	Player1Cards      []Card             `json:"player1_cards"`      // 玩家1的卡牌
	Player2Cards      []Card             `json:"player2_cards"`      // 玩家2的卡牌
	Player1HP         int                `json:"player1_hp"`         // 玩家1血量
	Player2HP         int                `json:"player2_hp"`         // 玩家2血量
	Player1Multiplier float64            `json:"player1_multiplier"` // 玩家1积分倍率
	Player2Multiplier float64            `json:"player2_multiplier"` // 玩家2积分倍率
	BattleResults     []CardBattleResult `json:"battle_results"`     // 卡牌对战结果
	IsGameOver        bool               `json:"is_game_over"`       // 游戏是否结束
	Winner            string             `json:"winner"`             // 获胜者地址
}

// CardBattleResult 单张卡牌对战结果
type CardBattleResult struct {
	Player1Card Card   `json:"player1_card"` // 玩家1的卡牌
	Player2Card Card   `json:"player2_card"` // 玩家2的卡牌
	ResultType  string `json:"result_type"`  // 对战结果类型 ("sheng", "beisheng", "ke", "beike", "ping")
	EffectValue []int  `json:"effect_value"` // 对战效果值数组 [玩家1受影响值, 玩家2受影响值]（正数表示回血，负数表示伤害）
	Reason      string `json:"reason"`       // 对战原因描述
}

// StageBattleInput 阶段对战输入
type StageBattleInput struct {
	Player1Address    string   `json:"player1_address"`    // 玩家1地址
	Player2Address    string   `json:"player2_address"`    // 玩家2地址
	Player1HP         int      `json:"player1_hp"`         // 玩家1初始血量
	Player2HP         int      `json:"player2_hp"`         // 玩家2初始血量
	Player1Multiplier float64  `json:"player1_multiplier"` // 玩家1积分倍率
	Player2Multiplier float64  `json:"player2_multiplier"` // 玩家2积分倍率
	Player1Cards      []string `json:"player1_cards"`      // 玩家1的3张卡牌
	Player2Cards      []string `json:"player2_cards"`      // 玩家2的3张卡牌
}

// StageBattleResult 阶段对战结果
type StageBattleResult struct {
	Stage             int                `json:"stage"`              // 阶段编号 (0, 1, 2)
	Player1Address    string             `json:"player1_address"`    // 玩家1地址
	Player2Address    string             `json:"player2_address"`    // 玩家2地址
	Player1HP         int                `json:"player1_hp"`         // 玩家1最终血量
	Player2HP         int                `json:"player2_hp"`         // 玩家2最终血量
	Player1Multiplier float64            `json:"player1_multiplier"` // 玩家1积分倍率
	Player2Multiplier float64            `json:"player2_multiplier"` // 玩家2积分倍率
	BattleResults     []CardBattleResult `json:"battle_results"`     // 3轮对战结果
	DetailedActions   []BattleAction     `json:"detailed_actions"`   // 详细动作数据
	IsGameOver        bool               `json:"is_game_over"`       // 游戏是否结束
	Winner            string             `json:"winner"`             // 获胜者地址（如果游戏结束）
}

// BattleSimulation 完整对战模拟结果
type BattleSimulation struct {
	RoomID           string         `json:"room_id"`           // 房间ID
	Stage            int            `json:"stage"`             // 当前阶段
	Player1          Player         `json:"player1"`           // 玩家1信息
	Player2          Player         `json:"player2"`           // 玩家2信息
	Actions          []BattleAction `json:"actions"`           // 当前阶段的动作信息
	IsGameOver       bool           `json:"is_game_over"`      // 游戏是否结束
	GameResult       string         `json:"game_result"`       // 游戏结果：win/lose/tie
	WinnerMultiplier float64        `json:"winner_multiplier"` // 赢家的最终倍率
}

// BattleAction 单次对战动作
type BattleAction struct {
	Round      int          `json:"round"`       // 回合数
	ActionType string       `json:"action_type"` // 动作类型：attack, heal
	Source     ActionActor  `json:"source"`      // 动作发起者
	Target     ActionActor  `json:"target"`      // 动作接收者
	Effect     ActionEffect `json:"effect"`      // 动作效果
	Message    string       `json:"message"`     // 描述信息
}

// ActionActor 动作参与者
type ActionActor struct {
	Address string `json:"address"` // 玩家地址
	Card    Card   `json:"card"`    // 卡牌信息
}

// ActionEffect 动作效果
type ActionEffect struct {
	Type  string `json:"type"`  // 效果类型：damage, heal
	Value int    `json:"value"` // 效果数值
}
