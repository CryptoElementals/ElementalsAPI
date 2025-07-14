package battle

import (
	"fmt"
	"strings"

	"github.com/CryptoElementals/common/db"
	"github.com/CryptoElementals/common/log"
	dao "github.com/CryptoElementals/common/models"
	"github.com/CryptoElementals/common/server/api"
	"github.com/CryptoElementals/common/server/services"
	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
	"github.com/mitchellh/mapstructure"
)

const GET_BATTLE_INFO_LABEL = "GetBattleInfo"

// GetBattleInfoRequest 请求结构体
type GetBattleInfoRequest struct {
	api.BaseRequest
	RoomID string `mapstructure:"RoomId" validate:"required"`
}

// GetBattleInfoResponse 响应结构体
type GetBattleInfoResponse struct {
	api.BaseResponse
	BattleInfo *services.BattleSimulation `json:"BattleInfo"`
}

type GetBattleInfoTask struct {
	Request  *GetBattleInfoRequest
	Response *GetBattleInfoResponse
}

// 解码请求
func NewGetBattleInfoRequest(data *map[string]interface{}) (*GetBattleInfoRequest, error) {
	req := &GetBattleInfoRequest{}
	err := mapstructure.Decode(*data, &req)
	if err != nil {
		return nil, err
	}
	req.BaseRequest.RequestUUID = (*data)["RequestUUID"].(string)
	return req, nil
}

func NewGetBattleInfoResponse(sessionId string) *GetBattleInfoResponse {
	return &GetBattleInfoResponse{
		BaseResponse: api.BaseResponse{
			Action:      GET_BATTLE_INFO_LABEL + "Response",
			RequestUUID: sessionId,
		},
	}
}

func NewGetBattleInfoTask(data *map[string]interface{}) (api.Task, error) {
	req, err := NewGetBattleInfoRequest(data)
	if err != nil {
		return nil, err
	}
	task := &GetBattleInfoTask{
		Request:  req,
		Response: NewGetBattleInfoResponse(req.BaseRequest.RequestUUID),
	}

	validate := validator.New()
	err = validate.Struct(task.Request)
	if err != nil {
		return nil, err
	}

	return task, nil
}

