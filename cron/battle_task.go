package cron

import (
	"context"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/CryptoElementals/common/db"
	"github.com/CryptoElementals/common/log"
	dao "github.com/CryptoElementals/common/models"
	"github.com/CryptoElementals/common/server/services"
)

// BattleTask 对战处理任务
type BattleTask struct {
	simulator *services.BattleSimulator
	interval  time.Duration
}

// NewBattleTask 创建新的对战处理任务
func NewBattleTask() *BattleTask {
	// 获取间隔时间
	interval := 1 * time.Second // 默认1秒
	if intervalStr := os.Getenv("BATTLE_PROCESSOR_INTERVAL"); intervalStr != "" {
		if seconds, err := strconv.Atoi(intervalStr); err == nil && seconds > 0 {
			interval = time.Duration(seconds) * time.Second
		}
	}

	return &BattleTask{
		simulator: services.NewBattleSimulator(),
		interval:  interval,
	}
}

// Run 执行对战处理任务
func (t *BattleTask) Run(ctx context.Context) {
	// 检查是否启用（默认启用，只有在明确设置为false时才禁用）
	enabled := true
	if enabledStr := os.Getenv("BATTLE_PROCESSOR_ENABLED"); enabledStr != "" {
		enabled = enabledStr == "true" || enabledStr == "1"
	}

	if !enabled {
		return // 如果明确禁用，直接返回
	}

	// 处理所有需要推演的房间
	t.processRooms()
}

// processRooms 处理所有需要推演的房间
func (t *BattleTask) processRooms() {
	// 获取所有需要处理的房间记录
	rooms, err := db.GetRoomsForBattleProcessing()
	if err != nil {
		log.Errorf("Failed to get rooms for battle processing: %v", err)
		return
	}

	if len(rooms) == 0 {
		log.Debug("No rooms need battle processing")
		return // 没有需要处理的房间
	}

	log.Infof("Found %d rooms for battle processing", len(rooms))

	// 按RoomID分组
	roomGroups := t.groupRoomsByRoomID(rooms)
	log.Infof("Grouped into %d room groups", len(roomGroups))

	// 处理每个房间组
	for roomID, roomList := range roomGroups {
		log.Infof("Processing room group: %s with %d players", roomID, len(roomList))
		err := t.processRoomGroup(roomID, roomList)
		if err != nil {
			log.Errorf("Failed to process room group %s: %v", roomID, err)
		}
	}
}

// groupRoomsByRoomID 按RoomID分组房间记录
func (t *BattleTask) groupRoomsByRoomID(rooms []dao.Room) map[string][]dao.Room {
	groups := make(map[string][]dao.Room)
	for _, room := range rooms {
		groups[room.RoomID] = append(groups[room.RoomID], room)
	}
	return groups
}

// processRoomGroup 处理单个房间组
func (t *BattleTask) processRoomGroup(roomID string, rooms []dao.Room) error {
	// 找到当前需要处理的最小stage
	currentStage := t.findCurrentStage(rooms)
	if currentStage == 0 {
		log.Debugf("No valid stage found for room %s", roomID)
		return nil // 没有需要处理的stage，不是错误
	}

	log.Infof("Found current stage %d for room %s", currentStage, roomID)

	// 获取当前stage的记录
	currentStageRooms := t.getRoomsByStage(rooms, currentStage)
	if len(currentStageRooms) != 2 {
		log.Debugf("Current stage %d has %d players, waiting for more players in room %s", currentStage, len(currentStageRooms), roomID)
		return nil // 当前stage不完整，等待更多玩家
	}

	// 直接从数据库获取上一stage的记录（用于获取HP和倍率）
	prevStageRooms, err := db.GetRoomsByStage(roomID, currentStage-1)
	if err != nil {
		log.Errorf("Failed to get previous stage %d records for room %s: %v", currentStage-1, roomID, err)
		return err
	}

	log.Infof("Found %d players in previous stage %d for room %s", len(prevStageRooms), currentStage-1, roomID)
	if len(prevStageRooms) != 2 {
		log.Debugf("Previous stage %d has %d players, waiting for more players in room %s", currentStage-1, len(prevStageRooms), roomID)
		return nil // 上一stage不完整，等待更多玩家
	}

	log.Infof("Starting battle simulation for room %s, stage %d", roomID, currentStage)

	// 执行对战推演
	err = t.simulateAndUpdate(roomID, currentStage, currentStageRooms, prevStageRooms)
	if err != nil {
		return err
	}

	log.Infof("Successfully processed room %s, stage %d", roomID, currentStage)
	return nil
}

// findCurrentStage 找到当前需要处理的最小stage
func (t *BattleTask) findCurrentStage(rooms []dao.Room) uint {
	var minStage uint = 0
	for _, room := range rooms {
		if !room.IsStageOver && room.Stage > 0 && room.Stage < 10 {
			if minStage == 0 || room.Stage < minStage {
				minStage = room.Stage
			}
		}
	}
	return minStage
}

