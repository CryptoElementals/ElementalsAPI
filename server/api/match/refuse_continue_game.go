package match

import (
	"context"
	"strings"

	"github.com/CryptoElementals/common/log"
	"github.com/CryptoElementals/common/rpc/client"
	"github.com/CryptoElementals/common/rpc/proto"
	"github.com/CryptoElementals/common/server/api"
	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
	"github.com/mitchellh/mapstructure"
)

const REFUSE_CONTINUE_GAME_LABEL = "RefuseContinueGame"

// RefuseContinueGameRequest 请求结构体
type RefuseContinueGameRequest struct {
	api.BaseRequest
	GameID      uint   `mapstructure:"GameID" validate:"required"`
	TempAddress string `mapstructure:"TempAddress" validate:"required"` // 临时地址
}

// RefuseContinueGameResponse 响应结构体
type RefuseContinueGameResponse struct {
	api.BaseResponse
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
		BaseResponse: api.BaseResponse{
			Action:      REFUSE_CONTINUE_GAME_LABEL + "Response",
			RequestUUID: sessionId,
		},
	}
}

func NewRefuseContinueGameTask(data *map[string]interface{}) (api.Task, error) {
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

func (task *RefuseContinueGameTask) Run(c *gin.Context) (api.Response, error) {
	// 获取玩家地址
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

	address = strings.ToLower(address)
	tempAddress := strings.ToLower(task.Request.TempAddress)

	log.Infof("RefuseContinueGame: %s, %s", address, tempAddress)

	// 通过gRPC调用RoomServer的RefuseContinueGame
	rpcClient := client.GetGlobalRpcClient()
	if rpcClient == nil {
		task.Response.BaseResponse.RetCode = 1002
		task.Response.BaseResponse.Message = "gRPC client not initialized"
		return task.Response, nil
	}

	refuseContinueGameReq := &proto.RefuseContinueGameRequest{
		Player: &proto.PlayerAddress{
			WalletAddress:    address,
			TemporaryAddress: tempAddress,
		},
		LastGameID: uint32(task.Request.GameID),
	}

	_, err := rpcClient.RefuseContinueGame(context.Background(), refuseContinueGameReq)
	if err != nil {
		task.Response.BaseResponse.RetCode = 1002
		task.Response.BaseResponse.Message = "RoomServer RefuseContinueGame failed: " + err.Error()
		return task.Response, nil
	}

	// 拒绝继续游戏成功
	task.Response.BaseResponse.RetCode = 0
	task.Response.BaseResponse.Message = "Successfully refused to continue game"

	return task.Response, nil
}
