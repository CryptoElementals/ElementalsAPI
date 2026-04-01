package api

import (
	"context"
	"strconv"
	"strings"

	"github.com/CryptoElementals/common/config"
	"github.com/CryptoElementals/common/db"
	"github.com/CryptoElementals/common/rpc/client"
	"github.com/CryptoElementals/common/rpc/proto"
	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
	"github.com/mitchellh/mapstructure"
)

func init() {
	Register(GET_GAME_PHASE_LABEL, NewGetGamePhaseTask, COOKIEAUTH)
}

// GetGamePhaseRequest 请求结构体
type GetGamePhaseRequest struct {
	BaseRequest
	TempAddress string `mapstructure:"TempAddress" validate:"required"` // 临时地址
	PlayerID    string `mapstructure:"PlayerID" validate:"required"`
}

// PvPInfo PvP对战信息
type PvPInfo struct {
	Phase           uint32 `json:"Phase"`           // None, Confirming, InBattle, WaitingContinue: 0123
	GameID          uint32 `json:"GameID"`          // 游戏ID
	RoundNumber     uint32 `json:"RoundNumber"`     // 回合数
	TurnNumber      uint32 `json:"TurnNumber"`      // 回合内的轮次
	TurnStartAt     int64  `json:"TurnStartAt"`     // 轮次开始时间
	TimeoutDuration uint64 `json:"TimeoutDuration"` // 超时时间
}

// GetGamePhaseResponse 响应结构体
type GetGamePhaseResponse struct {
	BaseResponse
	Mode    uint32        `json:"Mode"`              // 0:None, 1:PvP
	PvPInfo *PvPInfo      `json:"PvPInfo"`           // PvP对战信息
	Players []MatchPlayer `json:"Players,omitempty"` // 新增，集成对战玩家信息
}

type GetGamePhaseTask struct {
	Request  *GetGamePhaseRequest
	Response *GetGamePhaseResponse
}

// 解码请求
func NewGetGamePhaseRequest(data *map[string]interface{}) (*GetGamePhaseRequest, error) {
	req := &GetGamePhaseRequest{}
	err := mapstructure.Decode(*data, &req)
	if err != nil {
		return nil, err
	}
	req.BaseRequest.RequestUUID = (*data)["RequestUUID"].(string)
	return req, nil
}

func NewGetGamePhaseResponse(sessionId string) *GetGamePhaseResponse {
	return &GetGamePhaseResponse{
		BaseResponse: BaseResponse{
			Action:      GET_GAME_PHASE_LABEL + "Response",
			RequestUUID: sessionId,
		},
		PvPInfo: &PvPInfo{
			Phase: 0,
		},
	}
}

func NewGetGamePhaseTask(data *map[string]interface{}) (Task, error) {
	req, err := NewGetGamePhaseRequest(data)
	if err != nil {
		return nil, err
	}
	task := &GetGamePhaseTask{
		Request:  req,
		Response: NewGetGamePhaseResponse(req.BaseRequest.RequestUUID),
	}

	validate := validator.New()
	err = validate.Struct(task.Request)
	if err != nil {
		return nil, err
	}

	return task, nil
}

