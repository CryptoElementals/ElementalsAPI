package api

import (
	"net/http"
	"time"

	"github.com/CryptoElementals/common/log"
	"github.com/gin-contrib/sessions"
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

	userID, err := getUserIdByRefreshToken(refreshToken)
	if err != nil {
		log.Errorf("get user id by refresh token failed, refreshToken: %s, err: %v", refreshToken, err)
		return nil, err
	}

	// 设置 session cookie
	// 注意：跨域请求需要设置 SameSite=None 和 Secure=true
	session := sessions.Default(c)
	session.Options(sessions.Options{
		MaxAge:   globalSessionMaxAge,
		Path:     "/",
		SameSite: http.SameSiteNoneMode, // 允许跨域请求携带 cookie
		Secure:   true,                  // HTTPS 必需
		HttpOnly: true,                  // 防止 XSS 攻击
	})
	session.Set(SESSION_USER_KEY, userID)
	err = session.Save()
	if err != nil {
		log.Errorf("%s, save session failed, err: %v", task.Request.RequestUUID, err)
		// 即使保存 session 失败，也继续返回 refreshToken
	}

	task.Response.RefreshToken = refreshToken
	task.Response.RefreshTokenExpirationTime = int64(globalRefreshTokenMaxAge) + time.Now().Unix()
	return task.Response, nil
}
