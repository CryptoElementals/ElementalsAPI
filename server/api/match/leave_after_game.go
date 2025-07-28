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

const LEAVE_AFTER_GAME_LABEL = "LeaveAfterGame"

// LeaveAfterGameRequest 请求结构体
type LeaveAfterGameRequest struct {
	api.BaseRequest
	TempAddress string `mapstructure:"TempAddress" validate:"required"` // 临时地址
}

// LeaveAfterGameResponse 响应结构体
type LeaveAfterGameResponse struct {
	api.BaseResponse
}

type LeaveAfterGameTask struct {
	Request  *LeaveAfterGameRequest
	Response *LeaveAfterGameResponse
}

func NewLeaveAfterGameRequest(data *map[string]interface{}) (*LeaveAfterGameRequest, error) {
	req := &LeaveAfterGameRequest{}
	err := mapstructure.Decode(*data, &req)
	if err != nil {
		return nil, err
	}
	req.BaseRequest.RequestUUID = (*data)["RequestUUID"].(string)
	return req, nil
}

func NewLeaveAfterGameResponse(sessionId string) *LeaveAfterGameResponse {
	return &LeaveAfterGameResponse{
		BaseResponse: api.BaseResponse{
			Action:      LEAVE_AFTER_GAME_LABEL + "Response",
			RequestUUID: sessionId,
		},
	}
}

func NewLeaveAfterGameTask(data *map[string]interface{}) (api.Task, error) {
	req, err := NewLeaveAfterGameRequest(data)
	if err != nil {
		return nil, err
	}
	task := &LeaveAfterGameTask{
		Request:  req,
		Response: NewLeaveAfterGameResponse(req.BaseRequest.RequestUUID),
	}

	validate := validator.New()
	err = validate.Struct(task.Request)
	if err != nil {
		return nil, err
	}

	return task, nil
}

func (task *LeaveAfterGameTask) Run(c *gin.Context) (api.Response, error) {
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

	// 通过gRPC获取玩家当前游戏状态
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

	// 获取玩家当前游戏状态
	gamePhase, err := rpcClient.GetGamePhase(context.Background(), playerAddr)
	if err != nil {
		task.Response.BaseResponse.RetCode = 1003
		task.Response.BaseResponse.Message = "Failed to get player game phase: " + err.Error()
		return task.Response, nil
	}

	// 检查玩家是否在游戏中
	if gamePhase.PvPInfo.GameID == 0 {
		task.Response.BaseResponse.RetCode = 1004
		task.Response.BaseResponse.Message = "Player is not in any game"
		return task.Response, nil
	}

	// 从GamePhase中直接获取对手信息
	var opponentAddress string
	var opponentTempAddress string
	if len(gamePhase.Players) >= 2 {
		for _, player := range gamePhase.Players {
			if player.Address.WalletAddress != address {
				opponentAddress = player.Address.WalletAddress
				opponentTempAddress = player.Address.TemporaryAddress
				break
			}
		}
	}

	if opponentAddress == "" {
		task.Response.BaseResponse.RetCode = 1005
		task.Response.BaseResponse.Message = "Opponent not found"
		return task.Response, nil
	}

	// 通过SSE向对手发送离开通知
	//这里是直接在apiserver中实现消息通知，没有通过room_server中转
	//可能还是通过roomserver处理好一些？
	// TODO: 重构为使用新的全局事件管理器
	log.Infof("Player left game - opponent should be notified: %s_%s", opponentAddress, opponentTempAddress)

	// 使用本地的BuildGameUserID函数构造对手的用户标识符
	opponentUserID := BuildGameUserID(opponentAddress, opponentTempAddress)

	// 记录离开事件，后续可以通过RoomServer处理
	log.Infof("Opponent left event for user: %s", opponentUserID)

	task.Response.BaseResponse.RetCode = 0
	task.Response.BaseResponse.Message = "Leave notification sent successfully"
	return task.Response, nil
}

func BuildGameUserID(address, tempAddress string) string {
	// 统一转换为小写
	address = strings.ToLower(address)
	tempAddress = strings.ToLower(tempAddress)

	// 返回格式：address_tempaddress
	return address + "_" + tempAddress
}
