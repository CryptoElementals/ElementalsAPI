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
	Status     int32  `json:"Status"`     // PlayerStatus enum: 0=UNKNOWN, 1=IN_QUEUE, 2=MATCHED, 3=IN_GAME (4 deprecated)
	StatusName string `json:"StatusName"` // UNKNOWN, IN_QUEUE, MATCHED, IN_GAME
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

	// 通过gRPC调用RoomServer的GetPlayerStatus
	rpcClient := client.GetGlobalRpcClient()
	if rpcClient == nil {
		task.Response.BaseResponse.RetCode = 1002
		task.Response.BaseResponse.Message = "gRPC client not initialized"
		return task.Response, nil
	}

	req := &proto.PlayerAddress{
		Id:               playerID,
		TemporaryAddress: tempAddress,
	}

	resp, err := rpcClient.GetPlayerStatus(context.Background(), req)
	if err != nil {
		task.Response.BaseResponse.RetCode = 1002
		task.Response.BaseResponse.Message = "RoomServer GetPlayerStatus failed: " + err.Error()
		return task.Response, nil
	}

	task.Response.Status = int32(resp.Status)
	task.Response.StatusName = getPlayerStatusName(resp.Status)
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
	case proto.PlayerStatus_PLAYER_MATCHED:
		return "MATCHED"
	case proto.PlayerStatus_PLAYER_IN_GAME:
		return "IN_GAME"
	case proto.PlayerStatus_PLAYER_WAITTING_CONTINUE:
		return "WAITTING_CONTINUE" // deprecated proto value; server should not return this for new flows
	default:
		return "UNKNOWN"
	}
}

