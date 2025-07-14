package battle

import (
	"fmt"
	"strings"

	"github.com/CryptoElementals/common/db"
	"github.com/CryptoElementals/common/log"
	dao "github.com/CryptoElementals/common/models"
	"github.com/CryptoElementals/common/server/api"
	"github.com/CryptoElementals/common/server/services/battle"
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
	StageResult *battle.BattleResult `json:"StageResult"`
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

	// 构建阶段对战结果
	stageResult := task.buildStageResult(task.Request.RoomID, targetStage, player1, player2, address)

	task.Response.StageResult = stageResult
	task.Response.BaseResponse.RetCode = 0
	task.Response.BaseResponse.Message = "Stage battle info retrieved successfully"

	log.Infof("%s, stage battle info retrieved for room %s, stage %d", task.Request.RequestUUID, task.Request.RoomID, targetStage)
	return task.Response, nil
}

// buildStageResult 构建阶段对战结果
func (task *GetBattleInfoTask) buildStageResult(roomID string, stage uint, player1, player2 *dao.Room, currentAddress string) *battle.BattleResult {
	// 解析玩家卡牌
	player1Cards := task.parseCardsString(player1.Cards)
	player2Cards := task.parseCardsString(player2.Cards)

	// 如果没有卡牌数据，返回空结果
	if len(player1Cards) == 0 || len(player2Cards) == 0 {
		log.Warnf("No card data found for room %s, stage %d", roomID, stage)
		return &battle.BattleResult{
			Player1Address: strings.ToLower(player1.Address),
			Player2Address: strings.ToLower(player2.Address),
			Stage:          int(stage),
			Rounds:         []battle.RoundResult{},
		}
	}

	// 获取上一stage的血量作为初始血量
	var player1InitialHP, player2InitialHP int

	// 获取上一stage的血量
	prevStageRooms, err := db.GetRoomsByStage(roomID, stage-1)
	if err != nil {
		log.Errorf("Failed to get previous stage %d records for room %s: %v", stage-1, roomID, err)
		// 如果获取失败，使用当前血量
		player1InitialHP = player1.PlayerHP
		player2InitialHP = player2.PlayerHP
	} else if len(prevStageRooms) == 2 {
		// 获取上一stage的血量
		for _, room := range prevStageRooms {
			if strings.ToLower(room.Address) == strings.ToLower(player1.Address) {
				player1InitialHP = room.PlayerHP
			} else if strings.ToLower(room.Address) == strings.ToLower(player2.Address) {
				player2InitialHP = room.PlayerHP
			}
		}
	} else {
		// 如果上一stage数据不完整，使用当前血量
		player1InitialHP = player1.PlayerHP
		player2InitialHP = player2.PlayerHP
	}

	// 将卡牌字符串转换为卡牌ID数组
	player1CardIDs := task.convertCardsToIDs(player1Cards)
	player2CardIDs := task.convertCardsToIDs(player2Cards)

	// 创建对战输入
	input := &battle.BattleInput{
		Player1Address:    strings.ToLower(player1.Address),
		Player2Address:    strings.ToLower(player2.Address),
		Player1HP:         player1InitialHP,
		Player2HP:         player2InitialHP,
		Player1Multiplier: player1.Multiplier,
		Player2Multiplier: player2.Multiplier,
		Player1Cards:      player1CardIDs,
		Player2Cards:      player2CardIDs,
	}

	// 调用对战引擎
	engine := battle.NewBattleEngine()
	stageResult, err := engine.ExecuteBattle(input, int(stage))
	if err != nil {
		log.Errorf("Failed to simulate battle: %v", err)
		return &battle.BattleResult{
			Player1Address: strings.ToLower(player1.Address),
			Player2Address: strings.ToLower(player2.Address),
			Stage:          int(stage),
			Rounds:         []battle.RoundResult{},
		}
	}

	// 设置阶段编号
	stageResult.Stage = int(stage)

	return stageResult
}

// parseCardsString 解析卡牌字符串
func (task *GetBattleInfoTask) parseCardsString(cardsStr string) []string {
	if cardsStr == "" {
		return []string{}
	}
	// 假设格式是 "J0|M1|S2"
	return strings.Split(cardsStr, "|")
}

// convertCardsToIDs 将卡牌字符串转换为卡牌ID数组
func (task *GetBattleInfoTask) convertCardsToIDs(cardStrings []string) []int {
	cardIDs := make([]int, 0, len(cardStrings))

	// 简单的卡牌字符串到ID的映射
	cardIDMap := map[string]int{
		"J0": 1, "J1": 2, "J2": 3, "J3": 4, "J4": 5,
		"M0": 6, "M1": 7, "M2": 8, "M3": 9, "M4": 10,
		"S0": 11, "S1": 12, "S2": 13, "S3": 14, "S4": 15,
		"H0": 16, "H1": 17, "H2": 18, "H3": 19, "H4": 20,
		"T0": 21, "T1": 22, "T2": 23, "T3": 24, "T4": 25,
	}

	for _, cardStr := range cardStrings {
		if id, exists := cardIDMap[cardStr]; exists {
			cardIDs = append(cardIDs, id)
		}
	}

	return cardIDs
}

// RegisterBattleApis 注册对战相关API
func RegisterBattleApis() {
	api.Register(GET_BATTLE_INFO_LABEL, NewGetBattleInfoTask, api.COOKIEAUTH)
	api.Register(SSE_EXAMPLE_LABEL, NewSSEExampleTask, api.NOAUTH)
}