// getRoomsByStage 获取指定stage的房间记录
func (t *BattleTask) getRoomsByStage(rooms []dao.Room, stage uint) []dao.Room {
	var result []dao.Room
	for _, room := range rooms {
		if room.Stage == stage {
			result = append(result, room)
		}
	}
	return result
}

// simulateAndUpdate 执行对战推演并更新数据
func (t *BattleTask) simulateAndUpdate(roomID string, currentStage uint, currentRooms, prevRooms []dao.Room) error {
	// 获取玩家信息
	player1, player2 := t.getPlayers(currentRooms)
	prevPlayer1, prevPlayer2 := t.getPlayers(prevRooms)

	// 解析卡牌
	player1Cards := t.parseCardsString(player1.Cards)
	player2Cards := t.parseCardsString(player2.Cards)

	// 验证卡牌数量
	if len(player1Cards) != 3 || len(player2Cards) != 3 {
		return nil // 卡牌数量不正确，等待完整数据
	}

	// 创建对战输入
	input := &services.StageBattleInput{
		Player1Address:    player1.Address,
		Player2Address:    player2.Address,
		Player1HP:         prevPlayer1.PlayerHP,   // 使用上一stage的HP
		Player2HP:         prevPlayer2.PlayerHP,   // 使用上一stage的HP
		Player1Multiplier: prevPlayer1.Multiplier, // 使用上一stage的倍率
		Player2Multiplier: prevPlayer2.Multiplier, // 使用上一stage的倍率
		Player1Cards:      player1Cards,
		Player2Cards:      player2Cards,
	}

	// 执行对战推演
	result, err := t.simulator.SimulateStage(input, int(currentStage))
	if err != nil {
		return err
	}

	// 输出详细的对战过程日志
	t.logBattleDetails(roomID, currentStage, player1Cards, player2Cards, prevPlayer1.PlayerHP, prevPlayer2.PlayerHP, prevPlayer1.Multiplier, prevPlayer2.Multiplier, result)

	// 更新当前stage的状态为已完成
	err = t.updateRoomBattleState(roomID, player1.Address, currentStage, result.Player1HP, result.Player1Multiplier, true)
	if err != nil {
		return err
	}

	err = t.updateRoomBattleState(roomID, player2.Address, currentStage, result.Player2HP, result.Player2Multiplier, true)
	if err != nil {
		return err
	}

	// 如果游戏结束，创建stage 10的记录
	if result.IsGameOver {
		// 计算赢家在stage 1-3中的最高倍率
		winnerMaxMultiplier := t.calculateWinnerMaxMultiplier(roomID, result)

		err = t.createGameOverRecords(roomID, player1.Address, player2.Address, result, winnerMaxMultiplier)
		if err != nil {
			return err
		}
	}

	log.Infof("Battle simulation completed for room %s, stage %d: player1_hp=%d, player2_hp=%d, game_over=%v",
		roomID, currentStage, result.Player1HP, result.Player2HP, result.IsGameOver)

	return nil
}

// createGameOverRecords 创建游戏结束记录（stage 10）
func (t *BattleTask) createGameOverRecords(roomID, player1Addr, player2Addr string, result *services.StageBattleResult, winnerMaxMultiplier float64) error {
	// 确定获胜者
	var winner string
	if result.Winner == "tie" {
		winner = "tie" // 平局
	} else if result.Player1HP <= 0 {
		winner = player2Addr
	} else if result.Player2HP <= 0 {
		winner = player1Addr
	} else {
		// 血量比较（stage 3的情况）
		if result.Player1HP > result.Player2HP {
			winner = player1Addr
		} else if result.Player2HP > result.Player1HP {
			winner = player2Addr
		} else {
			winner = "tie" // 平局
		}
	}

	// 为两个玩家创建stage 10的记录，使用赢家的最高倍率
	gameOverRoom1 := &dao.Room{
		RoomID:      roomID,
		Address:     player1Addr,
		Stage:       10,
		Cards:       "",
		PlayerHP:    result.Player1HP,
		Multiplier:  winnerMaxMultiplier, // 使用赢家的最高倍率
		IsStageOver: true,                // stage 10立即设置为完成
	}

	gameOverRoom2 := &dao.Room{
		RoomID:      roomID,
		Address:     player2Addr,
		Stage:       10,
		Cards:       "",
		PlayerHP:    result.Player2HP,
		Multiplier:  winnerMaxMultiplier, // 使用赢家的最高倍率
		IsStageOver: true,                // stage 10立即设置为完成
	}

	// 保存到数据库
	err := db.CreateRoom(gameOverRoom1)
	if err != nil {
		return err
	}

	err = db.CreateRoom(gameOverRoom2)
	if err != nil {
		return err
	}

	// 更新玩家统计数据
	err = db.UpdateUserGameStats(player1Addr, player2Addr, winner, winnerMaxMultiplier)
	if err != nil {
		log.Errorf("Failed to update user game stats for room %s: %v", roomID, err)
		// 不返回错误，避免影响对战逻辑
	} else {
		log.Infof("Updated user game stats for room %s, winner: %s, multiplier: %.2f", roomID, winner, winnerMaxMultiplier)
	}

	// 更新match状态为ended
	err = db.UpdateMatchStatusByRoomID(roomID, "ended")
	if err != nil {
		log.Errorf("Failed to update match status to ended for room %s: %v", roomID, err)
		// 不返回错误，避免影响对战逻辑
	} else {
		log.Infof("Updated match status to ended for room %s", roomID)
	}

	log.Infof("Game over records created for room %s, winner: %s, final multiplier: %.2f", roomID, winner, winnerMaxMultiplier)
	return nil
}

