package api

import (
	"context"
	"fmt"
	"strings"

	"github.com/CryptoElementals/common/log"
	"github.com/CryptoElementals/common/rpc/client"
	"github.com/CryptoElementals/common/rpc/proto"
	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
	"github.com/mitchellh/mapstructure"
)

func init() {
	Register(GET_PLAYER_STATUS_LABEL, NewGetPlayerStatusTask, COOKIEAUTH)
}

// GetPlayerStatusRequest 请求结构体
type GetPlayerStatusRequest struct {
	BaseRequest
	TempAddress string `mapstructure:"TempAddress" validate:"required"` // 临时地址
	Address     string `mapstructure:"Address"`
}

// GetPlayerStatusResponse 响应结构体
// Status 取值：0 default，1 matching，2 confirming，3 inbattle，4 waitingcontinue
// Action 需返回 GetPlayerStatusResponse
type GetPlayerStatusResponse struct {
	BaseResponse
	Status uint32 `json:"Status"`
}

type GetPlayerStatusTask struct {
	Request  *GetPlayerStatusRequest
	Response *GetPlayerStatusResponse
}

// 解码请求
func NewGetPlayerStatusRequest(data *map[string]interface{}) (*GetPlayerStatusRequest, error) {
	req := &GetPlayerStatusRequest{}
	err := mapstructure.Decode(*data, &req)
	if err != nil {
		return nil, err
	}
	req.BaseRequest.RequestUUID = (*data)["RequestUUID"].(string)
	return req, nil
}

func NewGetPlayerStatusResponse(sessionId string) *GetPlayerStatusResponse {
	return &GetPlayerStatusResponse{
		BaseResponse: BaseResponse{
			Action:      GET_PLAYER_STATUS_LABEL + "Response",
			RequestUUID: sessionId,
		},
	}
}

func NewGetPlayerStatusTask(data *map[string]interface{}) (Task, error) {
	req, err := NewGetPlayerStatusRequest(data)
	if err != nil {
		return nil, err
	}
	task := &GetPlayerStatusTask{
		Request:  req,
		Response: NewGetPlayerStatusResponse(req.BaseRequest.RequestUUID),
	}

	validate := validator.New()
	err = validate.Struct(task.Request)
	if err != nil {
		return nil, err
	}

	return task, nil
}

// Run 执行任务
func (task *GetPlayerStatusTask) Run(c *gin.Context) (Response, error) {
	// 获取玩家地址（从认证中间件填充到请求结构）
	address := task.Request.Address
	if address == "" {
		return nil, fmt.Errorf("failed to get player address")
	}

	// 统一为小写
	address = strings.ToLower(address)
	tempAddress := strings.ToLower(task.Request.TempAddress)

	// gRPC 客户端
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

	// 1) 调用 IsPlayerInQueue
	inQueueResp, err := rpcClient.IsPlayerInQueue(context.Background(), playerAddr)
	if err != nil {
		task.Response.BaseResponse.RetCode = 1003
		task.Response.BaseResponse.Message = "RoomServer IsPlayerInQueue failed: " + err.Error()
		return task.Response, nil
	}
	if inQueueResp != nil && inQueueResp.IsInQueue {
		// 匹配中 -> Status=1
		task.Response.Status = 1
		task.Response.BaseResponse.RetCode = 0
		task.Response.BaseResponse.Message = "Player is in match queue"
		return task.Response, nil
	}

	// 2) 不在队列中，则调用 GetGamePhase 判定 phase
	gamePhase, err := rpcClient.GetGamePhase(context.Background(), playerAddr)
	log.Infof("gamePhase: %v, err: %v", gamePhase, err)
	if err != nil {
		task.Response.BaseResponse.RetCode = 0
		task.Response.BaseResponse.Message = "Player is not participating in any game"
		return task.Response, nil
	}

	// 将 proto.PlayerStatus 转换为所需的 Status：
	// 2: confirming, 3: inbattle, 4: waitingcontinue；0: default
	switch gamePhase.PvPInfo.Status {
	case proto.PlayerStatus_PLAYER_MATCHED:
		// confirming
		task.Response.Status = 2
		task.Response.BaseResponse.Message = "Player matched, waiting for confirmation"
	case proto.PlayerStatus_PLAYER_IN_GAME:
		// inbattle
		task.Response.Status = 3
		task.Response.BaseResponse.Message = "Player has entered battle"
	case proto.PlayerStatus_PLAYER_WAITTING_CONTINUE:
		// waitingcontinue
		task.Response.Status = 4
		task.Response.BaseResponse.Message = "Player is waiting for continue"
	default:
		// 不在队列且 phase 为 0 -> default
		task.Response.Status = 0
		task.Response.BaseResponse.Message = "Player is not participating in any game"
	}

	task.Response.BaseResponse.RetCode = 0
	return task.Response, nil
}
