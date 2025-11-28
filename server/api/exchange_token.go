package api

import (
	"time"

	"github.com/CryptoElementals/common/log"
	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
	"github.com/mitchellh/mapstructure"
)

func init() {
	Register(EXCHANGE_TOKEN_LABEL, NewExchangeTokenTask, NOAUTH)
}

type ExchangeTokenRequest struct {
	BaseRequest
	Code string `mapstructure:"Code" validate:"required"`
}

type ExchangeTokenResponse struct {
	BaseResponse
	RefreshToken               string
	RefreshTokenExpirationTime int64
}

type ExchangeTokenTask struct {
	Request  *ExchangeTokenRequest
	Response *ExchangeTokenResponse
}

func NewExchangeTokenRequest(data *map[string]interface{}) (*ExchangeTokenRequest, error) {
	req := &ExchangeTokenRequest{}
	err := mapstructure.Decode(*data, &req)
	if err != nil {
		return nil, err
	}
	req.BaseRequest.RequestUUID = (*data)["RequestUUID"].(string)
	return req, nil
}

func NewExchangeTokenResponse(sessionId string) *ExchangeTokenResponse {
	return &ExchangeTokenResponse{
		BaseResponse: BaseResponse{
			Action:      EXCHANGE_TOKEN_LABEL + "Response",
			RequestUUID: sessionId,
		},
	}
}

func NewExchangeTokenTask(data *map[string]interface{}) (Task, error) {
	req, err := NewExchangeTokenRequest(data)
	if err != nil {
		return nil, err
	}
	task := &ExchangeTokenTask{
		Request:  req,
		Response: NewExchangeTokenResponse(req.BaseRequest.RequestUUID),
	}
	validate := validator.New()
	err = validate.Struct(task.Request)
	if err != nil {
		return nil, err
	}
	return task, nil
}

func (task *ExchangeTokenTask) Run(c *gin.Context) (Response, error) {
	code := task.Request.Code
	refreshToken, err := ExchangeRefreshTokenByCode(code)
	if err != nil {
		log.Errorf("exchange token failed, code: %s, err: %v", code, err)
		return nil, err
	}
	task.Response.RefreshToken = refreshToken
	task.Response.RefreshTokenExpirationTime = int64(globalRefreshTokenMaxAge) + time.Now().Unix()
	return task.Response, nil
}
