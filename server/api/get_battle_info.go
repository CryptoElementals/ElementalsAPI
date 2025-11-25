package api

import (
	"context"
	"strconv"
	"strings"

	"github.com/CryptoElementals/common/db"
	"github.com/CryptoElementals/common/log"
	"github.com/CryptoElementals/common/rpc/client"
	"github.com/CryptoElementals/common/rpc/proto"
	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
	"github.com/mitchellh/mapstructure"
)

func init() {
	Register(GET_BATTLE_INFO_LABEL, NewGetBattleInfoTask, COOKIEAUTH)
}

// GetBattleInfoRequest 请求结构体
type GetBattleInfoRequest struct {
	BaseRequest
	GameID      uint32 `mapstructure:"GameID" validate:"required"` // 游戏ID
	Round       uint32 `mapstructure:"Round" validate:"required"`  // 回合号
	PlayerID    string `mapstructure:"PlayerID" validate:"required"`
	TempAddress string `mapstructure:"TempAddress"` // 临时地址
}

// PlayerCardStat 玩家卡牌统计信息
type PlayerCardStat struct {
	CardNumber       int32  `json:"CardNumber"`       // 卡牌序号
	CardID           int32  `json:"CardID"`           // 卡牌ID
	HPBefore         int32  `json:"HPBefore"`         // 使用前血量
	HPAfter          int32  `json:"HPAfter"`          // 使用后血量
	MultiplierBefore int32  `json:"MultiplierBefore"` // 使用前倍率
	MultiplierAfter  int32  `json:"MultiplierAfter"`  // 使用后倍率
	Description      string `json:"Description"`      // 描述
	ElementRelation  int32  `json:"ElementRelation"`  // 元素关系
}

// PlayerRoundStat 玩家回合统计
type PlayerRoundStat struct {
	PlayerAddress string           `json:"PlayerAddress"` // 玩家地址
	IsSelf        bool             `json:"IsSelf"`        // 是否是自己
	CardStats     []PlayerCardStat `json:"CardStats"`     // 卡牌统计
}

// PlayerReward 玩家奖励
type PlayerReward struct {
	PlayerAddress string `json:"PlayerAddress"` // 玩家地址
	TokenChange   int32  `json:"TokenChange"`   // 代币变化
	PointChange   int32  `json:"PointChange"`   // 积分变化
}

// BattleReward 对战奖励
type BattleReward struct {
	PlayerRewards []PlayerReward `json:"PlayerRewards"` // 每个玩家的奖励
	SystemFee     int32          `json:"SystemFee"`     // 系统抽水
}

// GameResult 游戏结果
type GameResult struct {
	Winner              string        `json:"Winner"`              // 获胜者地址
	GameResultType      uint32        `json:"GameResultType"`      // 游戏结果类型 (0:normal, 1:ko, 2:tie)
	GameFinalMultiplier uint32        `json:"GameFinalMultiplier"` // 游戏最终倍率（平局为1）
	Reward              *BattleReward `json:"Reward,omitempty"`    // 对战奖励
	//GameEndAt           uint64        `json:"GameEndAt"`           // 游戏结束时间
	GameContinueTimeout uint64 `json:"GameContinueTimeout"` // 游戏继续超时时间
}

// RoundResult 回合结果
type RoundResult struct {
	Round      uint32            `json:"Round"`      // 回合号
	IsGameOver bool              `json:"IsGameOver"` // 游戏是否结束
	Players    []PlayerRoundStat `json:"Players"`    // 玩家回合统计
}

// GetBattleInfoResponse 响应结构体
type GetBattleInfoResponse struct {
	BaseResponse
	RoundResult *RoundResult `json:"RoundResult"`          // 回合结果
	GameResult  *GameResult  `json:"GameResult,omitempty"` // 游戏结果（仅在游戏结束时包含）
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
		BaseResponse: BaseResponse{
			Action:      GET_BATTLE_INFO_LABEL + "Response",
			RequestUUID: sessionId,
		},
	}
}

