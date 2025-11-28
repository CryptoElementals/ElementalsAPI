package api

import (
	"context"

	"github.com/CryptoElementals/common/rpc/client"
	"github.com/gin-gonic/gin"
	"github.com/mitchellh/mapstructure"
	"google.golang.org/protobuf/types/known/emptypb"
)

func init() {
	Register(GET_GAME_TIMEOUT_CONFIG_LABEL, NewGetGameTimeoutConfigTask, NOAUTH)
}

// GetGameTimeoutConfigRequest 请求结构体
type GetGameTimeoutConfigRequest struct {
	BaseRequest
}

// GetGameTimeoutConfigResponse 响应结构体
type GetGameTimeoutConfigResponse struct {
	BaseResponse
	GameMatchTimeout    int64 `json:"GameMatchTimeout"`    // 游戏匹配超时时间（秒）
	RoundConfirmTimeout int64 `json:"RoundConfirmTimeout"` // 回合确认超时时间（秒）
	RoundTimeout        int64 `json:"RoundTimeout"`        // 回合超时时间（秒）
	ContinueTimeout     int64 `json:"ContinueTimeout"`     // 继续游戏超时时间（秒）
}

type GetGameTimeoutConfigTask struct {
	Request  *GetGameTimeoutConfigRequest
	Response *GetGameTimeoutConfigResponse
}

func NewGetGameTimeoutConfigRequest(data *map[string]interface{}) (*GetGameTimeoutConfigRequest, error) {
	req := &GetGameTimeoutConfigRequest{}
	err := mapstructure.Decode(*data, &req)
	if err != nil {
		return nil, err
	}
	req.BaseRequest.RequestUUID = (*data)["RequestUUID"].(string)
	return req, nil
}

func NewGetGameTimeoutConfigResponse(sessionId string) *GetGameTimeoutConfigResponse {
	return &GetGameTimeoutConfigResponse{
		BaseResponse: BaseResponse{
			Action:      GET_GAME_TIMEOUT_CONFIG_LABEL + "Response",
			RequestUUID: sessionId,
		},
	}
}

func NewGetGameTimeoutConfigTask(data *map[string]interface{}) (Task, error) {
	req, err := NewGetGameTimeoutConfigRequest(data)
	if err != nil {
		return nil, err
	}
	task := &GetGameTimeoutConfigTask{
		Request:  req,
		Response: NewGetGameTimeoutConfigResponse(req.BaseRequest.RequestUUID),
	}

	return task, nil
}

func (task *GetGameTimeoutConfigTask) Run(c *gin.Context) (Response, error) {
	// 通过gRPC调用RoomServer的GetGameTimeoutConfig
	rpcClient := client.GetGlobalRpcClient()
	if rpcClient == nil {
		task.Response.BaseResponse.RetCode = 1002
		task.Response.BaseResponse.Message = "gRPC client not initialized"
		return task.Response, nil
	}

	timeoutConfig, err := rpcClient.GetGameTimeoutConfig(context.Background(), &emptypb.Empty{})
	if err != nil {
		task.Response.BaseResponse.RetCode = 1002
		task.Response.BaseResponse.Message = "RoomServer GetGameTimeoutConfig failed: " + err.Error()
		return task.Response, nil
	}

	task.Response.GameMatchTimeout = timeoutConfig.GameMatchTimeout
	task.Response.RoundConfirmTimeout = timeoutConfig.RoundConfirmTimeout
	task.Response.RoundTimeout = timeoutConfig.RoundTimeout
	task.Response.ContinueTimeout = timeoutConfig.ContinueTimeout
	task.Response.BaseResponse.RetCode = 0
	task.Response.BaseResponse.Message = "Successfully retrieved game timeout config"
	return task.Response, nil
}