// calculateWinnerMaxMultiplier 计算赢家在stage 1-3中的最高倍率
func (t *BattleTask) calculateWinnerMaxMultiplier(roomID string, result *services.StageBattleResult) float64 {
	// 确定获胜者
	var winner string
	if result.Winner == "tie" {
		return 1.0 // 平局情况下倍率为1.0
	} else if result.Player1HP <= 0 {
		winner = result.Player2Address
	} else if result.Player2HP <= 0 {
		winner = result.Player1Address
	} else {
		// 血量比较（stage 3的情况）
		if result.Player1HP > result.Player2HP {
			winner = result.Player1Address
		} else if result.Player2HP > result.Player1HP {
			winner = result.Player2Address
		} else {
			return 1.0 // 平局情况下倍率为1.0
		}
	}

	// 获取赢家在stage 1-3中的所有倍率记录
	var winnerMultipliers []float64
	for stage := uint(1); stage <= 3; stage++ {
		stageRooms, err := db.GetRoomsByStage(roomID, stage)
		if err != nil {
			log.Errorf("Failed to get stage %d records for room %s: %v", stage, roomID, err)
			continue
		}

		// 找到赢家在该stage的记录
		for _, room := range stageRooms {
			if room.Address == winner && room.IsStageOver {
				winnerMultipliers = append(winnerMultipliers, room.Multiplier)
				break
			}
		}
	}

	// 找到最高倍率
	maxMultiplier := 1.0
	for _, multiplier := range winnerMultipliers {
		if multiplier > maxMultiplier {
			maxMultiplier = multiplier
		}
	}

	log.Infof("Winner %s max multiplier in stages 1-3: %.2f (from %v)", winner, maxMultiplier, winnerMultipliers)
	return maxMultiplier
}

// getPlayers 从房间记录中获取两个玩家
func (t *BattleTask) getPlayers(rooms []dao.Room) (*dao.Room, *dao.Room) {
	if len(rooms) != 2 {
		return nil, nil
	}
	return &rooms[0], &rooms[1]
}

// parseCardsString 解析卡牌字符串
func (t *BattleTask) parseCardsString(cardsStr string) []string {
	if cardsStr == "" {
		return []string{}
	}
	// 假设格式是 "J0|M1|S2"
	return strings.Split(cardsStr, "|")
}

// updateRoomBattleState 更新房间对战状态
func (t *BattleTask) updateRoomBattleState(roomID, address string, stage uint, hp int, multiplier float64, isStageOver bool) error {
	return db.UpdateRoomBattleStateByStage(roomID, address, stage, hp, multiplier, isStageOver)
}

// logBattleDetails 输出详细的对战过程日志
func (t *BattleTask) logBattleDetails(roomID string, stage uint, player1Cards, player2Cards []string, initialPlayer1HP, initialPlayer2HP int, initialPlayer1Multiplier, initialPlayer2Multiplier float64, result *services.StageBattleResult) {
	log.Infof("=== Battle Details for Room %s, Stage %d ===", roomID, stage)
	log.Infof("Player1 Cards: %v, Player2 Cards: %v", player1Cards, player2Cards)
	log.Infof("Initial HP: Player1=%d, Player2=%d", initialPlayer1HP, initialPlayer2HP)
	log.Infof("Initial Multipliers: Player1=%.2f, Player2=%.2f", initialPlayer1Multiplier, initialPlayer2Multiplier)
	log.Infof("Final HP: Player1=%d, Player2=%d", result.Player1HP, result.Player2HP)
	log.Infof("Multipliers: Player1=%.2f, Player2=%.2f", result.Player1Multiplier, result.Player2Multiplier)

	// 输出每张卡牌的对战结果
	for i, battleResult := range result.BattleResults {
		log.Infof("Round %d: %s vs %s", i+1, battleResult.Player1Card.Symbol, battleResult.Player2Card.Symbol)
		log.Infof("  Result: %s", battleResult.ResultType)
		log.Infof("  Effect: Player1=%d, Player2=%d", battleResult.EffectValue[0], battleResult.EffectValue[1])
		log.Infof("  Reason: %s", battleResult.Reason)
	}

	log.Infof("Game Over: %v", result.IsGameOver)
	log.Infof("=== End Battle Details ===")
}

// RegisterBattleTask 注册对战处理任务
func RegisterBattleTask() {
	task := NewBattleTask()

	// 注册对战处理任务，使用配置的间隔时间
	Register("battle_processing", task.Run, false, task.interval)
}
