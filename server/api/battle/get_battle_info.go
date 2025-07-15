package battle

import (
	"context"
	"strconv"
	"strings"

	"github.com/CryptoElementals/common/db"
	"github.com/CryptoElementals/common/log"
	dao "github.com/CryptoElementals/common/models"
	"github.com/CryptoElementals/common/rpc/proto"
	"github.com/CryptoElementals/common/server/api"
	"github.com/CryptoElementals/common/server/services/battle"
	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
	"github.com/mitchellh/mapstructure"
	"google.golang.org/grpc"
)

const GET_BATTLE_INFO_LABEL = "GetBattleInfo"
const roomServerAddr = "127.0.0.1:50051" // TODO: 替换为实际RoomServer地址

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
	Identity    int                  `json:"Identity"` // 当前请求者身份，1为player1，2为player2
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
	address = strings.ToLower(address)

	// 通过gRPC调用RoomServer的GetGameInfo
	conn, err := grpc.Dial(roomServerAddr, grpc.WithInsecure())
	if err != nil {
		task.Response.BaseResponse.RetCode = 1002
		task.Response.BaseResponse.Message = "Failed to connect to RoomServer: " + err.Error()
		return task.Response, nil
	}
	defer conn.Close()
	client := proto.NewRpcServiceClient(conn)

	roomIdUint, err := strconv.ParseUint(task.Request.RoomID, 10, 32)
	if err != nil {
		task.Response.BaseResponse.RetCode = 1003
		task.Response.BaseResponse.Message = "Invalid RoomID format"
		return task.Response, nil
	}
	gameInfoReq := &proto.GetGameInfoRequest{RoomId: uint32(roomIdUint)}
	gameInfo, err := client.GetGameInfo(context.Background(), gameInfoReq)
	if err != nil {
		task.Response.BaseResponse.RetCode = 1004
		task.Response.BaseResponse.Message = "RoomServer GetGameInfo failed: " + err.Error()
		return task.Response, nil
	}

	// 验证玩家是否是该房间的参与者，并确定身份
	identity := 0
	for i, p := range gameInfo.Players {
		if strings.ToLower(p.WalletAddress) == address {
			identity = i + 1 // 1为player1，2为player2
			break
		}
	}
	if identity == 0 {
		task.Response.BaseResponse.RetCode = 1005
		task.Response.BaseResponse.Message = "You are not a participant in this room"
		return task.Response, nil
	}

	// 选定要展示的stage（回合）
	var targetStage int
	if task.Request.Stage != nil {
		targetStage = int(*task.Request.Stage)
	} else {
		// 默认取最大回合号
		for _, round := range gameInfo.Rounds {
			if int(round.Number) > targetStage {
				targetStage = int(round.Number)
			}
		}
	}

	// 组装 Rounds
	var rounds []battle.RoundResult
	var player1FinalHP, player2FinalHP int
	var player1FinalMultiplier, player2FinalMultiplier float64
	for _, round := range gameInfo.Rounds {
		if int(round.Number) != targetStage {
			continue
		}
		if len(round.Players) != 2 {
			continue
		}
		p1 := round.Players[0]
		p2 := round.Players[1]
		// 只取每个玩家的最后一张卡
		var p1Card, p2Card *proto.RoundSubmittedCard
		if len(p1.Cards) > 0 {
			p1Card = p1.Cards[len(p1.Cards)-1]
		}
		if len(p2.Cards) > 0 {
			p2Card = p2.Cards[len(p2.Cards)-1]
		}
		// 组装回合结果
		rr := battle.RoundResult{
			RoundNumber:    int(round.Number),
			Player1CardID:  int(p1Card.GetSubmittedCardId()),
			Player2CardID:  int(p2Card.GetSubmittedCardId()),
			Player1HPAfter: int(p1Card.GetPlayerHealthEnd()),
			Player2HPAfter: int(p2Card.GetPlayerHealthEnd()),
			// 其他字段可根据需要补充
		}
		// 记录最终血量和倍率
		player1FinalHP = int(p1Card.GetPlayerHealthEnd())
		player2FinalHP = int(p2Card.GetPlayerHealthEnd())
		// proto 里 multiplier 是 uint32，转 float64
		player1FinalMultiplier = float64(p1Card.GetMultiplier())
		player2FinalMultiplier = float64(p2Card.GetMultiplier())
		rounds = append(rounds, rr)
	}

	// 组装 BattleResult
	stageResult := &battle.BattleResult{
		Player1Address:         strings.ToLower(gameInfo.Players[0].WalletAddress),
		Player2Address:         strings.ToLower(gameInfo.Players[1].WalletAddress),
		Stage:                  targetStage,
		Rounds:                 rounds,
		Player1FinalHP:         player1FinalHP,
		Player2FinalHP:         player2FinalHP,
		Player1FinalMultiplier: player1FinalMultiplier,
		Player2FinalMultiplier: player2FinalMultiplier,
	}

	// 结算 Winner、IsGameOver、GameResultType、Reward
	if gameInfo.Status == proto.GameStatus_GAME_END && gameInfo.Result != nil && len(gameInfo.Result.Players) == 2 {
		stageResult.IsGameOver = true
		// 判断胜负
		if gameInfo.Result.Players[0].Status == proto.GameResultPlayerStatus_GAME_RESULT_PLAYER_WIN {
			stageResult.Winner = strings.ToLower(gameInfo.Result.Players[0].Address.WalletAddress)
			stageResult.GameResultType = "win"
		} else if gameInfo.Result.Players[1].Status == proto.GameResultPlayerStatus_GAME_RESULT_PLAYER_WIN {
			stageResult.Winner = strings.ToLower(gameInfo.Result.Players[1].Address.WalletAddress)
			stageResult.GameResultType = "win"
		} else if gameInfo.Result.Players[0].Status == proto.GameResultPlayerStatus_GAME_RESULT_PLAYER_TIE || gameInfo.Result.Players[1].Status == proto.GameResultPlayerStatus_GAME_RESULT_PLAYER_TIE {
			stageResult.GameResultType = "tie"
		}
		// 组装奖励
		stageResult.Reward = &battle.BattleReward{
			Player1TokenChange: int(gameInfo.Result.Players[0].TokenDelta),
			Player2TokenChange: int(gameInfo.Result.Players[1].TokenDelta),
			Player1PointChange: int(gameInfo.Result.Players[0].Points),
			Player2PointChange: int(gameInfo.Result.Players[1].Points),
		}
	}

	task.Response.StageResult = stageResult
	task.Response.Identity = identity
	task.Response.BaseResponse.RetCode = 0
	task.Response.BaseResponse.Message = "Stage battle info retrieved successfully"
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
