package api

import (
	"context"
	"strconv"
	"strings"

	"github.com/CryptoElementals/common/log"
	"github.com/CryptoElementals/common/rpc/client"
	"github.com/CryptoElementals/common/rpc/proto"
	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
	"github.com/mitchellh/mapstructure"
)

func init() {
	Register(REFUSE_CONTINUE_GAME_LABEL, NewRefuseContinueGameTask, COOKIEAUTH)
}

// RefuseContinueGameRequest 请求结构体
type RefuseContinueGameRequest struct {
	BaseRequest
	GameID      uint   `mapstructure:"GameID" validate:"required"`
	TempAddress string `mapstructure:"TempAddress" validate:"required"` // 临时地址
	PlayerID    string `mapstructure:"PlayerID" validate:"required"`
}

// RefuseContinueGameResponse 响应结构体
type RefuseContinueGameResponse struct {
	BaseResponse
}

type RefuseContinueGameTask struct {
	Request  *RefuseContinueGameRequest
	Response *RefuseContinueGameResponse
}

func NewRefuseContinueGameRequest(data *map[string]interface{}) (*RefuseContinueGameRequest, error) {
	req := &RefuseContinueGameRequest{}
	err := mapstructure.Decode(*data, &req)
	if err != nil {
		return nil, err
	}
	req.BaseRequest.RequestUUID = (*data)["RequestUUID"].(string)
	return req, nil
}

func NewRefuseContinueGameResponse(sessionId string) *RefuseContinueGameResponse {
	return &RefuseContinueGameResponse{
		BaseResponse: BaseResponse{
			Action:      REFUSE_CONTINUE_GAME_LABEL + "Response",
			RequestUUID: sessionId,
		},
	}
}

func NewRefuseContinueGameTask(data *map[string]interface{}) (Task, error) {
	req, err := NewRefuseContinueGameRequest(data)
	if err != nil {
		return nil, err
	}
	task := &RefuseContinueGameTask{
		Request:  req,
		Response: NewRefuseContinueGameResponse(req.BaseRequest.RequestUUID),
	}

	validate := validator.New()
	err = validate.Struct(task.Request)
	if err != nil {
		return nil, err
	}

	return task, nil
}

func (task *RefuseContinueGameTask) Run(c *gin.Context) (Response, error) {
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

	log.Infof("RefuseContinueGame: player_id=%d, tempAddress=%s", playerID, tempAddress)

	// 通过gRPC调用RoomServer的RefuseContinueGame
	rpcClient := client.GetGlobalRpcClient()
	if rpcClient == nil {
		task.Response.BaseResponse.RetCode = 1002
		task.Response.BaseResponse.Message = "gRPC client not initialized"
		return task.Response, nil
	}

	refuseContinueGameReq := &proto.RefuseContinueGameRequest{
		Player: &proto.PlayerAddress{
			Id:               playerID,
			TemporaryAddress: tempAddress,
		},
		LastGameID: uint32(task.Request.GameID),
	}

	_, err = rpcClient.RefuseContinueGame(context.Background(), refuseContinueGameReq)
	if err != nil {
		task.Response.BaseResponse.RetCode = 1002
		task.Response.BaseResponse.Message = "RefuseContinueGame failed. Internal error: " + ShortGRPCError(err)
		return task.Response, nil
	}

	// 拒绝继续游戏成功
	task.Response.BaseResponse.RetCode = 0
	task.Response.BaseResponse.Message = "Successfully refused to continue game"

	return task.Response, nil
}
