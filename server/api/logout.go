package api

import (
	"github.com/CryptoElementals/common/errors"
	"github.com/CryptoElementals/common/log"
	"github.com/gin-contrib/sessions"
	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
	"github.com/mitchellh/mapstructure"
)

func init() {
	Register(LOGOUT_LABEL, NewLogoutTask, COOKIEAUTH)
}

type LogoutRequest struct {
	BaseRequest
	RefreshToken string `mapstructure:"RefreshToken" validate:"required"` // 必需：要删除的 refreshToken
}

type LogoutResponse struct {
	BaseResponse
}

type LogoutTask struct {
	Request  *LogoutRequest
	Response *LogoutResponse
}

func NewLogoutRequest(data *map[string]interface{}) (*LogoutRequest, error) {
	req := &LogoutRequest{}
	err := mapstructure.Decode(*data, &req)
	if err != nil {
		return nil, err
	}
	req.BaseRequest.RequestUUID = (*data)["RequestUUID"].(string)
	return req, nil
}

func NewLogoutResponse(sessionId string) *LogoutResponse {
	return &LogoutResponse{
		BaseResponse: BaseResponse{
			Action:      LOGOUT_LABEL + "Response",
			RequestUUID: sessionId,
		},
	}
}

func NewLogoutTask(data *map[string]interface{}) (Task, error) {
	req, err := NewLogoutRequest(data)
	if err != nil {
		return nil, err
	}
	task := &LogoutTask{
		Request:  req,
		Response: NewLogoutResponse(req.BaseRequest.RequestUUID),
	}

	validate := validator.New()
	err = validate.Struct(task.Request)
	if err != nil {
		return nil, err
	}

	return task, nil
}

func (task *LogoutTask) Run(c *gin.Context) (Response, error) {
	session := sessions.Default(c)
	
	// 从 session 中获取 playerID
	playerIDStr := session.Get(SESSION_USER_KEY)
	if playerIDStr == nil {
		// 如果没有 session，说明已经退出或未登录
		log.Infof("%s, logout called but no session found", task.Request.RequestUUID)
		return task.Response, nil
	}

	playerID := playerIDStr.(string)

	// 验证该 refreshToken 是否有效
	userID, err := getUserIdByRefreshToken(task.Request.RefreshToken)
	if err != nil {
		log.Errorf("%s, refresh token invalid or expired: %s, error: %v", task.Request.RequestUUID, task.Request.RefreshToken, err)
		// Token 无效或已过期，返回错误
		return nil, err
	}

	// 验证 token 对应的 playerID 是否与 cookie 中的 playerID 一致
	if userID != playerID {
		log.Errorf("%s, refresh token does not belong to current user: cookie player_id=%s, token player_id=%s", 
			task.Request.RequestUUID, playerID, userID)
		return nil, errors.LoginCookieInvalid("refresh token does not belong to current user")
	}

	// 删除指定的 refreshToken
	err = globalRefreshTokenCache.Delete(task.Request.RefreshToken)
	if err != nil {
		log.Errorf("%s, failed to delete refresh token: %s", task.Request.RequestUUID, err.Error())
		return nil, errors.ServiceUnavailable()
	}
	log.Infof("%s, deleted refresh token for player_id: %s", task.Request.RequestUUID, playerID)

	// 清除 session
	session.Clear()
	err = session.Save()
	if err != nil {
		log.Errorf("%s, failed to clear session: %s", task.Request.RequestUUID, err.Error())
		return nil, errors.ServiceUnavailable()
	}

	log.Infof("%s, logout successful for player_id: %s", task.Request.RequestUUID, playerID)
	return task.Response, nil
}