func (task *GetBattleInfoTask) Run(c *gin.Context) (api.Response, error) {
	// 获取玩家地址（从认证中间件设置的params中获取）
	_params, _ := c.Get("params")
	params, ok := _params.(*map[string]interface{})
	if !ok {
		task.Response.BaseResponse.RetCode = 1001
		task.Response.BaseResponse.Message = "参数解析失败"
		return task.Response, nil
	}

	address, ok := (*params)["Address"].(string)
	if !ok || address == "" {
		task.Response.BaseResponse.RetCode = 1001
		task.Response.BaseResponse.Message = "未获取到玩家地址"
		return task.Response, nil
	}

	// 根据RoomID获取所有房间记录
	rooms, err := db.GetRoomsByRoomID(task.Request.RoomID)
	if err != nil {
		log.Errorf("%s, failed to get rooms for room_id %s: %v", task.Request.RequestUUID, task.Request.RoomID, err)
		task.Response.BaseResponse.RetCode = 1002
		task.Response.BaseResponse.Message = "房间不存在"
		return task.Response, nil
	}

	// 检查房间记录数量
	if len(rooms) == 0 {
		task.Response.BaseResponse.RetCode = 1002
		task.Response.BaseResponse.Message = "房间不存在"
		return task.Response, nil
	}

	// 验证玩家是否是该房间的参与者
	found := false
	for _, room := range rooms {
		if strings.ToLower(room.Address) == strings.ToLower(address) {
			found = true
			break
		}
	}

	if !found {
		task.Response.BaseResponse.RetCode = 1003
		task.Response.BaseResponse.Message = "您不是该房间的参与者"
		return task.Response, nil
	}

	// 找到当前最新的已完成的stage
	var maxCompletedStage uint = 0
	for _, room := range rooms {
		if room.IsStageOver && room.Stage > maxCompletedStage {
			maxCompletedStage = room.Stage
		}
	}

	// 如果没有找到任何已完成的stage，返回错误
	if maxCompletedStage == 0 {
		task.Response.BaseResponse.RetCode = 1004
		task.Response.BaseResponse.Message = "房间中没有已完成的阶段数据"
		return task.Response, nil
	}

	// 获取该已完成的stage的所有房间记录
	var stageRooms []dao.Room
	for _, room := range rooms {
		if room.Stage == maxCompletedStage && room.IsStageOver {
			stageRooms = append(stageRooms, room)
		}
	}

	// 检查是否有两个玩家的记录
	if len(stageRooms) != 2 {
		task.Response.BaseResponse.RetCode = 1005
		task.Response.BaseResponse.Message = "该阶段数据不完整，需要两个玩家"
		return task.Response, nil
	}

	// 获取两个玩家的信息
	var player1, player2 *dao.Room
	for i := range stageRooms {
		if i == 0 {
			player1 = &stageRooms[i]
		} else {
			player2 = &stageRooms[i]
		}
	}

	// 创建响应数据（直接使用数据库中的数据）
	battleInfo := &services.BattleSimulation{
		RoomID: task.Request.RoomID,
		Stage:  int(maxCompletedStage),
		Player1: services.Player{
			Address:    player1.Address,
			HP:         player1.PlayerHP,
			Multiplier: player1.Multiplier,
			IsMyself:   strings.ToLower(player1.Address) == strings.ToLower(address),
		},
		Player2: services.Player{
			Address:    player2.Address,
			HP:         player2.PlayerHP,
			Multiplier: player2.Multiplier,
			IsMyself:   strings.ToLower(player2.Address) == strings.ToLower(address),
		},
		Actions:    []services.BattleAction{}, // 空数组，详细结果可通过日志查看
		IsGameOver: maxCompletedStage == 10,   // stage 10表示游戏结束
		GameResult: "",                        // 将在下面设置
	}

	// 如果游戏结束，设置游戏结果和赢家倍率
	if maxCompletedStage == 10 {
		// 设置游戏结果
		if player1.PlayerHP <= 0 {
			// 玩家1血量为0，玩家2获胜
			if strings.ToLower(player1.Address) == strings.ToLower(address) {
				battleInfo.GameResult = "lose" // 当前用户失败
			} else {
				battleInfo.GameResult = "win" // 当前用户获胜
			}
		} else if player2.PlayerHP <= 0 {
			// 玩家2血量为0，玩家1获胜
			if strings.ToLower(player1.Address) == strings.ToLower(address) {
				battleInfo.GameResult = "win" // 当前用户获胜
			} else {
				battleInfo.GameResult = "lose" // 当前用户失败
			}
		} else {
			// 血量比较（stage 3的情况）
			if player1.PlayerHP > player2.PlayerHP {
				if strings.ToLower(player1.Address) == strings.ToLower(address) {
					battleInfo.GameResult = "win" // 当前用户获胜
				} else {
					battleInfo.GameResult = "lose" // 当前用户失败
				}
			} else if player2.PlayerHP > player1.PlayerHP {
				if strings.ToLower(player1.Address) == strings.ToLower(address) {
					battleInfo.GameResult = "lose" // 当前用户失败
				} else {
					battleInfo.GameResult = "win" // 当前用户获胜
				}
			} else {
				battleInfo.GameResult = "tie" // 平局
			}
		}

		// 设置赢家的最终倍率（stage 10中双方倍率相同，都是赢家的最高倍率）
		battleInfo.WinnerMultiplier = player1.Multiplier // 在stage 10中，双方倍率相同
	} else {
		// 非stage 10，设置当前阶段的倍率
		// 对于stage 1-3，使用当前阶段的倍率
		currentStageMultiplier := 1.0 + float64(maxCompletedStage-1)*0.5
		battleInfo.WinnerMultiplier = currentStageMultiplier
	}

	// 构建详细的对战过程数据
	battleInfo.Actions = task.buildBattleDetails(task.Request.RoomID, maxCompletedStage, player1, player2)

	task.Response.BattleInfo = battleInfo
	task.Response.BaseResponse.RetCode = 0
	task.Response.BaseResponse.Message = "获取对战信息成功"

	log.Infof("%s, battle info retrieved for room %s, stage %d", task.Request.RequestUUID, task.Request.RoomID, maxCompletedStage)
	return task.Response, nil
}