func (task *GetGamePhaseTask) Run(c *gin.Context) (Response, error) {
	// 解析 PlayerID（由中间件从会话中注入），前端只需要传临时地址
	playerIDStr := strings.TrimSpace(task.Request.PlayerID)
	if playerIDStr == "" {
		task.Response.BaseResponse.RetCode = 1001
		task.Response.BaseResponse.Message = "player id is empty"
		return task.Response, nil
	}
	playerID, err := strconv.ParseInt(playerIDStr, 10, 64)
	if err != nil {
		task.Response.BaseResponse.RetCode = 1001
		task.Response.BaseResponse.Message = "invalid player id"
		return task.Response, nil
	}
	tempAddress := strings.ToLower(task.Request.TempAddress)

	// 通过gRPC调用RoomServer的GetPlayerInfo
	rpcClient := client.GetGlobalRpcClient()
	if rpcClient == nil {
		task.Response.BaseResponse.RetCode = 1002
		task.Response.BaseResponse.Message = "gRPC client not initialized"
		return task.Response, nil
	}

	playerAddr := &proto.PlayerAddress{
		Id:               playerID,
		TemporaryAddress: tempAddress,
	}

	gamePhase, err := rpcClient.GetGamePhase(context.Background(), playerAddr)
	if err != nil {
		task.Response.BaseResponse.RetCode = 1003
		task.Response.BaseResponse.Message = "RoomServer GetPlayerInfo failed: " + err.Error()
		return task.Response, nil
	}

	// Populate PvPInfo from GamePhase
	task.Response.Mode = uint32(gamePhase.GameType)
	task.Response.PvPInfo.GameID = gamePhase.GameID
	task.Response.PvPInfo.RoundNumber = gamePhase.RoundNumber
	task.Response.PvPInfo.TurnNumber = gamePhase.TurnNumber
	task.Response.PvPInfo.TurnStartAt = gamePhase.TurnStartAt
	task.Response.PvPInfo.TimeoutDuration = uint64(config.GameParams.GameContinueTimeout)

	// If GameID == 0 and no players, check queue on lobby
	if gamePhase.GameID == 0 && len(gamePhase.Players) == 0 {
		lobbyClient := client.GetGlobalLobbyClient()
		if lobbyClient == nil {
			task.Response.PvPInfo.Phase = 0
			task.Response.BaseResponse.Message = "gRPC lobby client not initialized"
			task.Response.BaseResponse.RetCode = 1002
			return task.Response, nil
		}
		inQueueResp, err := lobbyClient.IsPlayerInQueue(context.Background(), playerAddr)
		if err == nil && inQueueResp != nil && inQueueResp.IsInQueue {
			task.Response.PvPInfo.Phase = 1
			task.Response.BaseResponse.Message = "Player is in match queue"
		} else {
			task.Response.PvPInfo.Phase = 0
			task.Response.BaseResponse.Message = "Player is not participating in any game"
		}
	} else if gamePhase.GameID == 0 && len(gamePhase.Players) > 0 {
		// GameID == 0 but has players means waiting for continue
		task.Response.PvPInfo.Phase = 4
		task.Response.BaseResponse.Message = "Player is waiting for continue"
	} else if gamePhase.GameID != 0 {
		// GameID != 0 means game has started
		// Check if all players are ready (TurnStatus == PLAYER_TURN_READY) to determine if matched or in game
		allReady := true
		for _, p := range gamePhase.Players {
			if p.TurnStatus != proto.PlayerTurnStatus_PLAYER_TURN_READY &&
				p.TurnStatus != proto.PlayerTurnStatus_PLAYER_TURN_COMMITMENT_SUBMITTED &&
				p.TurnStatus != proto.PlayerTurnStatus_PLAYER_TURN_CARD_SUBMITTED {
				allReady = false
				break
			}
		}
		if allReady && len(gamePhase.Players) > 0 {
			// All players ready but game just started - likely matched
			task.Response.PvPInfo.Phase = 2
			task.Response.BaseResponse.Message = "Player matched, waiting for confirmation"
		} else {
			// Game is in progress
			task.Response.PvPInfo.Phase = 3
			task.Response.BaseResponse.Message = "Player has entered battle"
		}
	} else {
		task.Response.PvPInfo.Phase = 0
		task.Response.BaseResponse.Message = "Player is not participating in any game"
	}

	// 补充玩家信息
	if len(gamePhase.Players) > 0 {
		players := make([]MatchPlayer, 0, len(gamePhase.Players))
		for _, p := range gamePhase.Players {
			uidStr := strconv.FormatInt(p.Address.Id, 10)
			userProfile, err := db.GetUserProfileByPlayerID(uidStr)
			if err != nil || userProfile == nil {
				continue
			}

			// Determine if player is confirmed based on TurnStatus
			isConfirmed := p.TurnStatus == proto.PlayerTurnStatus_PLAYER_TURN_READY ||
				p.TurnStatus == proto.PlayerTurnStatus_PLAYER_TURN_COMMITMENT_SUBMITTED ||
				p.TurnStatus == proto.PlayerTurnStatus_PLAYER_TURN_CARD_SUBMITTED

			// Extract cards from Card field if available
			var cards []uint32
			if p.Card != nil {
				cards = []uint32{*p.Card}
			}

			players = append(players, MatchPlayer{
				Address:           userProfile.Address,
				IsMyself:          p.Address.TemporaryAddress == tempAddress && p.Address.Id == playerID,
				IsConfirmed:       isConfirmed,
				Cards:             cards,
				Name:              userProfile.Name,
				AvatarURL:         userProfile.AvatarURL,
				CurrentHP:         int32(p.CurrentHP),
				CurrentMultiplier: int32(p.CurrentMultiplier),
				InitialHP:         int32(config.GameParams.InitialHP),
				MaxHPOneLine:      int32(config.GameParams.InitialHP),
				InitialMultipler:  int32(config.GameParams.InitialMultiplier),
			})
		}
		task.Response.Players = players
	}

	task.Response.BaseResponse.RetCode = 0
	return task.Response, nil
}

type MatchPlayer struct {
	Address           string   `json:"Address"`
	Name              string   `json:"Name"`
	AvatarURL         string   `json:"AvatarURL"`
	IsMyself          bool     `json:"IsMyself"`
	IsConfirmed       bool     `json:"IsConfirmed"`
	Cards             []uint32 `json:"Cards"`
	CurrentHP         int32    `json:"CurrentHP"`
	CurrentMultiplier int32    `json:"CurrentMultiplier"`
	InitialHP         int32    `json:"InitialHP"`
	MaxHPOneLine      int32    `json:"MaxHPOneLine"`
	InitialMultipler  int32    `json:"InitialMultipler"`
}
