package api

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/CryptoElementals/common/rpc/client"
	pb "github.com/CryptoElementals/common/rpc/proto"
	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
	"github.com/mitchellh/mapstructure"
)

func init() {
	Register(SURRENDER_LABEL, NewSurrenderTask, COOKIEAUTH)
}

// SurrenderRequest 请求结构体
type SurrenderRequest struct {
	BaseRequest
	TempAddress string `mapstructure:"TempAddress" validate:"required"` // 临时地址
	GameID      int64  `mapstructure:"GameID" validate:"required"`      // 游戏ID
	PlayerID    string `mapstructure:"PlayerID" validate:"required"`
}

// SurrenderResponse 响应结构体
type SurrenderResponse struct {
	BaseResponse
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
		BaseResponse: BaseResponse{
			Action:      SURRENDER_LABEL + "Response",
			RequestUUID: sessionId,
		},
	}
}

func NewSurrenderTask(data *map[string]interface{}) (Task, error) {
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

func init() {
	Register(SURRENDER_LABEL, NewSurrenderTask, COOKIEAUTH)
}

// Run 执行任务
func (task *SurrenderTask) Run(c *gin.Context) (Response, error) {
	// 解析 PlayerID（由中间件从会话中注入），前端只需要传临时地址
	playerIDStr := strings.TrimSpace(task.Request.PlayerID)
	if playerIDStr == "" {
		return nil, fmt.Errorf("player id is empty")
	}
	playerID, err := strconv.ParseInt(playerIDStr, 10, 64)
	if err != nil {
		return nil, fmt.Errorf("invalid player id: %v", err)
	}
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
			Id:               playerID,
			TemporaryAddress: tempAddress,
		},
	}

	// 调用 Surrender RPC
	_, err = rpcClient.Surrender(context.Background(), req)
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
