package battle

import "fmt"

// ActionGenerator 动作生成器
type ActionGenerator struct{}

// NewActionGenerator 创建新的动作生成器
func NewActionGenerator() *ActionGenerator {
	return &ActionGenerator{}
}

// GenerateDetailedActions 生成详细动作数据
func (ag *ActionGenerator) GenerateDetailedActions(result *StageBattleResult, player1Addr, player2Addr string) []BattleAction {
	var actions []BattleAction

	// 遍历每轮对战结果
	for round, battleResult := range result.BattleResults {
		// 根据对战结果类型生成详细动作
		roundActions := ag.generateActionsFromBattleResult(round+1, battleResult, player1Addr, player2Addr)
		actions = append(actions, roundActions...)
	}

	return actions
}

// generateActionsFromBattleResult 从对战结果生成详细动作
func (ag *ActionGenerator) generateActionsFromBattleResult(round int, battleResult CardBattleResult, player1Addr, player2Addr string) []BattleAction {
	var actions []BattleAction

	switch battleResult.ResultType {
	case "ke":
		// 克：攻击对方两次
		damage := battleResult.Player1Card.Attack - battleResult.Player2Card.Defense
		if damage < 0 {
			damage = 0
		}

		// 第一次攻击
		action1 := BattleAction{
			Round:      round,
			ActionType: "attack",
			Source: ActionActor{
				Address: player1Addr,
				Card:    battleResult.Player1Card,
			},
			Target: ActionActor{
				Address: player2Addr,
				Card:    battleResult.Player2Card,
			},
			Effect: ActionEffect{
				Type:  "damage",
				Value: damage,
			},
			Message: fmt.Sprintf("%s克%s，第一次攻击", battleResult.Player1Card.Symbol, battleResult.Player2Card.Symbol),
		}
		actions = append(actions, action1)

		// 第二次攻击
		action2 := BattleAction{
			Round:      round,
			ActionType: "attack",
			Source: ActionActor{
				Address: player1Addr,
				Card:    battleResult.Player1Card,
			},
			Target: ActionActor{
				Address: player2Addr,
				Card:    battleResult.Player2Card,
			},
			Effect: ActionEffect{
				Type:  "damage",
				Value: damage,
			},
			Message: fmt.Sprintf("%s克%s，第二次攻击", battleResult.Player1Card.Symbol, battleResult.Player2Card.Symbol),
		}
		actions = append(actions, action2)

	case "beike":
		// 被克：被攻击两次
		damage := battleResult.Player2Card.Attack - battleResult.Player1Card.Defense
		if damage < 0 {
			damage = 0
		}

		// 第一次被攻击
		action1 := BattleAction{
			Round:      round,
			ActionType: "attack",
			Source: ActionActor{
				Address: player2Addr,
				Card:    battleResult.Player2Card,
			},
			Target: ActionActor{
				Address: player1Addr,
				Card:    battleResult.Player1Card,
			},
			Effect: ActionEffect{
				Type:  "damage",
				Value: damage,
			},
			Message: fmt.Sprintf("%s被%s克，第一次被攻击", battleResult.Player1Card.Symbol, battleResult.Player2Card.Symbol),
		}
		actions = append(actions, action1)

		// 第二次被攻击
		action2 := BattleAction{
			Round:      round,
			ActionType: "attack",
			Source: ActionActor{
				Address: player2Addr,
				Card:    battleResult.Player2Card,
			},
			Target: ActionActor{
				Address: player1Addr,
				Card:    battleResult.Player1Card,
			},
			Effect: ActionEffect{
				Type:  "damage",
				Value: damage,
			},
			Message: fmt.Sprintf("%s被%s克，第二次被攻击", battleResult.Player1Card.Symbol, battleResult.Player2Card.Symbol),
		}
		actions = append(actions, action2)

	case "sheng":
		// 生：给对方回血
		healAmount := battleResult.Player1Card.LifeForce
		action := BattleAction{
			Round:      round,
			ActionType: "heal",
			Source: ActionActor{
				Address: player1Addr,
				Card:    battleResult.Player1Card,
			},
			Target: ActionActor{
				Address: player2Addr,
				Card:    battleResult.Player2Card,
			},
			Effect: ActionEffect{
				Type:  "heal",
				Value: healAmount,
			},
			Message: fmt.Sprintf("%s生%s，治疗%d点血量", battleResult.Player1Card.Symbol, battleResult.Player2Card.Symbol, healAmount),
		}
		actions = append(actions, action)

	case "beisheng":
		// 被生：自己回血
		healAmount := battleResult.Player2Card.LifeForce
		action := BattleAction{
			Round:      round,
			ActionType: "heal",
			Source: ActionActor{
				Address: player2Addr,
				Card:    battleResult.Player2Card,
			},
			Target: ActionActor{
				Address: player1Addr,
				Card:    battleResult.Player1Card,
			},
			Effect: ActionEffect{
				Type:  "heal",
				Value: healAmount,
			},
			Message: fmt.Sprintf("%s被%s生，治疗%d点血量", battleResult.Player1Card.Symbol, battleResult.Player2Card.Symbol, healAmount),
		}
		actions = append(actions, action)

	case "ping":
		// 平：双方各攻击对方一次（拆解为两个独立动作）
		damage1 := battleResult.Player1Card.Attack - battleResult.Player2Card.Defense
		if damage1 < 0 {
			damage1 = 0
		}
		damage2 := battleResult.Player2Card.Attack - battleResult.Player1Card.Defense
		if damage2 < 0 {
			damage2 = 0
		}

		// 玩家1攻击玩家2
		action1 := BattleAction{
			Round:      round,
			ActionType: "attack",
			Source: ActionActor{
				Address: player1Addr,
				Card:    battleResult.Player1Card,
			},
			Target: ActionActor{
				Address: player2Addr,
				Card:    battleResult.Player2Card,
			},
			Effect: ActionEffect{
				Type:  "damage",
				Value: damage1,
			},
			Message: fmt.Sprintf("无生克关系，%s攻击%s", battleResult.Player1Card.Symbol, battleResult.Player2Card.Symbol),
		}
		actions = append(actions, action1)

		// 玩家2攻击玩家1
		action2 := BattleAction{
			Round:      round,
			ActionType: "attack",
			Source: ActionActor{
				Address: player2Addr,
				Card:    battleResult.Player2Card,
			},
			Target: ActionActor{
				Address: player1Addr,
				Card:    battleResult.Player1Card,
			},
			Effect: ActionEffect{
				Type:  "damage",
				Value: damage2,
			},
			Message: fmt.Sprintf("无生克关系，%s攻击%s", battleResult.Player2Card.Symbol, battleResult.Player1Card.Symbol),
		}
		actions = append(actions, action2)
	}

	return actions
}

// ConvertStageResultToActions 将阶段结果转换为动作列表
func (ag *ActionGenerator) ConvertStageResultToActions(stageResult *StageBattleResult, player1Addr, player2Addr string) []BattleAction {
	var actions []BattleAction

	// 遍历每轮对战结果
	for round, battleResult := range stageResult.BattleResults {
		// 根据对战结果类型生成详细动作
		roundActions := ag.generateActionsFromBattleResult(round+1, battleResult, player1Addr, player2Addr)
		actions = append(actions, roundActions...)
	}

	return actions
}
