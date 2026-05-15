package api

import (
	"context"
	"encoding/hex"
	"strconv"
	"strings"

	"github.com/CryptoElementals/common/rpc/client"
	"github.com/CryptoElementals/common/rpc/proto"
	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
	"github.com/mitchellh/mapstructure"
)

func init() {
	Register(SUBMIT_PLAYER_CARD_LABEL, NewSubmitPlayerCardTask, COOKIEAUTH)
}

// SubmitPlayerCardRequest 请求结构体
type SubmitPlayerCardRequest struct {
	BaseRequest
	GameID      int64  `mapstructure:"GameID" validate:"required"`
	RoundNumber uint32 `mapstructure:"RoundNumber" validate:"required"`
	TurnNumber  uint32 `mapstructure:"TurnNumber" validate:"required"`
	Card        uint32 `mapstructure:"Card" validate:"required"`
	Salt        string `mapstructure:"Salt" validate:"required"`      // hex 字符串或普通字符串
	Signature   string `mapstructure:"Signature" validate:"required"` // hex 字符串
	TempAddress string `mapstructure:"TempAddress" validate:"required"`
	PlayerID    string `mapstructure:"PlayerID" validate:"required"`
}

// SubmitPlayerCardResponse 响应结构体
type SubmitPlayerCardResponse struct {
	BaseResponse
}

type SubmitPlayerCardTask struct {
	Request  *SubmitPlayerCardRequest
	Response *SubmitPlayerCardResponse
}

func NewSubmitPlayerCardRequest(data *map[string]interface{}) (*SubmitPlayerCardRequest, error) {
	req := &SubmitPlayerCardRequest{}
	err := mapstructure.Decode(*data, &req)
	if err != nil {
		return nil, err
	}
	req.BaseRequest.RequestUUID = (*data)["RequestUUID"].(string)
	return req, nil
}

func NewSubmitPlayerCardResponse(sessionId string) *SubmitPlayerCardResponse {
	return &SubmitPlayerCardResponse{
		BaseResponse: BaseResponse{
			Action:      SUBMIT_PLAYER_CARD_LABEL + "Response",
			RequestUUID: sessionId,
		},
	}
}

func NewSubmitPlayerCardTask(data *map[string]interface{}) (Task, error) {
	req, err := NewSubmitPlayerCardRequest(data)
	if err != nil {
		return nil, err
	}
	task := &SubmitPlayerCardTask{
		Request:  req,
		Response: NewSubmitPlayerCardResponse(req.BaseRequest.RequestUUID),
	}

	validate := validator.New()
	err = validate.Struct(task.Request)
	if err != nil {
		return nil, err
	}

	return task, nil
}

func (task *SubmitPlayerCardTask) Run(c *gin.Context) (Response, error) {
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

	// 统一将地址转为小写
	tempAddress := strings.ToLower(task.Request.TempAddress)

	// 将 hex 字符串转换为 bytes（如果 Salt 是 hex 格式）
	var saltBytes []byte
	saltHex := strings.TrimPrefix(task.Request.Salt, "0x")
	if decoded, err := hex.DecodeString(saltHex); err == nil {
		// 如果能够成功解码为 hex，使用解码后的 bytes
		saltBytes = decoded
	} else {
		// 否则直接使用字符串的 bytes
		saltBytes = []byte(task.Request.Salt)
	}

	signatureHex := strings.TrimPrefix(task.Request.Signature, "0x")
	signatureBytes, err := hex.DecodeString(signatureHex)
	if err != nil {
		task.Response.BaseResponse.RetCode = 1003
		task.Response.BaseResponse.Message = "Invalid signature hex format: " + err.Error()
		return task.Response, nil
	}

	// 通过gRPC调用RoomServer的SubmitPlayerCard
	rpcClient := client.RoomClientForType(ServerTypeFromGin(c))
	if rpcClient == nil {
		task.Response.BaseResponse.RetCode = 1002
		task.Response.BaseResponse.Message = "gRPC client not initialized"
		return task.Response, nil
	}

	req := &proto.SubmitPlayerCardRequest{
		GameID:      task.Request.GameID,
		RoundNumber: task.Request.RoundNumber,
		TurnNumber:  task.Request.TurnNumber,
		Card:        task.Request.Card,
		Salt:        saltBytes,
		Signature:   signatureBytes,
		Address: &proto.PlayerAddress{
			Id:               playerID,
			TemporaryAddress: tempAddress,
		},
	}

	_, err = rpcClient.SubmitPlayerCard(context.Background(), req)
	if err != nil {
		task.Response.BaseResponse.RetCode = 1002
		task.Response.BaseResponse.Message = "RoomServer SubmitPlayerCard failed: " + err.Error()
		return task.Response, nil
	}

	task.Response.BaseResponse.RetCode = 0
	task.Response.BaseResponse.Message = "Successfully submitted player card"
	return task.Response, nil
}
