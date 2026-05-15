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

	lobbyClient := client.LobbyClientForType(ServerTypeFromGin(c))
	if lobbyClient == nil {
		task.Response.BaseResponse.RetCode = 1002
		task.Response.BaseResponse.Message = "gRPC lobby client not initialized"
		return task.Response, nil
	}

	playerAddr := &proto.PlayerAddress{
		Id:               playerID,
		TemporaryAddress: lowercaseTempAddress,
	}

	_, err = lobbyClient.JoinQueue(context.Background(), playerAddr)
	if err != nil {
		shortErr := ShortGRPCError(err)
		// 检查apiserver的TokenThreshold配置是否与roomserver一致
		if strings.Contains(shortErr, "user token is not enough") {
			task.Response.BaseResponse.RetCode = 1004
			//task.Response.BaseResponse.Message = fmt.Sprintf("Insufficient available tokens, need at least %d tokens to join match queue", config.GameParams.TokenThreshold)
			task.Response.BaseResponse.Message = "Insufficient available tokens"
			return task.Response, nil
		} else if strings.Contains(shortErr, "player cannot join queue, player status: PLAYER_IN_GAME") {
			// 玩家已经在对局中，不能再次加入匹配队列，返回业务错误码 1006
			task.Response.BaseResponse.RetCode = 1006
			task.Response.BaseResponse.Message = "Player is already in game and cannot join match queue"
			return task.Response, nil
		} else if strings.Contains(shortErr, "player cannot join queue, player status: PLAYER_MATCHED") {
			// 玩家已经匹配上在等待确认，不能再次加入匹配队列，返回业务错误码 1007
			task.Response.BaseResponse.RetCode = 1007
			task.Response.BaseResponse.Message = "Player is already matched and cannot join match queue"
			return task.Response, nil
		} else if strings.Contains(shortErr, "player in tournament") {
			// lobby JoinQueue: FailedPrecondition when player is queued or in bracket in an open/in-progress tournament
			task.Response.BaseResponse.RetCode = 1009
			task.Response.BaseResponse.Message = "Player is in a tournament and cannot join match queue"
			return task.Response, nil
		}
		task.Response.BaseResponse.RetCode = 1002
		task.Response.BaseResponse.Message = "Lobby JoinQueue failed: " + err.Error()
		return task.Response, nil
	}

	// roomserver 进行匹配
	task.Response.BaseResponse.RetCode = 0
	task.Response.BaseResponse.Message = "Successfully joined match queue"

	return task.Response, nil
}
