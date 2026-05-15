package api

import (
	"context"
	"strconv"
	"strings"

	"github.com/CryptoElementals/common/rpc/client"
	"github.com/CryptoElementals/common/rpc/proto"
	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
	"github.com/mitchellh/mapstructure"
)

func init() {
	Register(GET_PLAYER_STATUS_LABEL, NewGetPlayerStatusTask, COOKIEAUTH)
}

// GetPlayerStatusRequest 请求结构体
type GetPlayerStatusRequest struct {
	BaseRequest
	TempAddress string `mapstructure:"TempAddress" validate:"required"`
	PlayerID    string `mapstructure:"PlayerID" validate:"required"`
}

// GetPlayerStatusResponse 响应结构体
type GetPlayerStatusResponse struct {
	BaseResponse
	Status         int32  `json:"Status"`                   // PlayerStatus enum (see rpc.PlayerStatus)
	StatusName     string `json:"StatusName"`               // UNKNOWN, IN_QUEUE, MATCHED, IN_GAME, tournament names, etc.
	Since          *int64 `json:"Since,omitempty"`          // Detail: queue join ms, match pending since, or in-game since (Unix ms)
	MatchID        *int64 `json:"MatchID,omitempty"`        // Pending match id when awaiting confirmations
	MatchTimeoutMs *int64 `json:"MatchTimeoutMs,omitempty"` // Confirmation window in ms (MATCHED / pending queue match)
	GameID         *int64 `json:"GameID,omitempty"`         // Active PVP game id when in game
}

type GetPlayerStatusTask struct {
	Request  *GetPlayerStatusRequest
	Response *GetPlayerStatusResponse
}

func NewGetPlayerStatusRequest(data *map[string]interface{}) (*GetPlayerStatusRequest, error) {
	req := &GetPlayerStatusRequest{}
	err := mapstructure.Decode(*data, &req)
	if err != nil {
		return nil, err
	}
	req.BaseRequest.RequestUUID = (*data)["RequestUUID"].(string)
	return req, nil
}

func NewGetPlayerStatusResponse(sessionId string) *GetPlayerStatusResponse {
	return &GetPlayerStatusResponse{
		BaseResponse: BaseResponse{
			Action:      GET_PLAYER_STATUS_LABEL + "Response",
			RequestUUID: sessionId,
		},
	}
}

func NewGetPlayerStatusTask(data *map[string]interface{}) (Task, error) {
	req, err := NewGetPlayerStatusRequest(data)
	if err != nil {
		return nil, err
	}
	task := &GetPlayerStatusTask{
		Request:  req,
		Response: NewGetPlayerStatusResponse(req.BaseRequest.RequestUUID),
	}

	validate := validator.New()
	err = validate.Struct(task.Request)
	if err != nil {
		return nil, err
	}

	return task, nil
}

func (task *GetPlayerStatusTask) Run(c *gin.Context) (Response, error) {
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

	lobbyClient := client.LobbyClientForType(ServerTypeFromGin(c))
	if lobbyClient == nil {
		task.Response.BaseResponse.RetCode = 1002
		task.Response.BaseResponse.Message = "gRPC lobby client not initialized"
		return task.Response, nil
	}

	req := &proto.PlayerAddress{
		Id:               playerID,
		TemporaryAddress: tempAddress,
	}

	resp, err := lobbyClient.GetPlayerStatus(context.Background(), req)
	if err != nil {
		task.Response.BaseResponse.RetCode = 1002
		task.Response.BaseResponse.Message = "Lobby GetPlayerStatus failed: " + err.Error()
		return task.Response, nil
	}

	task.Response.Status = int32(resp.Status)
	task.Response.StatusName = getPlayerStatusName(resp.Status)
	if q := resp.GetInQueue(); q != nil {
		s := q.Since
		task.Response.Since = &s
	} else if m := resp.GetInMatch(); m != nil {
		mid := m.ID
		task.Response.MatchID = &mid
		s := m.Since
		task.Response.Since = &s
		t := m.Timeout
		task.Response.MatchTimeoutMs = &t
	} else if g := resp.GetInGame(); g != nil {
		gid := g.ID
		task.Response.GameID = &gid
		s := g.Since
		task.Response.Since = &s
	}
	task.Response.BaseResponse.RetCode = 0
	task.Response.BaseResponse.Message = "Successfully retrieved player status"
	return task.Response, nil
}

// getPlayerStatusName 将 PlayerStatus 枚举值转换为状态名称
func getPlayerStatusName(status proto.PlayerStatus) string {
	switch status {
	case proto.PlayerStatus_PLAYER_UNKNOWN:
		return "UNKNOWN"
	case proto.PlayerStatus_PLAYER_IN_QUEUE:
		return "IN_QUEUE"
	case proto.PlayerStatus_PLAYER_MATCHED, proto.PlayerStatus_PLAYER_PENDING_QUEUE_MATCH:
		return "MATCHED"
	case proto.PlayerStatus_PLAYER_IN_GAME:
		return "IN_GAME"
	case proto.PlayerStatus_PLAYER_TOURNAMENT_QUEUED:
		return "TOURNAMENT_QUEUED"
	case proto.PlayerStatus_PLAYER_TOURNAMENT_IN_PROGRESS:
		return "TOURNAMENT_IN_PROGRESS"
	default:
		return "UNKNOWN"
	}
}
