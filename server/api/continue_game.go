package api

import (
	"context"
	"fmt"
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
	Register(CONTINUE_GAME_LABEL, NewContinueGameTask, COOKIEAUTH)
}

// ContinueGameRequest 请求结构体
type ContinueGameRequest struct {
	BaseRequest
	GameID      uint   `mapstructure:"GameID" validate:"required"`
	TempAddress string `mapstructure:"TempAddress" validate:"required"`
	Address     string `mapstructure:"Address"`
}

// ContinueGameResponse 响应结构体
type ContinueGameResponse struct {
	BaseResponse
}

type ContinueGameTask struct {
	Request  *ContinueGameRequest
	Response *ContinueGameResponse
}

// 解码请求
func NewContinueGameRequest(data *map[string]interface{}) (*ContinueGameRequest, error) {
	req := &ContinueGameRequest{}
	err := mapstructure.Decode(*data, &req)
	if err != nil {
		return nil, err
	}
	req.BaseRequest.RequestUUID = (*data)["RequestUUID"].(string)
	return req, nil
}

func NewContinueGameResponse(sessionId string) *ContinueGameResponse {
	return &ContinueGameResponse{
		BaseResponse: BaseResponse{
			Action:      CONTINUE_GAME_LABEL + "Response",
			RequestUUID: sessionId,
		},
	}
}

func NewContinueGameTask(data *map[string]interface{}) (Task, error) {
	req, err := NewContinueGameRequest(data)
	if err != nil {
		return nil, err
	}
	task := &ContinueGameTask{
		Request:  req,
		Response: NewContinueGameResponse(req.BaseRequest.RequestUUID),
	}

	validate := validator.New()
	err = validate.Struct(task.Request)
	if err != nil {
		return nil, err
	}

	return task, nil
}

func (task *ContinueGameTask) Run(c *gin.Context) (Response, error) {
	// 获取玩家地址（从认证中间件填充到请求结构）
	address := task.Request.Address
	if address == "" {
		task.Response.BaseResponse.RetCode = 1001
		task.Response.BaseResponse.Message = "Parameter parsing failed"
		return task.Response, nil
	}

	// 将地址转换为小写，确保与数据库中存储的格式一致
	address = strings.ToLower(address)
	tempAddress := strings.ToLower(task.Request.TempAddress)

	// 检查用户token数量是否足够
	userToken, err := db.GetPlayerToken(c.Request.Context(), address)
	if err != nil {
		task.Response.BaseResponse.RetCode = 1003
		task.Response.BaseResponse.Message = "Failed to get user token information"
		return task.Response, nil
	}

	// 获取用户已锁定的代币总数
	totalLockedTokens, err := db.GetTotalLockedTokensByAddress(address)
	if err != nil {
		task.Response.BaseResponse.RetCode = 1003
		task.Response.BaseResponse.Message = "Failed to get locked token information"
		return task.Response, nil
	}

	var currentTokens int32 = 0
	if userToken != nil {
		currentTokens = userToken.TokenAmount
	}

	// 计算可用代币数量
	availableTokens := int(currentTokens) - totalLockedTokens

	if availableTokens < config.GameParams.TokenThreshold {
		task.Response.BaseResponse.RetCode = 1004
		task.Response.BaseResponse.Message = fmt.Sprintf("Insufficient available tokens, need at least %d tokens to continue game", config.GameParams.TokenThreshold)
		return task.Response, nil
	}

	// 通过gRPC调用RoomServer的ContinueGame
	rpcClient := client.GetGlobalRpcClient()
	if rpcClient == nil {
		task.Response.BaseResponse.RetCode = 1002
		task.Response.BaseResponse.Message = "gRPC client not initialized"
		return task.Response, nil
	}

	continueGameReq := &proto.ContinueGameRequest{
		Player: &proto.PlayerAddress{
			WalletAddress:    address,
			TemporaryAddress: tempAddress,
		},
		LastGameID: uint32(task.Request.GameID),
	}

	_, err = rpcClient.ContinueGame(context.Background(), continueGameReq)
	if err != nil {
		task.Response.BaseResponse.RetCode = 1002
		task.Response.BaseResponse.Message = "RoomServer ContinueGame failed: " + err.Error()
		return task.Response, nil
	}

	// 继续游戏成功
	task.Response.BaseResponse.RetCode = 0
	task.Response.BaseResponse.Message = "Successfully continued game"

	return task.Response, nil
}