func NewGetBattleInfoTask(data *map[string]interface{}) (Task, error) {
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

func (task *GetBattleInfoTask) Run(c *gin.Context) (Response, error) {
	// 通过 PlayerID 解析玩家地址
	profile, err := db.GetUserProfileByPlayerID(strings.TrimSpace(task.Request.PlayerID))
	if err != nil || profile == nil || profile.Address == "" {
		task.Response.BaseResponse.RetCode = 1001
		task.Response.BaseResponse.Message = "Failed to get player address by player id"
		return task.Response, nil
	}
	address := profile.Address

	// 将地址转换为小写，确保与数据库中存储的格式一致
	address = strings.ToLower(address)

	tempAddress := ""
	if len(task.Request.TempAddress) > 0 {
		tempAddress = strings.ToLower(task.Request.TempAddress)
	}

	// 通过gRPC调用RoomServer的GetBattleInfo
	rpcClient := client.GetGlobalRpcClient()
	if rpcClient == nil {
		task.Response.BaseResponse.RetCode = 1002
		task.Response.BaseResponse.Message = "gRPC client not initialized"
		return task.Response, nil
	}

	req := &proto.GetBattleInfoRequest{
		GameID:      task.Request.GameID,
		RoundNumber: task.Request.Round,
	}

	battleInfo, err := rpcClient.GetBattleInfo(context.Background(), req)
	if err != nil {
		task.Response.BaseResponse.RetCode = 1003
		task.Response.BaseResponse.Message = "RoomServer GetBattleInfo failed: " + err.Error()
		return task.Response, nil
	}

	// 转换回合结果
	roundResult := &RoundResult{
		Round:      battleInfo.RoundResult.RoundNumber,
		Players:    make([]PlayerRoundStat, 0, len(battleInfo.RoundResult.Players)),
		IsGameOver: battleInfo.RoundResult.IsGameOver,
	}

	// 转换玩家统计信息
	for _, player := range battleInfo.RoundResult.Players {
		// 通过 player_id 获取用户地址
		playerAddr := ""
		if up, e := db.GetUserProfileByPlayerID(strconv.FormatInt(player.PlayerId, 10)); e == nil && up != nil {
			playerAddr = up.Address
		}
		playerStat := PlayerRoundStat{
			PlayerAddress: playerAddr,
			IsSelf:        player.PlayerId == profile.PlayerID,
			CardStats:     make([]PlayerCardStat, 0, len(player.CardStats)),
		}

		// 转换卡牌信息，去掉了不需要的effects字段
		for _, cardStat := range player.CardStats {
			cardStatInfo := PlayerCardStat{
				CardNumber:       cardStat.CardNumber,
				CardID:           cardStat.CardID,
				HPBefore:         cardStat.HPBefore,
				HPAfter:          cardStat.HPAfter,
				MultiplierBefore: cardStat.MultiplierBefore,
				MultiplierAfter:  cardStat.MultiplierAfter,
				Description:      cardStat.Description,
				ElementRelation:  int32(cardStat.ElementRelation),
			}
			playerStat.CardStats = append(playerStat.CardStats, cardStatInfo)
		}

		roundResult.Players = append(roundResult.Players, playerStat)
	}

	task.Response.RoundResult = roundResult

	// 转换游戏结果（若有）
	if battleInfo.GameResult != nil {
		// Winner 地址解析
		winnerAddr := ""
		if up, e := db.GetUserProfileByPlayerID(strconv.FormatInt(battleInfo.GameResult.WinnerPlayerId, 10)); e == nil && up != nil {
			winnerAddr = up.Address
		}
		gameResult := &GameResult{
			Winner:              winnerAddr,
			GameResultType:      uint32(battleInfo.GameResult.GameResultType),
			GameFinalMultiplier: uint32(battleInfo.GameResult.Multiplier),
		}

		// 转换奖励信息
		if battleInfo.GameResult.Reward != nil {
			reward := &BattleReward{
				PlayerRewards: make([]PlayerReward, 0, len(battleInfo.GameResult.Reward.PlayerRewards)),
				SystemFee:     battleInfo.GameResult.Reward.SystemFee,
			}

			for _, pr := range battleInfo.GameResult.Reward.PlayerRewards {
				addr := ""
				if up, e := db.GetUserProfileByPlayerID(strconv.FormatInt(pr.PlayerId, 10)); e == nil && up != nil {
					addr = up.Address
				}
				reward.PlayerRewards = append(reward.PlayerRewards, PlayerReward{
					PlayerAddress: addr,
					TokenChange:   pr.TokenChange,
					PointChange:   pr.PointChange,
				})
			}

			gameResult.Reward = reward
		}

		if len(tempAddress) > 0 {
			playerAddr := &proto.PlayerAddress{
				Id:               profile.PlayerID,
				TemporaryAddress: tempAddress,
			}

			gamePhase, err := rpcClient.GetGamePhase(context.Background(), playerAddr)
			if err != nil {
				task.Response.BaseResponse.RetCode = 1003
				task.Response.BaseResponse.Message = "RoomServer GetGamePhase in GetBattleInfo failed: " + err.Error()
				return task.Response, nil
			}
			log.Debugf("RequestUUID: %s, gamePhase: %+v", task.Request.BaseRequest.RequestUUID, gamePhase)
		}

		task.Response.GameResult = gameResult
	}

	// 返回成功
	task.Response.BaseResponse.RetCode = 0
	task.Response.BaseResponse.Message = "Successfully retrieved battle info"
	return task.Response, nil
}
