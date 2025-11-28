package api

import (
	"context"
	"encoding/hex"
	"strings"

	"github.com/CryptoElementals/common/db"
	"github.com/CryptoElementals/common/rpc/client"
	"github.com/CryptoElementals/common/rpc/proto"
	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
	"github.com/mitchellh/mapstructure"
)

func init() {
	Register(SUBMIT_PLAYER_COMMITMENT_LABEL, NewSubmitPlayerCommitmentTask, COOKIEAUTH)
}

// SubmitPlayerCommitmentRequest 请求结构体
type SubmitPlayerCommitmentRequest struct {
	BaseRequest
	GameID      uint32 `mapstructure:"GameID" validate:"required"`
	RoundNumber uint32 `mapstructure:"RoundNumber" validate:"required"`
	TurnNumber  uint32 `mapstructure:"TurnNumber" validate:"required"`
	Commitment  string `mapstructure:"Commitment" validate:"required"` // hex 字符串
	Signature   string `mapstructure:"Signature" validate:"required"`  // hex 字符串
	TempAddress string `mapstructure:"TempAddress" validate:"required"`
	PlayerID    string `mapstructure:"PlayerID" validate:"required"`
}

// SubmitPlayerCommitmentResponse 响应结构体
type SubmitPlayerCommitmentResponse struct {
	BaseResponse
}

type SubmitPlayerCommitmentTask struct {
	Request  *SubmitPlayerCommitmentRequest
	Response *SubmitPlayerCommitmentResponse
}

func NewSubmitPlayerCommitmentRequest(data *map[string]interface{}) (*SubmitPlayerCommitmentRequest, error) {
	req := &SubmitPlayerCommitmentRequest{}
	err := mapstructure.Decode(*data, &req)
	if err != nil {
		return nil, err
	}
	req.BaseRequest.RequestUUID = (*data)["RequestUUID"].(string)
	return req, nil
}

func NewSubmitPlayerCommitmentResponse(sessionId string) *SubmitPlayerCommitmentResponse {
	return &SubmitPlayerCommitmentResponse{
		BaseResponse: BaseResponse{
			Action:      SUBMIT_PLAYER_COMMITMENT_LABEL + "Response",
			RequestUUID: sessionId,
		},
	}
}

func NewSubmitPlayerCommitmentTask(data *map[string]interface{}) (Task, error) {
	req, err := NewSubmitPlayerCommitmentRequest(data)
	if err != nil {
		return nil, err
	}
	task := &SubmitPlayerCommitmentTask{
		Request:  req,
		Response: NewSubmitPlayerCommitmentResponse(req.BaseRequest.RequestUUID),
	}

	validate := validator.New()
	err = validate.Struct(task.Request)
	if err != nil {
		return nil, err
	}

	return task, nil
}

func (task *SubmitPlayerCommitmentTask) Run(c *gin.Context) (Response, error) {
	// 通过 PlayerID 解析玩家地址
	profile, err := db.GetUserProfileByPlayerID(strings.TrimSpace(task.Request.PlayerID))
	if err != nil || profile == nil || profile.Address == "" {
		task.Response.BaseResponse.RetCode = 1001
		task.Response.BaseResponse.Message = "Failed to get player address by player id"
		return task.Response, nil
	}

	// 统一将地址转为小写
	tempAddress := strings.ToLower(task.Request.TempAddress)

	// 将 hex 字符串转换为 bytes
	commitmentHex := strings.TrimPrefix(task.Request.Commitment, "0x")
	commitmentBytes, err := hex.DecodeString(commitmentHex)
	if err != nil {
		task.Response.BaseResponse.RetCode = 1003
		task.Response.BaseResponse.Message = "Invalid commitment hex format: " + err.Error()
		return task.Response, nil
	}

	signatureHex := strings.TrimPrefix(task.Request.Signature, "0x")
	signatureBytes, err := hex.DecodeString(signatureHex)
	if err != nil {
		task.Response.BaseResponse.RetCode = 1003
		task.Response.BaseResponse.Message = "Invalid signature hex format: " + err.Error()
		return task.Response, nil
	}

	// 通过gRPC调用RoomServer的SubmitPlayerCommitment
	rpcClient := client.GetGlobalRpcClient()
	if rpcClient == nil {
		task.Response.BaseResponse.RetCode = 1002
		task.Response.BaseResponse.Message = "gRPC client not initialized"
		return task.Response, nil
	}

	req := &proto.SubmitPlayerCommitmentRequest{
		GameID:      task.Request.GameID,
		RoundNumber: task.Request.RoundNumber,
		TurnNumber:  task.Request.TurnNumber,
		Commitment:  commitmentBytes,
		Signature:   signatureBytes,
		Address: &proto.PlayerAddress{
			Id:               profile.PlayerID,
			TemporaryAddress: tempAddress,
		},
	}

	_, err = rpcClient.SubmitPlayerCommitment(context.Background(), req)
	if err != nil {
		task.Response.BaseResponse.RetCode = 1002
		task.Response.BaseResponse.Message = "RoomServer SubmitPlayerCommitment failed: " + err.Error()
		return task.Response, nil
	}

	task.Response.BaseResponse.RetCode = 0
	task.Response.BaseResponse.Message = "Successfully submitted player commitment"
	return task.Response, nil
}
