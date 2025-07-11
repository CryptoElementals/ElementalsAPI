package services

import (
	"github.com/CryptoElementals/common/server/services/battle"
)

// 重新导出类型，保持向后兼容
type Card = battle.Card
type Player = battle.Player
type StageResult = battle.StageResult
type CardBattleResult = battle.CardBattleResult
type StageBattleInput = battle.StageBattleInput
type StageBattleResult = battle.StageBattleResult
type BattleSimulation = battle.BattleSimulation
type BattleAction = battle.BattleAction
type ActionActor = battle.ActionActor
type ActionEffect = battle.ActionEffect

// BattleSimulator 对战推演器（使用新的模块化系统）
type BattleSimulator struct {
	*battle.BattleSimulator
}

// NewBattleSimulator 创建新的对战推演器
func NewBattleSimulator() *BattleSimulator {
	return &BattleSimulator{
		BattleSimulator: battle.NewBattleSimulator(),
	}
}
