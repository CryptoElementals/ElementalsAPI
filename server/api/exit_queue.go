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
	Register(EXIT_QUEUE_LABEL, NewExitQueueTask, COOKIEAUTH)
}

// ExitQueueRequest 请求结构体
type ExitQueueRequest struct {
	BaseRequest
	Mode        string `mapstructure:"Mode" validate:"required"`
	TempAddress string `mapstructure:"TempAddress" validate:"required"`
	PlayerID    string `mapstructure:"PlayerID" validate:"required"`
}

// ExitQueueResponse 响应结构体
type ExitQueueResponse struct {
	BaseResponse
}

type ExitQueueTask struct {
	Request  *ExitQueueRequest
	Response *ExitQueueResponse
}

// 解码请求
func NewExitQueueRequest(data *map[string]interface{}) (*ExitQueueRequest, error) {
	req := &ExitQueueRequest{}
	err := mapstructure.Decode(*data, &req)
	if err != nil {
		return nil, err
	}
	req.BaseRequest.RequestUUID = (*data)["RequestUUID"].(string)
	return req, nil
}

func NewExitQueueResponse(sessionId string) *ExitQueueResponse {
	return &ExitQueueResponse{
		BaseResponse: BaseResponse{
			Action:      EXIT_QUEUE_LABEL + "Response",
			RequestUUID: sessionId,
		},
	}
}

func NewExitQueueTask(data *map[string]interface{}) (Task, error) {
	req, err := NewExitQueueRequest(data)
	if err != nil {
		return nil, err
	}
	task := &ExitQueueTask{
		Request:  req,
		Response: NewExitQueueResponse(req.BaseRequest.RequestUUID),
	}

	validate := validator.New()
	err = validate.Struct(task.Request)
	if err != nil {
		return nil, err
	}

	return task, nil
}

func (task *ExitQueueTask) Run(c *gin.Context) (Response, error) {
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

	// 	// 验证游戏模式
	validModes := []string{"PvP", "Tournament"}
	modeValid := false
	for _, validMode := range validModes {
		if task.Request.Mode == validMode {
			modeValid = true
			break
		}
	}
	if !modeValid {
		task.Response.BaseResponse.RetCode = 1005
		task.Response.BaseResponse.Message = "Invalid game mode. Only PvP and Tournament are supported"
		return task.Response, nil
	}

	// 通过gRPC调用RoomServer的ExitQueue
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

	_, err = rpcClient.ExitQueue(context.Background(), playerAddr)
	if err != nil {
		task.Response.BaseResponse.RetCode = 1002
		task.Response.BaseResponse.Message = "ExitQueue failed. Internal error: " + ShortGRPCError(err)
		return task.Response, nil
	}

	task.Response.BaseResponse.RetCode = 0
	task.Response.BaseResponse.Message = "Successfully left match queue"

	return task.Response, nil
}
