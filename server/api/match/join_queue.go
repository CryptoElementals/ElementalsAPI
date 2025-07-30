package match

import (
	"context"
	"fmt"
	"strings"

	"github.com/CryptoElementals/common/config"
	"github.com/CryptoElementals/common/db"
	"github.com/CryptoElementals/common/rpc/client"
	"github.com/CryptoElementals/common/rpc/proto"
	"github.com/CryptoElementals/common/server/api"
	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
	"github.com/mitchellh/mapstructure"
)

const JOIN_QUEUE_LABEL = "JoinQueue"

// JoinQueueRequest 请求结构体
type JoinQueueRequest struct {
	api.BaseRequest
	Mode        string `mapstructure:"Mode" validate:"required"`
	TempAddress string `mapstructure:"TempAddress" validate:"required"`
}

// JoinQueueResponse 响应结构体
type JoinQueueResponse struct {
	api.BaseResponse
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
		BaseResponse: api.BaseResponse{
			Action:      JOIN_QUEUE_LABEL + "Response",
			RequestUUID: sessionId,
		},
	}
}

func NewJoinQueueTask(data *map[string]interface{}) (api.Task, error) {
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

func (task *JoinQueueTask) Run(c *gin.Context) (api.Response, error) {
	// 获取玩家地址（从认证中间件设置的params中获取）
	_params, _ := c.Get("params")
	params, ok := _params.(*map[string]interface{})
	if !ok {
		task.Response.BaseResponse.RetCode = 1001
		task.Response.BaseResponse.Message = "Parameter parsing failed"
		return task.Response, nil
	}

	address, ok := (*params)["Address"].(string)
	if !ok || address == "" {
		task.Response.BaseResponse.RetCode = 1001
		task.Response.BaseResponse.Message = "Failed to get player address"
		return task.Response, nil
	}

	// 将地址转换为小写，确保与数据库中存储的格式一致（用于数据库查询）
	lowercaseAddress := strings.ToLower(address)
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

	// 检查用户token数量是否足够
	userToken, err := db.GetPlayerToken(c.Request.Context(), lowercaseAddress)
	if err != nil {
		task.Response.BaseResponse.RetCode = 1003
		task.Response.BaseResponse.Message = "Failed to get user token information"
		return task.Response, nil
	}

	// 获取用户已锁定的代币总数
	totalLockedTokens, err := db.GetTotalLockedTokensByAddress(lowercaseAddress)
	if err != nil {
		task.Response.BaseResponse.RetCode = 1003
		task.Response.BaseResponse.Message = "Failed to get locked token information"
		return task.Response, nil
	}

	var currentTokens int32 = 0
	if userToken != nil {
		currentTokens = userToken.TokenAmount
	}

	// 计算可用代币数量：用户token减去已锁定
	availableTokens := int(currentTokens) - totalLockedTokens

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
		WalletAddress:    lowercaseAddress,
		TemporaryAddress: lowercaseTempAddress,
	}

	_, err = rpcClient.JoinQueue(context.Background(), playerAddr)
	if err != nil {
		task.Response.BaseResponse.RetCode = 1002
		task.Response.BaseResponse.Message = "RoomServer JoinQueue failed: " + err.Error()
		return task.Response, nil
	}

	// roomserver 进行匹配
	task.Response.BaseResponse.RetCode = 0
	task.Response.BaseResponse.Message = "Successfully joined match queue"

	return task.Response, nil
}

// RegisterMatchApis 注册匹配相关API
func RegisterMatchApis() {
	api.Register(JOIN_QUEUE_LABEL, NewJoinQueueTask, api.COOKIEAUTH)
	api.Register(EXIT_QUEUE_LABEL, NewExitQueueTask, api.COOKIEAUTH)
	api.Register(CONFIRM_BATTLE_LABEL, NewConfirmBattleTask, api.COOKIEAUTH)
	api.Register(GET_GAME_PHASE_LABEL, NewGetGamePhaseTask, api.COOKIEAUTH)
	api.Register(REFUSE_CONTINUE_GAME_LABEL, NewRefuseContinueGameTask, api.COOKIEAUTH)
	api.Register(CONTINUE_GAME_LABEL, NewContinueGameTask, api.COOKIEAUTH)
}
