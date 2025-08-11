package api

import (
	"context"
	"fmt"
	"strings"

	"github.com/CryptoElementals/common/rpc/client"
	"github.com/CryptoElementals/common/rpc/proto"
	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
	"github.com/mitchellh/mapstructure"
)

func init() {
	Register(IS_PLAYER_IN_QUEUE_LABEL, NewIsPlayerInQueueTask, COOKIEAUTH)
}

// IsPlayerInQueueRequest 请求结构体
type IsPlayerInQueueRequest struct {
	BaseRequest
	TempAddress string `mapstructure:"TempAddress" validate:"required"` // 临时地址
	Address     string `mapstructure:"Address"`
}

// IsPlayerInQueueResponse 响应结构体
type IsPlayerInQueueResponse struct {
	BaseResponse
	IsInQueue bool `json:"IsInQueue"` // 是否在队列中
}

type IsPlayerInQueueTask struct {
	Request  *IsPlayerInQueueRequest
	Response *IsPlayerInQueueResponse
}

// 解码请求
func NewIsPlayerInQueueRequest(data *map[string]interface{}) (*IsPlayerInQueueRequest, error) {
	req := &IsPlayerInQueueRequest{}
	err := mapstructure.Decode(*data, &req)
	if err != nil {
		return nil, err
	}
	req.BaseRequest.RequestUUID = (*data)["RequestUUID"].(string)
	return req, nil
}

func NewIsPlayerInQueueResponse(sessionId string) *IsPlayerInQueueResponse {
	return &IsPlayerInQueueResponse{
		BaseResponse: BaseResponse{
			Action:      IS_PLAYER_IN_QUEUE_LABEL + "Response",
			RequestUUID: sessionId,
		},
	}
}

func NewIsPlayerInQueueTask(data *map[string]interface{}) (Task, error) {
	req, err := NewIsPlayerInQueueRequest(data)
	if err != nil {
		return nil, err
	}
	task := &IsPlayerInQueueTask{
		Request:  req,
		Response: NewIsPlayerInQueueResponse(req.BaseRequest.RequestUUID),
	}

	validate := validator.New()
	err = validate.Struct(task.Request)
	if err != nil {
		return nil, err
	}

	return task, nil
}

// Run 执行任务
func (task *IsPlayerInQueueTask) Run(c *gin.Context) (Response, error) {
	// 获取玩家地址（从认证中间件填充到请求结构）
	address := task.Request.Address
	if address == "" {
		return nil, fmt.Errorf("failed to get player address")
	}

	// 将地址转换为小写，确保与数据库中存储的格式一致
	address = strings.ToLower(address)
	tempAddress := strings.ToLower(task.Request.TempAddress)

	// 通过gRPC调用RoomServer的IsPlayerInQueue
	rpcClient := client.GetGlobalRpcClient()
	if rpcClient == nil {
		task.Response.BaseResponse.RetCode = 1002
		task.Response.BaseResponse.Message = "gRPC client not initialized"
		return task.Response, nil
	}

	req := &proto.PlayerAddress{
		WalletAddress:    address,
		TemporaryAddress: tempAddress,
	}

	response, err := rpcClient.IsPlayerInQueue(context.Background(), req)
	if err != nil {
		task.Response.BaseResponse.RetCode = 1003
		task.Response.BaseResponse.Message = "RoomServer IsPlayerInQueue failed: " + err.Error()
		return task.Response, nil
	}

	// 设置响应结果
	task.Response.IsInQueue = response.IsInQueue
	task.Response.BaseResponse.RetCode = 0
	task.Response.BaseResponse.Message = "Successfully checked player queue status"

	return task.Response, nil
}
