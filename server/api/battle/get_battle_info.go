package battle

import (
	"context"
	"strconv"
	"strings"

	"github.com/CryptoElementals/common/rpc/proto"
	"github.com/CryptoElementals/common/server/api"
	"github.com/CryptoElementals/common/server/services/battle"
	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
	"github.com/mitchellh/mapstructure"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
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
	conn, err := grpc.NewClient(roomServerAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
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
	gameInfoReq := &proto.GetGameInfoRequest{GameId: uint32(roomIdUint)}
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

// RegisterBattleApis 注册对战相关API
func RegisterBattleApis() {
	api.Register(GET_BATTLE_INFO_LABEL, NewGetBattleInfoTask, api.COOKIEAUTH)
	api.Register(SSE_EXAMPLE_LABEL, NewSSEExampleTask, api.NOAUTH)
	api.Register(SUBSCRIBE_GAME_INFO_LABEL, NewSubscribeGameInfoTask, api.COOKIEAUTH)
}
