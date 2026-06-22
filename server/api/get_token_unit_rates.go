package api

import (
	"context"

	"github.com/CryptoElementals/common/errors"
	"github.com/CryptoElementals/common/rpc/client"
	"github.com/gin-gonic/gin"
	"github.com/mitchellh/mapstructure"
)

func init() {
	Register(GET_TOKEN_UNIT_RATES_LABEL, NewGetTokenUnitRatesTask, NOAUTH)
}

type GetTokenUnitRatesRequest struct {
	BaseRequest
}

type MulDivRateDTO struct {
	Mul string `json:"Mul"`
	Div string `json:"Div"`
}

type GetTokenUnitRatesResponse struct {
	BaseResponse
	TokenToUsdt MulDivRateDTO `json:"TokenToUsdt"`
	UsdtToWei   MulDivRateDTO `json:"UsdtToWei"`
	TokenToWei  MulDivRateDTO `json:"TokenToWei"`
	UsdtToToken MulDivRateDTO `json:"UsdtToToken"`
	WeiToUsdt   MulDivRateDTO `json:"WeiToUsdt"`
	WeiToToken  MulDivRateDTO `json:"WeiToToken"`
}

type GetTokenUnitRatesTask struct {
	Request  *GetTokenUnitRatesRequest
	Response *GetTokenUnitRatesResponse
}

func NewGetTokenUnitRatesRequest(data *map[string]interface{}) (*GetTokenUnitRatesRequest, error) {
	req := &GetTokenUnitRatesRequest{}
	if err := mapstructure.Decode(*data, &req); err != nil {
		return nil, err
	}
	req.BaseRequest.RequestUUID = (*data)["RequestUUID"].(string)
	return req, nil
}

func NewGetTokenUnitRatesResponse(sessionID string) *GetTokenUnitRatesResponse {
	return &GetTokenUnitRatesResponse{
		BaseResponse: BaseResponse{
			Action:      GET_TOKEN_UNIT_RATES_LABEL + "Response",
			RequestUUID: sessionID,
		},
	}
}

func NewGetTokenUnitRatesTask(data *map[string]interface{}) (Task, error) {
	req, err := NewGetTokenUnitRatesRequest(data)
	if err != nil {
		return nil, err
	}
	return &GetTokenUnitRatesTask{
		Request:  req,
		Response: NewGetTokenUnitRatesResponse(req.BaseRequest.RequestUUID),
	}, nil
}

func (task *GetTokenUnitRatesTask) Run(c *gin.Context) (Response, error) {
	resp, err := client.GetTokenUnitRates(context.Background(), ServerTypeFromGin(c))
	if err != nil {
		return nil, errors.ActionError(err.Error())
	}
	task.Response.TokenToUsdt = mulDivRateDTO(resp.GetTokenToUsdt())
	task.Response.UsdtToWei = mulDivRateDTO(resp.GetUsdtToWei())
	task.Response.TokenToWei = mulDivRateDTO(resp.GetTokenToWei())
	task.Response.UsdtToToken = mulDivRateDTO(resp.GetUsdtToToken())
	task.Response.WeiToUsdt = mulDivRateDTO(resp.GetWeiToUsdt())
	task.Response.WeiToToken = mulDivRateDTO(resp.GetWeiToToken())
	return task.Response, nil
}

func mulDivRateDTO(rate interface {
	GetMul() string
	GetDiv() string
}) MulDivRateDTO {
	if rate == nil {
		return MulDivRateDTO{}
	}
	return MulDivRateDTO{Mul: rate.GetMul(), Div: rate.GetDiv()}
}
