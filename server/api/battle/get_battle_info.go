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
	GameFinalMultiplier float64              `json:"GameFinalMultiplier"` // Game final multiplier (take loser's multiplier, tie is 1)
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
	var targetRound int
	if task.Request.Round != nil {
		targetRound = int(*task.Request.Round)
	} else {
		// 默认取最大回合号
		for _, round := range gameInfo.Rounds {
			if int(round.Number) > targetRound {
				targetRound = int(round.Number)
			}
		}
	}

	// 获取指定回合的数据
	var targetRoundData *proto.Round
	for _, round := range gameInfo.Rounds {
		if int(round.Number) == targetRound {
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
	for _, playerRoundInfo := range targetRoundData.Players {
		playerAddr := strings.ToLower(playerRoundInfo.PlayerAddress.WalletAddress)

		var cardStats []battle.PlayerCardStat
		for i, card := range playerRoundInfo.Cards {

			cardStat := battle.PlayerCardStat{
				CardNumber:       i + 1,
				CardID:           int(card.SubmittedCardId),
				HPBefore:         int(card.PlayerHealthBefore),
				HPAfter:          int(card.PlayerHealthEnd),
				MultiplierBefore: card.MultiplierBefore,
				MultiplierAfter:  card.MultiplierAfter,
				ElementRelation:  card.ElementRelation,
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
	var gameFinalMultiplier float64
	var reward *battle.BattleReward

	// 预计算输家地址，便于后面计算倍率
	getLoserAddress := func() string {
		if winner == "" || gameResultType == "tie" {
			return ""
		}
		for _, p := range gameInfo.Players {
			addr := strings.ToLower(p.WalletAddress)
			if addr != winner {
				return addr
			}
		}
		return ""
	}

	if gameInfo.Status == proto.GameStatus_GAME_END && gameInfo.Result != nil {
		isGameOver = true

		// 判断胜负
		if len(gameInfo.Result.Players) >= 2 {
			player1Result := gameInfo.Result.Players[0]
			player2Result := gameInfo.Result.Players[1]

			if player1Result.Status == proto.GameResultPlayerStatus_GAME_RESULT_PLAYER_WIN {
				winner = strings.ToLower(player1Result.Address.WalletAddress)
				gameResultType = "win"
			} else if player2Result.Status == proto.GameResultPlayerStatus_GAME_RESULT_PLAYER_WIN {
				winner = strings.ToLower(player2Result.Address.WalletAddress)
				gameResultType = "win"
			} else if player1Result.Status == proto.GameResultPlayerStatus_GAME_RESULT_PLAYER_TIE ||
				player2Result.Status == proto.GameResultPlayerStatus_GAME_RESULT_PLAYER_TIE {
				winner = "tie"
				gameResultType = "tie"
				gameFinalMultiplier = 1.0
			}

			// 构建奖励信息
			var playerRewards []battle.PlayerReward
			for _, playerResult := range gameInfo.Result.Players {
				playerAddr := strings.ToLower(playerResult.Address.WalletAddress)
				playerReward := battle.PlayerReward{
					PlayerAddress: playerAddr,
					TokenChange:   int(playerResult.TokenDelta),
					PointChange:   int(playerResult.Points),
				}
				playerRewards = append(playerRewards, playerReward)
			}

			reward = &battle.BattleReward{
				PlayerRewards: playerRewards,
				SystemFee:     0, // TODO: 计算系统手续费
			}

			// 计算最终倍率（取输家最后一张牌的 MultiplierAfter，平局为 1）
			if gameResultType == "tie" {
				gameFinalMultiplier = 1
			} else {
				loserAddr := getLoserAddress()
				if loserAddr != "" {
					// 以最后一个 round 为准，遍历找到输家
					if len(gameInfo.Rounds) > 0 {
						lastRound := gameInfo.Rounds[len(gameInfo.Rounds)-1]
						for _, pr := range lastRound.Players {
							if strings.ToLower(pr.PlayerAddress.WalletAddress) == loserAddr {
								if len(pr.Cards) > 0 {
									lastCard := pr.Cards[len(pr.Cards)-1]
									gameFinalMultiplier = lastCard.MultiplierAfter
								}
								break
							}
						}
					}
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
