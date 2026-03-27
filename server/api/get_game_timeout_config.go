package api

import (
	"context"

	"github.com/CryptoElementals/common/rpc/client"
	"github.com/gin-gonic/gin"
	"github.com/mitchellh/mapstructure"
	"google.golang.org/protobuf/types/known/emptypb"
)

func init() {
	// 该 API 目前未被使用，超时时间在推送事件里返回
	Register(GET_GAME_TIMEOUT_CONFIG_LABEL, NewGetGameTimeoutConfigTask, NOAUTH)
}

// GetGameTimeoutConfigRequest 请求结构体
type GetGameTimeoutConfigRequest struct {
	BaseRequest
}

// GetGameTimeoutConfigResponse 响应结构体
type GetGameTimeoutConfigResponse struct {
	BaseResponse
	ConfirmationTimeout         int64 `json:"ConfirmationTimeout"`         // 确认超时时间（秒）
	CommitmentSubmissionTimeout int64 `json:"CommitmentSubmissionTimeout"` // 承诺提交超时时间（秒）
	CardSubmissionTimeout       int64 `json:"CardSubmissionTimeout"`       // 卡牌提交超时时间（秒）
	GameContinueTimeout         int64 `json:"GameContinueTimeout"`         // 继续游戏超时时间（秒）
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
		task.Response.BaseResponse.Message = "GetGameTimeoutConfig failed. Internal error: " + ShortGRPCError(err)
		return task.Response, nil
	}

	task.Response.ConfirmationTimeout = timeoutConfig.ConfirmationTimeout
	task.Response.CommitmentSubmissionTimeout = timeoutConfig.CommitmentSubmissionTimeout
	task.Response.CardSubmissionTimeout = timeoutConfig.CardSubmissionTimeout
	task.Response.GameContinueTimeout = timeoutConfig.GameContinueTimeout
	task.Response.BaseResponse.RetCode = 0
	task.Response.BaseResponse.Message = "Successfully retrieved game timeout config"
	return task.Response, nil
}
