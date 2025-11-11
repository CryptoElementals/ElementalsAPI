package api

import (
	"encoding/json"
	"time"

	"github.com/CryptoElementals/common/cache"
	"github.com/CryptoElementals/common/errors"
	"github.com/CryptoElementals/common/log"
	"github.com/gin-contrib/sessions"
	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"
	"github.com/mitchellh/mapstructure"
)

func init() {
	Register(REFRESH_LABEL, NewRefreshDillTask, VERIFYAUTH)
}

type RefreshDillRequest struct {
	BaseRequest
	RefreshToken string
}

type RefreshDillResponse struct {
	BaseResponse
	RefreshToken               string
	RefreshTokenExpirationTime int64 // timestamp
}

type RefreshDillTask struct {
	Request  *RefreshDillRequest
	Response *RefreshDillResponse
}

// 将 map 类型的数据解码为 LoginDillRequest 结构体，并提取 RequestUUID
func NewRefreshDillRequest(data *map[string]interface{}) (*RefreshDillRequest, error) {
	req := &RefreshDillRequest{}
	err := mapstructure.Decode(*data, &req)
	if err != nil {
		return nil, err
	}
	req.BaseRequest.RequestUUID = (*data)["RequestUUID"].(string)

	return req, nil
}

func NewRefreshDillResponse(sessionId string) *RefreshDillResponse {
	return &RefreshDillResponse{
		BaseResponse: BaseResponse{
			Action:      REFRESH_LABEL + "Response",
			RequestUUID: sessionId,
		},
	}
}

func NewRefreshDillTask(data *map[string]interface{}) (Task, error) {
	req, err := NewRefreshDillRequest(data)
	if err != nil {
		return nil, err
	}
	task := &RefreshDillTask{
		Request:  req,
		Response: NewRefreshDillResponse(req.BaseRequest.RequestUUID), //respose里加上request的uuid，与cookieValue两回事
	}

	validate := validator.New()
	err = validate.Struct(task.Request)
	if err != nil {
		return nil, err
	}

	return task, nil
}

func (task *RefreshDillTask) Run(c *gin.Context) (Response, error) {
	// 验证 nonce 是否存在于 Session 中
	session := sessions.Default(c)
	session.Options(sessions.Options{
		MaxAge: globalSessionMaxAge,
	})

	refreshToken := task.Request.RefreshToken
	user, err := getUserByRefreshToken(refreshToken)
	if err != nil {
		return nil, err
	}

	// 续期 refresh token
	userJSON, _ := json.Marshal(user)
	err = globalRefreshTokenCache.Set(refreshToken, string(userJSON), globalRefreshTokenMaxAge)
	if err != nil {
		log.Errorf("update refresh failed, err: %s", err.Error())
	}
	//2 写入会话 user
	session.Set(SESSION_USER_KEY, string(userJSON))
	err = session.Save()
	if err != nil {
		log.Errorf("%s, delete nonce from session failed, %s", task.Request.RequestUUID, err.Error())
	}
	task.Response.RefreshToken = refreshToken
	task.Response.RefreshTokenExpirationTime = int64(globalRefreshTokenMaxAge) + time.Now().Unix()
	return task.Response, nil
}

func SaveRefreshTokenForUser(user *LoginUser) (string, error) {
	token := uuid.NewString()
	if _, err := globalRefreshTokenCache.Exist(token); err == nil {
		return "", errors.SaveRefreshTokenFailed()
	} else if err != cache.ErrNotFound {
		log.Errorf("get refresh token failed: %s", err.Error())
		return "", errors.SaveRefreshTokenFailed()
	}
	b, err := json.Marshal(user)
	if err != nil {
		return "", errors.SaveRefreshTokenFailed()
	}
	if err := globalRefreshTokenCache.Set(token, string(b), globalRefreshTokenMaxAge); err != nil {
		return "", err
	}
	return token, nil
}

func getUserByRefreshToken(token string) (*LoginUser, error) {
	res, err := globalRefreshTokenCache.Get(token)
	if err == cache.ErrNotFound || res == "" {
		return nil, errors.RefreshTokenInvalid(token)
	}
	if err != nil {
		log.Errorf("get user by refresh token failed: %s", err.Error())
		return nil, errors.ServiceUnavailable()
	}
	var u LoginUser
	if err := json.Unmarshal([]byte(res), &u); err != nil {
		log.Errorf("unmarshal user by refresh token failed: %s", err.Error())
		return nil, errors.ServiceUnavailable()
	}
	return &u, nil
}
