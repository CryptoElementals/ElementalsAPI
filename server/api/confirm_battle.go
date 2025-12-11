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
	Register(CONFIRM_BATTLE_LABEL, NewConfirmBattleTask, COOKIEAUTH)
}

// ConfirmBattleRequest 请求结构体
type ConfirmBattleRequest struct {
	BaseRequest
	GameID      uint32 `mapstructure:"GameID" validate:"required"`
	RoundNumber uint   `mapstructure:"RoundNumber" validate:"required,min=1"`
	TurnNumber  uint   `mapstructure:"TurnNumber" validate:"required,min=1"`
	TempAddress string `mapstructure:"TempAddress" validate:"required"`
	PlayerID    string `mapstructure:"PlayerID" validate:"required"`
}

// ConfirmBattleResponse 响应结构体
type ConfirmBattleResponse struct {
	BaseResponse
}

type ConfirmBattleTask struct {
	Request  *ConfirmBattleRequest
	Response *ConfirmBattleResponse
}

// 解码请求
func NewConfirmBattleRequest(data *map[string]interface{}) (*ConfirmBattleRequest, error) {
	req := &ConfirmBattleRequest{}
	err := mapstructure.Decode(*data, &req)
	if err != nil {
		return nil, err
	}
	req.BaseRequest.RequestUUID = (*data)["RequestUUID"].(string)
	return req, nil
}

func NewConfirmBattleResponse(sessionId string) *ConfirmBattleResponse {
	return &ConfirmBattleResponse{
		BaseResponse: BaseResponse{
			Action:      CONFIRM_BATTLE_LABEL + "Response",
			RequestUUID: sessionId,
		},
	}
}

func NewConfirmBattleTask(data *map[string]interface{}) (Task, error) {
	req, err := NewConfirmBattleRequest(data)
	if err != nil {
		return nil, err
	}
	task := &ConfirmBattleTask{
		Request:  req,
		Response: NewConfirmBattleResponse(req.BaseRequest.RequestUUID),
	}

	validate := validator.New()
	err = validate.Struct(task.Request)
	if err != nil {
		return nil, err
	}

	return task, nil
}

func (task *ConfirmBattleTask) Run(c *gin.Context) (Response, error) {
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

	tempAddress := strings.ToLower(task.Request.TempAddress)

	// 通过gRPC调用RoomServer的ConfirmBattle
	rpcClient := client.GetGlobalRpcClient()
	if rpcClient == nil {
		task.Response.BaseResponse.RetCode = 1002
		task.Response.BaseResponse.Message = "gRPC client not initialized"
		return task.Response, nil
	}

	req := &proto.ConfirmBattleRequest{
		GameID:      uint32(task.Request.GameID),
		RoundNumber: uint32(task.Request.RoundNumber),
		TurnNumber:  uint32(task.Request.TurnNumber),
		PlayerAddress: &proto.PlayerAddress{
			Id:               playerID,
			TemporaryAddress: tempAddress,
		},
	}

	_, err = rpcClient.ConfirmBattle(context.Background(), req)
	if err != nil {
		task.Response.BaseResponse.RetCode = 1002
		task.Response.BaseResponse.Message = "RoomServer ConfirmBattle failed: " + err.Error()
		return task.Response, nil
	}

	task.Response.BaseResponse.RetCode = 0
	task.Response.BaseResponse.Message = "Successfully confirmed battle"
	return task.Response, nil
}
