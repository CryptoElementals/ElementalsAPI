package api

import (
	"context"
	"encoding/hex"
	"strconv"
	"strings"

	"github.com/CryptoElementals/common/errors"
	"github.com/CryptoElementals/common/rpc/client"
	"github.com/CryptoElementals/common/rpc/proto"
	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
	"github.com/mitchellh/mapstructure"
)

func init() {
	Register(REQUEST_WITHDRAW_LABEL, NewRequestWithdrawTask, COOKIEAUTH)
}

type RequestWithdrawRequest struct {
	BaseRequest
	PlayerID    string `mapstructure:"PlayerID" validate:"required"`
	TokenAmount int32  `mapstructure:"TokenAmount" validate:"min=1"`
	Signature   string `mapstructure:"Signature" validate:"required"`
}

type RequestWithdrawResponse struct {
	BaseResponse
	RequestID        string `json:"RequestID"`
	TxHash           string `json:"TxHash"`
	CollectorAddress string `json:"CollectorAddress"`
	LedgerID         uint64 `json:"LedgerID"`
	Status           string `json:"Status"`
}

type RequestWithdrawTask struct {
	Request  *RequestWithdrawRequest
	Response *RequestWithdrawResponse
}

func NewRequestWithdrawRequest(data *map[string]interface{}) (*RequestWithdrawRequest, error) {
	req := &RequestWithdrawRequest{}
	if err := mapstructure.Decode(*data, &req); err != nil {
		return nil, err
	}
	req.BaseRequest.RequestUUID = (*data)["RequestUUID"].(string)
	return req, nil
}

func NewRequestWithdrawResponse(sessionID string) *RequestWithdrawResponse {
	return &RequestWithdrawResponse{
		BaseResponse: BaseResponse{
			Action:      REQUEST_WITHDRAW_LABEL + "Response",
			RequestUUID: sessionID,
		},
	}
}

func NewRequestWithdrawTask(data *map[string]interface{}) (Task, error) {
	req, err := NewRequestWithdrawRequest(data)
	if err != nil {
		return nil, err
	}
	task := &RequestWithdrawTask{
		Request:  req,
		Response: NewRequestWithdrawResponse(req.BaseRequest.RequestUUID),
	}
	validate := validator.New()
	if err := validate.Struct(task.Request); err != nil {
		return nil, err
	}
	return task, nil
}

func (task *RequestWithdrawTask) Run(c *gin.Context) (Response, error) {
	playerID, err := strconv.ParseInt(strings.TrimSpace(task.Request.PlayerID), 10, 64)
	if err != nil || playerID <= 0 {
		return nil, errors.ParamsJudgeError("invalid player id")
	}

	sigBytes, err := decodeHexSignature(task.Request.Signature)
	if err != nil {
		return nil, errors.ParamsJudgeError(err.Error())
	}

	resp, err := client.RequestWithdraw(context.Background(), ServerTypeFromGin(c), &proto.RequestWithdrawRequest{
		PlayerId:    playerID,
		TokenAmount: task.Request.TokenAmount,
		Signature:   sigBytes,
	})
	if err != nil {
		return nil, errors.ActionError(err.Error())
	}

	task.Response.RequestID = resp.GetRequestId()
	task.Response.TxHash = resp.GetTxHash()
	task.Response.CollectorAddress = resp.GetCollectorAddress()
	task.Response.LedgerID = resp.GetLedgerId()
	task.Response.Status = resp.GetStatus()
	return task.Response, nil
}

func decodeHexSignature(sig string) ([]byte, error) {
	raw := strings.TrimSpace(sig)
	raw = strings.TrimPrefix(raw, "0x")
	if raw == "" {
		return nil, errors.ParamsJudgeError("signature is empty")
	}
	b, err := hex.DecodeString(raw)
	if err != nil {
		return nil, errors.ParamsJudgeError("invalid signature hex")
	}
	return b, nil
}
