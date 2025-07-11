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
	Stage  *uint  `mapstructure:"Stage"` // 可选的stage参数，如果指定则查询对应stage的数据
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
		task.Response.BaseResponse.Message = "Failed to parse parameters"
		return task.Response, nil
	}

	address, ok := (*params)["Address"].(string)
	if !ok || address == "" {
		task.Response.BaseResponse.RetCode = 1001
		task.Response.BaseResponse.Message = "Failed to get player address"
		return task.Response, nil
	}

	// 确保地址使用小写
	address = strings.ToLower(address)

	// 根据RoomID获取所有房间记录
	rooms, err := db.GetRoomsByRoomID(task.Request.RoomID)
	if err != nil {
		log.Errorf("%s, failed to get rooms for room_id %s: %v", task.Request.RequestUUID, task.Request.RoomID, err)
		task.Response.BaseResponse.RetCode = 1002
		task.Response.BaseResponse.Message = "Room does not exist"
		return task.Response, nil
	}

	// 检查房间记录数量
	if len(rooms) == 0 {
		task.Response.BaseResponse.RetCode = 1002
		task.Response.BaseResponse.Message = "Room does not exist"
		return task.Response, nil
	}

	// 验证玩家是否是该房间的参与者
	found := false
	for _, room := range rooms {
		if strings.ToLower(room.Address) == address {
			found = true
			break
		}
	}

	if !found {
		task.Response.BaseResponse.RetCode = 1003
		task.Response.BaseResponse.Message = "You are not a participant in this room"
		return task.Response, nil
	}

	// 确定要查询的stage
	var targetStage uint = 0

	if task.Request.Stage != nil {
		// 如果指定了stage，使用指定的stage
		targetStage = *task.Request.Stage

		// 验证指定的stage是否存在且已完成
		stageExists := false
		for _, room := range rooms {
			if room.Stage == targetStage && room.IsStageOver {
				stageExists = true
				break
			}
		}

		if !stageExists {
			task.Response.BaseResponse.RetCode = 1006
			task.Response.BaseResponse.Message = fmt.Sprintf("Stage %d does not exist or is not completed", targetStage)
			return task.Response, nil
		}
	} else {
		// 如果没有指定stage，找到当前最新的已完成的stage
		for _, room := range rooms {
			if room.IsStageOver && room.Stage > targetStage {
				targetStage = room.Stage
			}
		}

		// 如果没有找到任何已完成的stage，返回错误
		if targetStage == 0 {
			task.Response.BaseResponse.RetCode = 1004
			task.Response.BaseResponse.Message = "No completed stage data found in room"
			return task.Response, nil
		}
	}

	// 获取该stage的所有房间记录
	var stageRooms []dao.Room
	for _, room := range rooms {
		if room.Stage == targetStage && room.IsStageOver {
			stageRooms = append(stageRooms, room)
		}
	}

	// 检查是否有两个玩家的记录
	if len(stageRooms) != 2 {
		task.Response.BaseResponse.RetCode = 1005
		task.Response.BaseResponse.Message = "Incomplete stage data, requires two players"
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
		Stage:  int(targetStage),
		Player1: services.Player{
			Address:    strings.ToLower(player1.Address),
			HP:         player1.PlayerHP,
			Multiplier: player1.Multiplier,
			IsMyself:   strings.ToLower(player1.Address) == address,
		},
		Player2: services.Player{
			Address:    strings.ToLower(player2.Address),
			HP:         player2.PlayerHP,
			Multiplier: player2.Multiplier,
			IsMyself:   strings.ToLower(player2.Address) == address,
		},
		Actions:    []services.BattleAction{}, // 空数组，详细结果可通过日志查看
		IsGameOver: targetStage == 10,         // stage 10表示游戏结束
		GameResult: "",                        // 将在下面设置
	}

	// 如果游戏结束，设置游戏结果和赢家倍率
	if targetStage == 10 {
		// 设置游戏结果
		if player1.PlayerHP <= 0 {
			// 玩家1血量为0，玩家2获胜
			if strings.ToLower(player1.Address) == address {
				battleInfo.GameResult = "lose" // 当前用户失败
			} else {
				battleInfo.GameResult = "win" // 当前用户获胜
			}
		} else if player2.PlayerHP <= 0 {
			// 玩家2血量为0，玩家1获胜
			if strings.ToLower(player1.Address) == address {
				battleInfo.GameResult = "win" // 当前用户获胜
			} else {
				battleInfo.GameResult = "lose" // 当前用户失败
			}
		} else {
			// 血量比较（stage 3的情况）
			if player1.PlayerHP > player2.PlayerHP {
				if strings.ToLower(player1.Address) == address {
					battleInfo.GameResult = "win" // 当前用户获胜
				} else {
					battleInfo.GameResult = "lose" // 当前用户失败
				}
			} else if player2.PlayerHP > player1.PlayerHP {
				if strings.ToLower(player1.Address) == address {
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
		// 非stage 10，WinnerMultiplier字段不使用，因为使用的是双方各自的倍率
		battleInfo.WinnerMultiplier = 0
	}

	// 构建详细的对战过程数据
	battleInfo.Actions = task.buildBattleDetails(task.Request.RoomID, targetStage, player1, player2)

	task.Response.BattleInfo = battleInfo
	task.Response.BaseResponse.RetCode = 0
	task.Response.BaseResponse.Message = "Battle info retrieved successfully"

	log.Infof("%s, battle info retrieved for room %s, stage %d", task.Request.RequestUUID, task.Request.RoomID, targetStage)
	return task.Response, nil
}

// buildBattleDetails 构建详细的对战过程数据
func (task *GetBattleInfoTask) buildBattleDetails(roomID string, stage uint, player1, player2 *dao.Room) []services.BattleAction {
	// 创建对战模拟器
	simulator := services.NewBattleSimulator()

	// 首先尝试从缓存获取
	if cachedActions, found := simulator.GetDetailedActionsFromCache(roomID, int(stage)); found {
		return cachedActions
	}

	// 缓存未命中，调用 SimulateStage（会自动缓存）
	return task.simulateAndCache(roomID, stage, player1, player2)
}

// simulateAndCache 模拟对战并自动缓存
func (task *GetBattleInfoTask) simulateAndCache(roomID string, stage uint, player1, player2 *dao.Room) []services.BattleAction {
	// 解析玩家卡牌
	player1Cards := task.parseCardsString(player1.Cards)
	player2Cards := task.parseCardsString(player2.Cards)

	// 如果没有卡牌数据，返回空数组
	if len(player1Cards) == 0 || len(player2Cards) == 0 {
		log.Warnf("No card data found for room %s, stage %d", roomID, stage)
		return []services.BattleAction{}
	}

	// 获取上一stage的血量作为初始血量
	var player1InitialHP, player2InitialHP int

	// 获取上一stage的血量
	prevStageRooms, err := db.GetRoomsByStage(roomID, stage-1)
	if err != nil {
		log.Errorf("Failed to get previous stage %d records for room %s: %v", stage-1, roomID, err)
		return []services.BattleAction{}
	}

	if len(prevStageRooms) != 2 {
		log.Errorf("Previous stage %d has %d players, expected 2 for room %s", stage-1, len(prevStageRooms), roomID)
		return []services.BattleAction{}
	}

	// 获取上一stage的血量
	for _, room := range prevStageRooms {
		if strings.ToLower(room.Address) == strings.ToLower(player1.Address) {
			player1InitialHP = room.PlayerHP
		} else if strings.ToLower(room.Address) == strings.ToLower(player2.Address) {
			player2InitialHP = room.PlayerHP
		}
	}

	// 创建对战输入
	input := &services.StageBattleInput{
		Player1Address:    strings.ToLower(player1.Address),
		Player2Address:    strings.ToLower(player2.Address),
		Player1HP:         player1InitialHP,
		Player2HP:         player2InitialHP,
		Player1Multiplier: player1.Multiplier,
		Player2Multiplier: player2.Multiplier,
		Player1Cards:      player1Cards,
		Player2Cards:      player2Cards,
	}

	// 调用 SimulateStage（会自动缓存详细动作）
	simulator := services.NewBattleSimulator()
	result, err := simulator.SimulateStage(input, int(stage))
	if err != nil {
		log.Errorf("Failed to simulate battle: %v", err)
		return []services.BattleAction{}
	}

	return result.DetailedActions
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
}
