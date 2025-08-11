package api

import (
	"github.com/CryptoElementals/common/log"
	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
	"github.com/mitchellh/mapstructure"
)

func init() {
	Register(IS_WALLET_LOGGED_IN_LABEL, NewIsWalletLoggedInTask, VERIFYAUTH)
}

type IsWalletLoggedInRequest struct {
	BaseRequest
	RefreshToken string `mapstructure:"RefreshToken" validate:"required"`
}

type IsWalletLoggedInResponse struct {
	BaseResponse
	WalletLoggedIn bool   `json:"WalletLoggedIn"`
	Address        string `json:"Address,omitempty"`
}

type IsWalletLoggedInTask struct {
	Request  *IsWalletLoggedInRequest
	Response *IsWalletLoggedInResponse
}

// 将 map 类型的数据解码为 IsWalletLoggedInRequest 结构体，并提取 RequestUUID
func NewIsWalletLoggedInRequest(data *map[string]interface{}) (*IsWalletLoggedInRequest, error) {
	req := &IsWalletLoggedInRequest{}
	err := mapstructure.Decode(*data, &req)
	if err != nil {
		return nil, err
	}
	req.BaseRequest.RequestUUID = (*data)["RequestUUID"].(string)

	return req, nil
}

func NewIsWalletLoggedInResponse(sessionId string) *IsWalletLoggedInResponse {
	return &IsWalletLoggedInResponse{
		BaseResponse: BaseResponse{
			Action:      IS_WALLET_LOGGED_IN_LABEL + "Response",
			RequestUUID: sessionId,
		},
	}
}

func NewIsWalletLoggedInTask(data *map[string]interface{}) (Task, error) {
	req, err := NewIsWalletLoggedInRequest(data)
	if err != nil {
		return nil, err
	}
	task := &IsWalletLoggedInTask{
		Request:  req,
		Response: NewIsWalletLoggedInResponse(req.BaseRequest.RequestUUID),
	}

	validate := validator.New()
	err = validate.Struct(task.Request)
	if err != nil {
		return nil, err
	}

	return task, nil
}

func (task *IsWalletLoggedInTask) Run(c *gin.Context) (Response, error) {
	refreshToken := task.Request.RefreshToken

	// 验证 RefreshToken 是否有效
	addr, err := getAddrByRefreshToken(refreshToken)
	if err != nil {
		// RefreshToken 无效或过期
		log.Infof("%s, refresh token invalid or expired: %s", task.Request.RequestUUID, refreshToken)
		task.Response.WalletLoggedIn = false
		task.Response.Address = ""
		return task.Response, nil
	}

	// RefreshToken 有效，钱包已登录
	log.Infof("%s, wallet logged in: %s", task.Request.RequestUUID, addr)
	task.Response.WalletLoggedIn = true
	task.Response.Address = addr
	return task.Response, nil
}
