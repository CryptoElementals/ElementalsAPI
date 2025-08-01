package battle

import (
	"context"
	"fmt"
	"strings"

	"github.com/CryptoElementals/common/rpc/client"
	pb "github.com/CryptoElementals/common/rpc/proto"
	"github.com/CryptoElementals/common/server/api"
	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
	"github.com/mitchellh/mapstructure"
)

const SURRENDER_LABEL = "Surrender"

// SurrenderRequest 请求结构体
type SurrenderRequest struct {
	api.BaseRequest
	TempAddress string `mapstructure:"TempAddress" validate:"required"` // 临时地址
	GameID      uint32 `mapstructure:"GameID" validate:"required"`      // 游戏ID
}

// SurrenderResponse 响应结构体
type SurrenderResponse struct {
	api.BaseResponse
}

type SurrenderTask struct {
	Request  *SurrenderRequest
	Response *SurrenderResponse
}

// 解码请求
func NewSurrenderRequest(data *map[string]interface{}) (*SurrenderRequest, error) {
	req := &SurrenderRequest{}
	err := mapstructure.Decode(*data, &req)
	if err != nil {
		return nil, err
	}
	req.BaseRequest.RequestUUID = (*data)["RequestUUID"].(string)
	return req, nil
}

func NewSurrenderResponse(sessionId string) *SurrenderResponse {
	return &SurrenderResponse{
		BaseResponse: api.BaseResponse{
			Action:      SURRENDER_LABEL + "Response",
			RequestUUID: sessionId,
		},
	}
}

func NewSurrenderTask(data *map[string]interface{}) (api.Task, error) {
	req, err := NewSurrenderRequest(data)
	if err != nil {
		return nil, err
	}
	task := &SurrenderTask{
		Request:  req,
		Response: NewSurrenderResponse(req.BaseRequest.RequestUUID),
	}

	validate := validator.New()
	err = validate.Struct(task.Request)
	if err != nil {
		return nil, err
	}

	return task, nil
}

// Run 执行任务
func (task *SurrenderTask) Run(c *gin.Context) (api.Response, error) {
	// 获取玩家地址（从认证中间件设置的params中获取）
	_params, _ := c.Get("params")
	params, ok := _params.(*map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("parameter parsing failed")
	}

	address, ok := (*params)["Address"].(string)
	if !ok || address == "" {
		return nil, fmt.Errorf("failed to get player address")
	}

	// 将地址转换为小写，确保与数据库中存储的格式一致
	address = strings.ToLower(address)
	tempAddress := strings.ToLower(task.Request.TempAddress)

	// 通过gRPC调用RoomServer的Surrender
	rpcClient := client.GetGlobalRpcClient()
	if rpcClient == nil {
		task.Response.BaseResponse.RetCode = 1002
		task.Response.BaseResponse.Message = "gRPC client not initialized"
		return task.Response, nil
	}

	// 创建 SurrenderRequest
	req := &pb.SurrenderRequest{
		GameID: task.Request.GameID,
		Address: &pb.PlayerAddress{
			WalletAddress:    address,
			TemporaryAddress: tempAddress,
		},
	}

	// 调用 Surrender RPC
	_, err := rpcClient.Surrender(context.Background(), req)
	if err != nil {
		task.Response.BaseResponse.RetCode = 1003
		task.Response.BaseResponse.Message = "RoomServer Surrender failed: " + err.Error()
		return task.Response, nil
	}

	// 返回成功
	task.Response.BaseResponse.RetCode = 0
	task.Response.BaseResponse.Message = "Successfully surrendered"
	return task.Response, nil
}
