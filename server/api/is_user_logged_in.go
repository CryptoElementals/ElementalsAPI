package api

import (
	"github.com/CryptoElementals/common/log"
	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
	"github.com/mitchellh/mapstructure"
)

func init() {
	Register(IS_USER_LOGGED_IN_LABEL, NewIsUserLoggedInTask, VERIFYAUTH)
}

type IsUserLoggedInRequest struct {
	BaseRequest
	RefreshToken string `mapstructure:"RefreshToken" validate:"required"`
}

type IsUserLoggedInResponse struct {
	BaseResponse
	UserLoggedIn bool   `json:"UserLoggedIn"`
	Address      string `json:"Address,omitempty"`
	Email        string `json:"Email,omitempty"`
}

type IsUserLoggedInTask struct {
	Request  *IsUserLoggedInRequest
	Response *IsUserLoggedInResponse
}

// 将 map 类型的数据解码为 IsUserLoggedInRequest 结构体，并提取 RequestUUID
func NewIsUserLoggedInRequest(data *map[string]interface{}) (*IsUserLoggedInRequest, error) {
	req := &IsUserLoggedInRequest{}
	err := mapstructure.Decode(*data, &req)
	if err != nil {
		return nil, err
	}
	req.BaseRequest.RequestUUID = (*data)["RequestUUID"].(string)

	return req, nil
}

func NewIsUserLoggedInResponse(sessionId string) *IsUserLoggedInResponse {
	return &IsUserLoggedInResponse{
		BaseResponse: BaseResponse{
			Action:      IS_USER_LOGGED_IN_LABEL + "Response",
			RequestUUID: sessionId,
		},
	}
}

func NewIsUserLoggedInTask(data *map[string]interface{}) (Task, error) {
	req, err := NewIsUserLoggedInRequest(data)
	if err != nil {
		return nil, err
	}
	task := &IsUserLoggedInTask{
		Request:  req,
		Response: NewIsUserLoggedInResponse(req.BaseRequest.RequestUUID),
	}

	validate := validator.New()
	err = validate.Struct(task.Request)
	if err != nil {
		return nil, err
	}

	return task, nil
}

func (task *IsUserLoggedInTask) Run(c *gin.Context) (Response, error) {
	refreshToken := task.Request.RefreshToken

	// 验证 RefreshToken 是否有效
	user, err := getUserByRefreshToken(refreshToken)
	if err != nil {
		// RefreshToken 无效或过期
		log.Infof("%s, refresh token invalid or expired: %s", task.Request.RequestUUID, refreshToken)
		task.Response.UserLoggedIn = false
		task.Response.Address = ""
		task.Response.Email = ""
		return task.Response, nil
	}

	// RefreshToken 有效，钱包/邮箱已登录
	log.Infof("%s, user logged in: %s/%s", task.Request.RequestUUID, user.Type, user.Address)
	task.Response.UserLoggedIn = true
	task.Response.Address = user.Address
	task.Response.Email = user.Email
	return task.Response, nil
}