// buildBattleDetails 构建详细的对战过程数据
func (task *GetBattleInfoTask) buildBattleDetails(roomID string, stage uint, player1, player2 *dao.Room) []services.BattleAction {
	var battleDetails []services.BattleAction

	// 解析玩家卡牌
	player1Cards := task.parseCardsString(player1.Cards)
	player2Cards := task.parseCardsString(player2.Cards)

	// 如果没有卡牌数据，返回空数组
	if len(player1Cards) == 0 || len(player2Cards) == 0 {
		log.Warnf("No card data found for room %s, stage %d", roomID, stage)
		return battleDetails
	}

	// 获取上一stage的血量作为初始血量
	prevStageRooms, err := db.GetRoomsByStage(roomID, stage-1)
	if err != nil {
		log.Errorf("Failed to get previous stage %d records for room %s: %v", stage-1, roomID, err)
		return battleDetails
	}

	if len(prevStageRooms) != 2 {
		log.Errorf("Previous stage %d has %d players, expected 2 for room %s", stage-1, len(prevStageRooms), roomID)
		return battleDetails
	}

	// 获取上一stage的血量
	var player1InitialHP, player2InitialHP int
	for _, room := range prevStageRooms {
		if strings.ToLower(room.Address) == strings.ToLower(player1.Address) {
			player1InitialHP = room.PlayerHP
		} else if strings.ToLower(room.Address) == strings.ToLower(player2.Address) {
			player2InitialHP = room.PlayerHP
		}
	}

	return task.simulateAndConvertToActions(stage, player1Cards, player2Cards,
		player1.Address, player2.Address, player1InitialHP, player2InitialHP)
}

// simulateAndConvertToActions 模拟对战并转换为详细动作
func (task *GetBattleInfoTask) simulateAndConvertToActions(stage uint, player1Cards, player2Cards []string,
	player1Addr, player2Addr string, player1InitialHP, player2InitialHP int) []services.BattleAction {

	// 创建对战模拟器
	simulator := services.NewBattleSimulator()

	// 创建对战输入
	input := &services.StageBattleInput{
		Player1Address:    player1Addr,
		Player2Address:    player2Addr,
		Player1HP:         player1InitialHP,
		Player2HP:         player2InitialHP,
		Player1Multiplier: 1.0 + float64(stage-1)*0.5,
		Player2Multiplier: 1.0 + float64(stage-1)*0.5,
		Player1Cards:      player1Cards,
		Player2Cards:      player2Cards,
	}

	// 执行对战模拟
	result, err := simulator.SimulateStage(input, int(stage))
	if err != nil {
		log.Errorf("Failed to simulate battle: %v", err)
		return []services.BattleAction{}
	}

	// 将对战结果转换为详细动作
	return task.convertBattleResultsToActions(result, player1InitialHP, player2InitialHP)
}

// convertBattleResultsToActions 将对战结果转换为详细动作
func (task *GetBattleInfoTask) convertBattleResultsToActions(result *services.StageBattleResult,
	player1InitialHP, player2InitialHP int) []services.BattleAction {

	var battleActions []services.BattleAction

	// 遍历每轮对战结果
	for round, battleResult := range result.BattleResults {
		// 根据对战结果类型生成详细动作
		actions := task.generateActionsFromBattleResult(round+1, battleResult,
			result.Player1Address, result.Player2Address,
			player1InitialHP, player2InitialHP)

		battleActions = append(battleActions, actions...)
	}

	return battleActions
}

