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
	Register(SYNC_GAME_PHASE_LABEL, NewSyncGamePhaseTask, COOKIEAUTH)
}

type SyncGamePhaseRequest struct {
	BaseRequest
	TempAddress string `mapstructure:"TempAddress" validate:"required"`
	PlayerID    string `mapstructure:"PlayerID" validate:"required"`
}

type SyncGamePhaseResponse struct {
	BaseResponse
}

type SyncGamePhaseTask struct {
	Request  *SyncGamePhaseRequest
	Response *SyncGamePhaseResponse
}

func NewSyncGamePhaseRequest(data *map[string]interface{}) (*SyncGamePhaseRequest, error) {
	req := &SyncGamePhaseRequest{}
	if err := mapstructure.Decode(*data, &req); err != nil {
		return nil, err
	}
	req.BaseRequest.RequestUUID = (*data)["RequestUUID"].(string)
	return req, nil
}

func NewSyncGamePhaseResponse(sessionID string) *SyncGamePhaseResponse {
	return &SyncGamePhaseResponse{
		BaseResponse: BaseResponse{
			Action:      SYNC_GAME_PHASE_LABEL + "Response",
			RequestUUID: sessionID,
		},
	}
}

func NewSyncGamePhaseTask(data *map[string]interface{}) (Task, error) {
	req, err := NewSyncGamePhaseRequest(data)
	if err != nil {
		return nil, err
	}
	task := &SyncGamePhaseTask{
		Request:  req,
		Response: NewSyncGamePhaseResponse(req.BaseRequest.RequestUUID),
	}
	if err := validator.New().Struct(task.Request); err != nil {
		return nil, err
	}
	return task, nil
}

func (task *SyncGamePhaseTask) Run(c *gin.Context) (Response, error) {
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

	rpcClient := client.GetGlobalRpcClient()
	if rpcClient == nil {
		task.Response.BaseResponse.RetCode = 1002
		task.Response.BaseResponse.Message = "gRPC client not initialized"
		return task.Response, nil
	}

	req := &proto.PlayerAddress{
		Id:               playerID,
		TemporaryAddress: strings.ToLower(task.Request.TempAddress),
	}
	_, err = rpcClient.SyncGamePhase(context.Background(), req)
	if err != nil {
		task.Response.BaseResponse.RetCode = 1002
		task.Response.BaseResponse.Message = "RoomServer SyncGamePhase failed: " + err.Error()
		return task.Response, nil
	}

	task.Response.BaseResponse.RetCode = 0
	task.Response.BaseResponse.Message = "Successfully synced game phase"
	return task.Response, nil
}
