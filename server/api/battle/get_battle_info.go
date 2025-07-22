package battle

import (
	"context"
	"strconv"
	"strings"

	"github.com/CryptoElementals/common/room_server/battle"
	"github.com/CryptoElementals/common/rpc/proto"
	"github.com/CryptoElementals/common/server/api"
	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
	"github.com/mitchellh/mapstructure"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

const GET_BATTLE_INFO_LABEL = "GetBattleInfo"

// GetBattleInfoRequest 请求结构体
type GetBattleInfoRequest struct {
	api.BaseRequest
	RoomID string `mapstructure:"RoomId" validate:"required"`
	Round  *uint  `mapstructure:"Round"` // 可选的Round参数，如果指定则查询对应Round的数据
}

// API专用的PlayerRoundStat，包含IsSelf字段
type APIPlayerRoundStat struct {
	PlayerAddress string                  `json:"PlayerAddress"` // 玩家地址
	IsSelf        bool                    `json:"IsSelf"`        // 是否是自己
	CardStats     []battle.PlayerCardStat `json:"CardStats"`     // 每次出牌后的信息
}

// API专用的RoundResult，使用包含IsSelf的PlayerRoundStat
type APIRoundResult struct {
	Players             []APIPlayerRoundStat `json:"Players"`             // 所有玩家的回合数据
	Round               uint                 `json:"Round"`               // Round number
	GameFinalMultiplier uint                 `json:"GameFinalMultiplier"` // Game final multiplier (take loser's multiplier, tie is 1)
	Winner              string               `json:"Winner"`              // Winner address
	IsGameOver          bool                 `json:"IsGameOver"`          // Whether game is over
	GameResultType      string               `json:"GameResultType"`      // Game result type
	Reward              *battle.BattleReward `json:"Reward"`              // Battle reward (token and point)
}

// GetBattleInfoResponse 响应结构体
type GetBattleInfoResponse struct {
	api.BaseResponse
	RoundResult *APIRoundResult `json:"RoundResult"`
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

	// 验证玩家是否是该房间的参与者
	isParticipant := false
	for _, p := range gameInfo.Players {
		if strings.ToLower(p.WalletAddress) == address {
			isParticipant = true
			break
		}
	}
	if !isParticipant {
		task.Response.BaseResponse.RetCode = 1005
		task.Response.BaseResponse.Message = "You are not a participant in this room"
		return task.Response, nil
	}

	// 选定要展示的Round（回合）
	var targetRound uint
	if task.Request.Round != nil {
		targetRound = uint(*task.Request.Round)
	} else {
		// 默认取最大回合号
		for _, round := range gameInfo.Rounds {
			if uint(round.Number) > targetRound {
				targetRound = uint(round.Number)
			}
		}
	}

	// 获取指定回合的数据
	var targetRoundData *proto.Round
	for _, round := range gameInfo.Rounds {
		if uint(round.Number) == targetRound {
			targetRoundData = round
			break
		}
	}

	if targetRoundData == nil {
		task.Response.BaseResponse.RetCode = 1006
		task.Response.BaseResponse.Message = "Target round not found"
		return task.Response, nil
	}

	// 构建API专用的玩家统计数据，包含IsSelf字段
	var apiPlayerStats []APIPlayerRoundStat

	// 为每个玩家构建回合统计数据
	for _, playerRoundInfo := range targetRoundData.PlayerRoundInfos {
		playerAddr := strings.ToLower(playerRoundInfo.PlayerAddress.WalletAddress)

		var cardStats []battle.PlayerCardStat
		for i, card := range playerRoundInfo.SubmittedCards {

			// 将枚举类型 ElementRelation 转换为字符串表示
			var elemRelation string
			switch card.ElementRelation {
			case proto.ElementRelation_OVER_POWER:
				elemRelation = "overpower"
			case proto.ElementRelation_OVER_POWERED:
				elemRelation = "overpowered"
			case proto.ElementRelation_NURTURE:
				elemRelation = "nurture"
			case proto.ElementRelation_NURTURED:
				elemRelation = "nurtured"
			case proto.ElementRelation_TIE:
				elemRelation = "tie"
			default:
				elemRelation = "unknown"
			}

			cardStat := battle.PlayerCardStat{
				CardNumber:       i + 1,
				CardID:           int(card.SubmittedCardId),
				HPBefore:         int(card.PlayerHealthBefore),
				HPAfter:          int(card.PlayerHealthEnd),
				MultiplierBefore: card.MultiplierBefore,
				MultiplierAfter:  card.MultiplierAfter,
				ElementRelation:  elemRelation,
			}
			cardStats = append(cardStats, cardStat)
		}

		// 判断是否为当前请求者
		isSelf := playerAddr == address

		apiPlayerStat := APIPlayerRoundStat{
			PlayerAddress: playerAddr,
			CardStats:     cardStats,
			IsSelf:        isSelf,
		}
		apiPlayerStats = append(apiPlayerStats, apiPlayerStat)
	}

	// 确定游戏是否结束以及胜负结果
	var isGameOver bool
	var winner string
	var gameResultType string
	var gameFinalMultiplier uint
	var reward *battle.BattleReward

	if gameInfo.Status == proto.GameStatus_GAME_END && gameInfo.Result != nil {
		isGameOver = true

		// 解析游戏结果类型
		switch gameInfo.Result.GameResultType {
		case proto.GameResultType_GAME_NORMAL:
			gameResultType = "normal"
		case proto.GameResultType_GAME_KO:
			gameResultType = "ko"
		case proto.GameResultType_GAME_TIE:
			gameResultType = "tie"
		}

		// 最终倍率
		gameFinalMultiplier = uint(gameInfo.Result.Multiplier)

		// 转换奖励数据
		if gameInfo.Result.Reward != nil {
			var playerRewards []battle.PlayerReward
			for _, pr := range gameInfo.Result.Reward.PlayerRewards {
				playerRewards = append(playerRewards, battle.PlayerReward{
					PlayerAddress: strings.ToLower(pr.WalletAddress),
					TokenChange:   int(pr.TokenChange),
					PointChange:   int(pr.PointChange),
				})
			}
			reward = &battle.BattleReward{
				PlayerRewards: playerRewards,
				SystemFee:     int(gameInfo.Result.Reward.SystemFee),
			}

			// 计算赢家：TokenChange 为正值者视为赢家
			if gameResultType != "tie" {
				maxToken := -1 << 31
				for _, pr := range reward.PlayerRewards {
					if pr.TokenChange > maxToken {
						maxToken = pr.TokenChange
						winner = pr.PlayerAddress
					}
				}
				if winner == "" {
					winner = "tie"
				}
			} else {
				winner = "tie"
				if gameFinalMultiplier == 0 {
					gameFinalMultiplier = 1
				}
			}
		}
	}

	// 构建API专用的RoundResult
	roundResult := &APIRoundResult{
		Players:             apiPlayerStats,
		Round:               uint(targetRound),
		GameFinalMultiplier: gameFinalMultiplier,
		Winner:              winner,
		IsGameOver:          isGameOver,
		GameResultType:      gameResultType,
		Reward:              reward,
	}

	task.Response.RoundResult = roundResult
	task.Response.BaseResponse.RetCode = 0
	task.Response.BaseResponse.Message = "Round battle info retrieved successfully"
	return task.Response, nil
}

// RegisterBattleApis 注册对战相关API
func RegisterBattleApis() {
	api.Register(GET_BATTLE_INFO_LABEL, NewGetBattleInfoTask, api.COOKIEAUTH)
	api.Register(SSE_EXAMPLE_LABEL, NewSSEExampleTask, api.NOAUTH)
	api.Register(SUBSCRIBE_GAME_INFO_LABEL, NewSubscribeGameInfoTask, api.COOKIEAUTH)
}
