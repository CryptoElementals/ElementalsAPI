package api

import (
	"context"
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
	Register(GET_WITHDRAWABLE_TOKEN_AMOUNT_LABEL, NewGetWithdrawableTokenAmountTask, COOKIEAUTH)
}

type GetWithdrawableTokenAmountRequest struct {
	BaseRequest
	PlayerID string `mapstructure:"PlayerID" validate:"required"`
}

type GetWithdrawableTokenAmountResponse struct {
	BaseResponse
	WithdrawableTokenAmount    int32 `json:"WithdrawableTokenAmount"`
	TokenAmount                int32 `json:"TokenAmount"`
	LockedTokens               int32 `json:"LockedTokens"`
	PendingWithdrawTokenAmount int32 `json:"PendingWithdrawTokenAmount"`
}

type GetWithdrawableTokenAmountTask struct {
	Request  *GetWithdrawableTokenAmountRequest
	Response *GetWithdrawableTokenAmountResponse
}

func NewGetWithdrawableTokenAmountRequest(data *map[string]interface{}) (*GetWithdrawableTokenAmountRequest, error) {
	req := &GetWithdrawableTokenAmountRequest{}
	if err := mapstructure.Decode(*data, &req); err != nil {
		return nil, err
	}
	req.BaseRequest.RequestUUID = (*data)["RequestUUID"].(string)
	return req, nil
}

func NewGetWithdrawableTokenAmountResponse(sessionID string) *GetWithdrawableTokenAmountResponse {
	return &GetWithdrawableTokenAmountResponse{
		BaseResponse: BaseResponse{
			Action:      GET_WITHDRAWABLE_TOKEN_AMOUNT_LABEL + "Response",
			RequestUUID: sessionID,
		},
	}
}

func NewGetWithdrawableTokenAmountTask(data *map[string]interface{}) (Task, error) {
	req, err := NewGetWithdrawableTokenAmountRequest(data)
	if err != nil {
		return nil, err
	}
	task := &GetWithdrawableTokenAmountTask{
		Request:  req,
		Response: NewGetWithdrawableTokenAmountResponse(req.BaseRequest.RequestUUID),
	}
	validate := validator.New()
	if err := validate.Struct(task.Request); err != nil {
		return nil, err
	}
	return task, nil
}

func (task *GetWithdrawableTokenAmountTask) Run(c *gin.Context) (Response, error) {
	playerID, err := strconv.ParseInt(strings.TrimSpace(task.Request.PlayerID), 10, 64)
	if err != nil || playerID <= 0 {
		return nil, errors.ParamsJudgeError("invalid player id")
	}

	resp, err := client.GetWithdrawableTokenAmount(context.Background(), ServerTypeFromGin(c), &proto.GetWithdrawableTokenAmountRequest{
		PlayerId: playerID,
	})
	if err != nil {
		return nil, errors.ActionError(err.Error())
	}

	task.Response.WithdrawableTokenAmount = resp.GetWithdrawableTokenAmount()
	task.Response.TokenAmount = resp.GetTokenAmount()
	task.Response.LockedTokens = resp.GetLockedTokens()
	task.Response.PendingWithdrawTokenAmount = resp.GetPendingWithdrawTokenAmount()
	return task.Response, nil
}
