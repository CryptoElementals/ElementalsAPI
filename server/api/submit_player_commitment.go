package api

import (
	"context"
	"encoding/hex"
	"strconv"
	"strings"

	"github.com/CryptoElementals/common/log"
	"github.com/CryptoElementals/common/rpc/client"
	"github.com/CryptoElementals/common/rpc/proto"
	"github.com/ethereum/go-ethereum/crypto"
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
	GameID      int64  `mapstructure:"GameID" validate:"required"`
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
	log.Infof("%s, SubmitPlayerCommitment request: GameID=%d, RoundNumber=%d, TurnNumber=%d, TempAddress=%s, PlayerID=%s",
		task.Request.RequestUUID, task.Request.GameID, task.Request.RoundNumber, task.Request.TurnNumber,
		task.Request.TempAddress, task.Request.PlayerID)

	// 解析 PlayerID（由中间件从会话中注入），前端只需要传临时地址
	playerIDStr := strings.TrimSpace(task.Request.PlayerID)
	if playerIDStr == "" {
		log.Errorf("%s, player id is empty", task.Request.RequestUUID)
		task.Response.BaseResponse.RetCode = 1001
		task.Response.BaseResponse.Message = "player id is empty"
		return task.Response, nil
	}
	playerID, err := strconv.ParseInt(playerIDStr, 10, 64)
	if err != nil {
		log.Errorf("%s, invalid player id: %v", task.Request.RequestUUID, err)
		task.Response.BaseResponse.RetCode = 1001
		task.Response.BaseResponse.Message = "invalid player id"
		return task.Response, nil
	}

	// 统一将地址转为小写
	tempAddress := strings.ToLower(task.Request.TempAddress)

	// 将 hex 字符串转换为 bytes
	commitmentHex := strings.TrimPrefix(task.Request.Commitment, "0x")
	commitmentBytes, err := hex.DecodeString(commitmentHex)
	if err != nil {
		log.Errorf("%s, Invalid commitment hex format: %v, commitment=%s", task.Request.RequestUUID, err, task.Request.Commitment)
		task.Response.BaseResponse.RetCode = 1003
		task.Response.BaseResponse.Message = "Invalid commitment hex format: " + err.Error()
		return task.Response, nil
	}

	log.Infof("%s, Commitment decoded: length=%d, hex=%x", task.Request.RequestUUID, len(commitmentBytes), commitmentBytes)

	signatureHex := strings.TrimPrefix(task.Request.Signature, "0x")
	signatureBytes, err := hex.DecodeString(signatureHex)
	if err != nil {
		log.Errorf("%s, Invalid signature hex format: %v, signature=%s", task.Request.RequestUUID, err, task.Request.Signature)
		task.Response.BaseResponse.RetCode = 1003
		task.Response.BaseResponse.Message = "Invalid signature hex format: " + err.Error()
		return task.Response, nil
	}

	recoveryID := uint8(0)
	if len(signatureBytes) == crypto.SignatureLength {
		recoveryID = signatureBytes[crypto.RecoveryIDOffset]
	}
	log.Infof("%s, Signature decoded: length=%d, recovery_id=%d (raw format: 0 or 1)", task.Request.RequestUUID, len(signatureBytes), recoveryID)

	// 通过gRPC调用RoomServer的SubmitPlayerCommitment
	rpcClient := client.RoomClientForType(ServerTypeFromGin(c))
	if rpcClient == nil {
		log.Errorf("%s, gRPC client not initialized", task.Request.RequestUUID)
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
			Id:               playerID,
			TemporaryAddress: tempAddress,
		},
	}

	log.Infof("%s, Calling RoomServer SubmitPlayerCommitment: GameID=%d, RoundNumber=%d, TurnNumber=%d, CommitmentLen=%d, SignatureLen=%d, TempAddress=%s",
		task.Request.RequestUUID, req.GameID, req.RoundNumber, req.TurnNumber, len(req.Commitment), len(req.Signature), req.Address.TemporaryAddress)

	_, err = rpcClient.SubmitPlayerCommitment(context.Background(), req)
	if err != nil {
		log.Errorf("%s, RoomServer SubmitPlayerCommitment failed: %v", task.Request.RequestUUID, err)
		task.Response.BaseResponse.RetCode = 1002
		task.Response.BaseResponse.Message = "RoomServer SubmitPlayerCommitment failed: " + err.Error()
		return task.Response, nil
	}

	log.Infof("%s, Successfully submitted player commitment", task.Request.RequestUUID)
	task.Response.BaseResponse.RetCode = 0
	task.Response.BaseResponse.Message = "Successfully submitted player commitment"
	return task.Response, nil
}
