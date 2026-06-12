package api

import (
	"context"
	"encoding/hex"
	"strconv"
	"strings"

	"github.com/CryptoElementals/common/rpc/client"
	"github.com/CryptoElementals/common/rpc/proto"
	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
	"github.com/mitchellh/mapstructure"
)

func init() {
	Register(GET_GAME_PHASE_LABEL, NewGetGamePhaseTask, COOKIEAUTH)
}

type GetGamePhaseRequest struct {
	BaseRequest
	TempAddress string `mapstructure:"TempAddress" validate:"required"`
	PlayerID    string `mapstructure:"PlayerID" validate:"required"`
}

type GetGamePhaseResponse struct {
	BaseResponse
	GamePhase *GamePhaseVO `json:"GamePhase,omitempty"`
}

type TurnCardPlayingInfoVO struct {
	TurnNumber uint32 `json:"TurnNumber,omitempty"`
	Commitment string `json:"Commitment,omitempty"`
	Card       uint32 `json:"Card,omitempty"`
}

type GamePhasePlayerVO struct {
	Address              *proto.PlayerAddress       `json:"Address,omitempty"`
	CurrentHP            uint32                     `json:"CurrentHP,omitempty"`
	TurnCardPlayingInfos []*TurnCardPlayingInfoVO   `json:"TurnCardPlayingInfos,omitempty"`
}

type GamePhaseVO struct {
	GameType         proto.GameType         `json:"GameType,omitempty"`
	GameID           int64                  `json:"GameID,omitempty"`
	RoundNumber      uint32                 `json:"RoundNumber,omitempty"`
	TurnNumber       uint32                 `json:"TurnNumber,omitempty"`
	TurnStartAt      int64                  `json:"TurnStartAt,omitempty"`
	TurnStatus       proto.TurnStatus       `json:"TurnStatus"`
	Timeout          int64                  `json:"Timeout,omitempty"`
	PlayerTurnStatus proto.PlayerTurnStatus `json:"PlayerTurnStatus"`
	Players          []*GamePhasePlayerVO   `json:"Players,omitempty"`
}

func toCommitmentHex(commitment []byte) string {
	if len(commitment) == 0 {
		return ""
	}
	return "0x" + hex.EncodeToString(commitment)
}

func toTurnCardPlayingInfosVO(infos []*proto.TurnCardPlayingInfo) []*TurnCardPlayingInfoVO {
	if len(infos) == 0 {
		return nil
	}
	out := make([]*TurnCardPlayingInfoVO, 0, len(infos))
	for _, info := range infos {
		if info == nil {
			continue
		}
		out = append(out, &TurnCardPlayingInfoVO{
			TurnNumber: info.GetTurnNumber(),
			Commitment: toCommitmentHex(info.GetCommitment()),
			Card:       info.GetCard(),
		})
	}
	return out
}

func toGamePhasePlayersVO(players []*proto.GamePhasePlayer) []*GamePhasePlayerVO {
	if len(players) == 0 {
		return nil
	}
	out := make([]*GamePhasePlayerVO, 0, len(players))
	for _, player := range players {
		if player == nil {
			continue
		}
		out = append(out, &GamePhasePlayerVO{
			Address:              player.GetAddress(),
			CurrentHP:            player.GetCurrentHP(),
			TurnCardPlayingInfos: toTurnCardPlayingInfosVO(player.GetTurnCardPlayingInfos()),
		})
	}
	return out
}

func toGamePhaseVO(gamePhase *proto.GamePhase) *GamePhaseVO {
	if gamePhase == nil {
		return nil
	}
	return &GamePhaseVO{
		GameType:         gamePhase.GetGameType(),
		GameID:           gamePhase.GetGameID(),
		RoundNumber:      gamePhase.GetRoundNumber(),
		TurnNumber:       gamePhase.GetTurnNumber(),
		TurnStartAt:      gamePhase.GetTurnStartAt(),
		TurnStatus:       gamePhase.GetTurnStatus(),
		Timeout:          gamePhase.GetTimeout(),
		PlayerTurnStatus: gamePhase.GetPlayerTurnStatus(),
		Players:          toGamePhasePlayersVO(gamePhase.GetPlayers()),
	}
}

type GetGamePhaseTask struct {
	Request  *GetGamePhaseRequest
	Response *GetGamePhaseResponse
}

func NewGetGamePhaseRequest(data *map[string]interface{}) (*GetGamePhaseRequest, error) {
	req := &GetGamePhaseRequest{}
	if err := mapstructure.Decode(*data, &req); err != nil {
		return nil, err
	}
	req.BaseRequest.RequestUUID = (*data)["RequestUUID"].(string)
	return req, nil
}

func NewGetGamePhaseResponse(sessionID string) *GetGamePhaseResponse {
	return &GetGamePhaseResponse{
		BaseResponse: BaseResponse{
			Action:      GET_GAME_PHASE_LABEL + "Response",
			RequestUUID: sessionID,
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
	if err := validator.New().Struct(task.Request); err != nil {
		return nil, err
	}
	return task, nil
}

func (task *GetGamePhaseTask) Run(c *gin.Context) (Response, error) {
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

	rpcClient := client.RoomClientForType(ServerTypeFromGin(c))
	if rpcClient == nil {
		task.Response.BaseResponse.RetCode = 1002
		task.Response.BaseResponse.Message = "gRPC client not initialized"
		return task.Response, nil
	}

	req := &proto.PlayerAddress{
		Id:               playerID,
		TemporaryAddress: strings.ToLower(task.Request.TempAddress),
	}
	gp, err := rpcClient.GetGamePhase(context.Background(), req)
	if err != nil {
		task.Response.BaseResponse.RetCode = 1002
		task.Response.BaseResponse.Message = "RoomServer GetGamePhase failed: " + err.Error()
		return task.Response, nil
	}

	task.Response.GamePhase = toGamePhaseVO(gp)
	task.Response.BaseResponse.RetCode = 0
	task.Response.BaseResponse.Message = "ok"
	return task.Response, nil
}
