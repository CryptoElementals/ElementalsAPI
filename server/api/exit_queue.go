package api

import (
	"context"
	"strings"

	"github.com/CryptoElementals/common/db"
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
	UserID      string `mapstructure:"UserID" validate:"required"`
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
	// 通过 UserID 解析玩家地址
	profile, err := db.GetUserProfileByUserID(strings.TrimSpace(task.Request.UserID))
	if err != nil || profile == nil || profile.Address == "" {
		task.Response.BaseResponse.RetCode = 1001
		task.Response.BaseResponse.Message = "Failed to get player address by user id"
		return task.Response, nil
	}
	address := profile.Address

	// 将地址转换为小写，确保与数据库中存储的格式一致
	address = strings.ToLower(address)
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
		WalletAddress:    address,
		TemporaryAddress: tempAddress,
	}

	_, err = rpcClient.ExitQueue(context.Background(), playerAddr)
	if err != nil {
		task.Response.BaseResponse.RetCode = 1002
		task.Response.BaseResponse.Message = "RoomServer ExitQueue failed: " + err.Error()
		return task.Response, nil
	}

	task.Response.BaseResponse.RetCode = 0
	task.Response.BaseResponse.Message = "Successfully left match queue"

	return task.Response, nil
}