// generateActionsFromBattleResult 从对战结果生成详细动作
func (task *GetBattleInfoTask) generateActionsFromBattleResult(round int, battleResult services.CardBattleResult,
	player1Addr, player2Addr string, player1HP, player2HP int) []services.BattleAction {

	var actions []services.BattleAction

	switch battleResult.ResultType {
	case "ke":
		// 克：攻击对方两次
		damage := battleResult.Player1Card.Attack - battleResult.Player2Card.Defense
		if damage < 0 {
			damage = 0
		}

		// 第一次攻击
		action1 := services.BattleAction{
			Round:      round,
			ActionType: "attack",
			Source: services.ActionActor{
				Address: player1Addr,
				Card:    battleResult.Player1Card,
			},
			Target: services.ActionActor{
				Address: player2Addr,
				Card:    battleResult.Player2Card,
			},
			Effect: services.ActionEffect{
				Type:  "damage",
				Value: damage,
			},
			Message: fmt.Sprintf("%s克%s，第一次攻击", battleResult.Player1Card.Symbol, battleResult.Player2Card.Symbol),
		}
		actions = append(actions, action1)

		// 第二次攻击
		action2 := services.BattleAction{
			Round:      round,
			ActionType: "attack",
			Source: services.ActionActor{
				Address: player1Addr,
				Card:    battleResult.Player1Card,
			},
			Target: services.ActionActor{
				Address: player2Addr,
				Card:    battleResult.Player2Card,
			},
			Effect: services.ActionEffect{
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
		action1 := services.BattleAction{
			Round:      round,
			ActionType: "attack",
			Source: services.ActionActor{
				Address: player2Addr,
				Card:    battleResult.Player2Card,
			},
			Target: services.ActionActor{
				Address: player1Addr,
				Card:    battleResult.Player1Card,
			},
			Effect: services.ActionEffect{
				Type:  "damage",
				Value: damage,
			},
			Message: fmt.Sprintf("%s被%s克，第一次被攻击", battleResult.Player1Card.Symbol, battleResult.Player2Card.Symbol),
		}
		actions = append(actions, action1)

		// 第二次被攻击
		action2 := services.BattleAction{
			Round:      round,
			ActionType: "attack",
			Source: services.ActionActor{
				Address: player2Addr,
				Card:    battleResult.Player2Card,
			},
			Target: services.ActionActor{
				Address: player1Addr,
				Card:    battleResult.Player1Card,
			},
			Effect: services.ActionEffect{
				Type:  "damage",
				Value: damage,
			},
			Message: fmt.Sprintf("%s被%s克，第二次被攻击", battleResult.Player1Card.Symbol, battleResult.Player2Card.Symbol),
		}
		actions = append(actions, action2)

	case "sheng":
		// 生：给对方回血
		healAmount := battleResult.Player1Card.LifeForce
		action := services.BattleAction{
			Round:      round,
			ActionType: "heal",
			Source: services.ActionActor{
				Address: player1Addr,
				Card:    battleResult.Player1Card,
			},
			Target: services.ActionActor{
				Address: player2Addr,
				Card:    battleResult.Player2Card,
			},
			Effect: services.ActionEffect{
				Type:  "heal",
				Value: healAmount,
			},
			Message: fmt.Sprintf("%s生%s，治疗%d点血量", battleResult.Player1Card.Symbol, battleResult.Player2Card.Symbol, healAmount),
		}
		actions = append(actions, action)

	case "beisheng":
		// 被生：自己回血
		healAmount := battleResult.Player2Card.LifeForce
		action := services.BattleAction{
			Round:      round,
			ActionType: "heal",
			Source: services.ActionActor{
				Address: player2Addr,
				Card:    battleResult.Player2Card,
			},
			Target: services.ActionActor{
				Address: player1Addr,
				Card:    battleResult.Player1Card,
			},
			Effect: services.ActionEffect{
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
		action1 := services.BattleAction{
			Round:      round,
			ActionType: "attack",
			Source: services.ActionActor{
				Address: player1Addr,
				Card:    battleResult.Player1Card,
			},
			Target: services.ActionActor{
				Address: player2Addr,
				Card:    battleResult.Player2Card,
			},
			Effect: services.ActionEffect{
				Type:  "damage",
				Value: damage1,
			},
			Message: fmt.Sprintf("无生克关系，%s攻击%s", battleResult.Player1Card.Symbol, battleResult.Player2Card.Symbol),
		}
		actions = append(actions, action1)

		// 玩家2攻击玩家1
		action2 := services.BattleAction{
			Round:      round,
			ActionType: "attack",
			Source: services.ActionActor{
				Address: player2Addr,
				Card:    battleResult.Player2Card,
			},
			Target: services.ActionActor{
				Address: player1Addr,
				Card:    battleResult.Player1Card,
			},
			Effect: services.ActionEffect{
				Type:  "damage",
				Value: damage2,
			},
			Message: fmt.Sprintf("无生克关系，%s攻击%s", battleResult.Player2Card.Symbol, battleResult.Player1Card.Symbol),
		}
		actions = append(actions, action2)
	}

	return actions
}

// parseCardsString 解析卡牌字符串
func (task *GetBattleInfoTask) parseCardsString(cardsStr string) []string {
	if cardsStr == "" {
		return []string{}
	}
	// 假设格式是 "J0|M1|S2"
	return strings.Split(cardsStr, "|")
}

// RegisterBattleApis 注册对战相关API
func RegisterBattleApis() {
	api.Register(GET_BATTLE_INFO_LABEL, NewGetBattleInfoTask, api.COOKIEAUTH)
	api.Register(SSE_EXAMPLE_LABEL, NewGetBattleInfoTask, api.NOAUTH)
}
