package api

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/CryptoElementals/common/config"
	"github.com/CryptoElementals/common/db"
	"github.com/CryptoElementals/common/rpc/client"
	"github.com/CryptoElementals/common/rpc/proto"
	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
	"github.com/mitchellh/mapstructure"
)

func init() {
	Register(JOIN_QUEUE_LABEL, NewJoinQueueTask, COOKIEAUTH)
}

// JoinQueueRequest 请求结构体
type JoinQueueRequest struct {
	BaseRequest
	Mode        string `mapstructure:"Mode" validate:"required"`
	TempAddress string `mapstructure:"TempAddress" validate:"required"`
	PlayerID    string `mapstructure:"PlayerID" validate:"required"`
}

// JoinQueueResponse 响应结构体
type JoinQueueResponse struct {
	BaseResponse
}

type JoinQueueTask struct {
	Request  *JoinQueueRequest
	Response *JoinQueueResponse
}

// 解码请求
func NewJoinQueueRequest(data *map[string]interface{}) (*JoinQueueRequest, error) {
	req := &JoinQueueRequest{}
	err := mapstructure.Decode(*data, &req)
	if err != nil {
		return nil, err
	}
	req.BaseRequest.RequestUUID = (*data)["RequestUUID"].(string)
	return req, nil
}

func NewJoinQueueResponse(sessionId string) *JoinQueueResponse {
	return &JoinQueueResponse{
		BaseResponse: BaseResponse{
			Action:      JOIN_QUEUE_LABEL + "Response",
			RequestUUID: sessionId,
		},
	}
}

func NewJoinQueueTask(data *map[string]interface{}) (Task, error) {
	req, err := NewJoinQueueRequest(data)
	if err != nil {
		return nil, err
	}
	task := &JoinQueueTask{
		Request:  req,
		Response: NewJoinQueueResponse(req.BaseRequest.RequestUUID),
	}

	validate := validator.New()
	err = validate.Struct(task.Request)
	if err != nil {
		return nil, err
	}

	return task, nil
}

func (task *JoinQueueTask) Run(c *gin.Context) (Response, error) {
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
	lowercaseTempAddress := strings.ToLower(task.Request.TempAddress)

	// 验证游戏模式
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

	// 检查用户token数量是否足够（仅按总 TokenAmount 判断，不再扣除锁仓）
	userToken, err := db.GetPlayerToken(c.Request.Context(), playerID)
	if err != nil {
		task.Response.BaseResponse.RetCode = 1003
		task.Response.BaseResponse.Message = "JoinQueue failed. Internal error: failed to get user token information"
		return task.Response, nil
	}

	var currentTokens int32 = 0
	if userToken != nil {
		currentTokens = userToken.TokenAmount
	}

	// 计算可用代币数量：仅使用当前总 token 数量
	availableTokens := int(currentTokens)

	// 要求用户至少有10000个可用代币才能加入匹配队列
	if availableTokens < config.GameParams.TokenThreshold {
		task.Response.BaseResponse.RetCode = 1004
		task.Response.BaseResponse.Message = fmt.Sprintf("Insufficient available tokens, need at least %d tokens to join match queue", config.GameParams.TokenThreshold)
		return task.Response, nil
	}

	// 通过gRPC调用RoomServer的JoinQueue
	rpcClient := client.GetGlobalRpcClient()
	if rpcClient == nil {
		task.Response.BaseResponse.RetCode = 1002
		task.Response.BaseResponse.Message = "gRPC client not initialized"
		return task.Response, nil
	}

	playerAddr := &proto.PlayerAddress{
		Id:               playerID,
		TemporaryAddress: lowercaseTempAddress,
	}

	_, err = rpcClient.JoinQueue(context.Background(), playerAddr)
	if err != nil {
		task.Response.BaseResponse.RetCode = 1002
		task.Response.BaseResponse.Message = "JoinQueue failed. Internal error: " + ShortGRPCError(err)
		return task.Response, nil
	}

	// roomserver 进行匹配
	task.Response.BaseResponse.RetCode = 0
	task.Response.BaseResponse.Message = "Successfully joined match queue"

	return task.Response, nil
}
