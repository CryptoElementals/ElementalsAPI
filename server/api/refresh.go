package api

import (
	"net/http"
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
		MaxAge:   globalSessionMaxAge,
		Path:     "/",
		SameSite: http.SameSiteNoneMode, // 允许跨域请求携带 cookie
		Secure:   true,                  // HTTPS 必需
		HttpOnly: true,                  // 防止 XSS 攻击
	})

	refreshToken := task.Request.RefreshToken
	userID, err := getUserIdByRefreshToken(refreshToken)
	if err != nil {
		return nil, err
	}

	// 续期 refresh token
	err = globalRefreshTokenCache.Set(refreshToken, userID, globalRefreshTokenMaxAge)
	if err != nil {
		log.Errorf("update refresh failed, err: %s", err.Error())
	}
	//2 写入会话 user
	session.Set(SESSION_USER_KEY, userID)
	err = session.Save()
	if err != nil {
		log.Errorf("%s, delete nonce from session failed, %s", task.Request.RequestUUID, err.Error())
	}
	task.Response.RefreshToken = refreshToken
	task.Response.RefreshTokenExpirationTime = int64(globalRefreshTokenMaxAge) + time.Now().Unix()
	return task.Response, nil
}

func SaveRefreshTokenForUserId(userID string) (string, error) {
	token := uuid.NewString()
	if _, err := globalRefreshTokenCache.Exist(token); err == nil {
		return "", errors.SaveRefreshTokenFailed()
	} else if err != cache.ErrNotFound {
		log.Errorf("get refresh token failed: %s", err.Error())
		return "", errors.SaveRefreshTokenFailed()
	}
	if err := globalRefreshTokenCache.Set(token, userID, globalRefreshTokenMaxAge); err != nil {
		return "", err
	}
	return token, nil
}

// SaveTempCodeForRefreshToken 生成一个短期 code，并保存 code->refresh_token 的映射（ttl 秒）
func SaveTempCodeForRefreshToken(refreshToken string, ttl int) (string, error) {
	code := uuid.NewString()
	key := "code:" + code
	if err := globalRefreshTokenCache.Set(key, refreshToken, ttl); err != nil {
		return "", err
	}
	return code, nil
}

// ExchangeRefreshTokenByCode 根据 code 找到 refresh_token，并删除该映射
func ExchangeRefreshTokenByCode(code string) (string, error) {
	key := "code:" + code
	val, err := globalRefreshTokenCache.Get(key)
	if err == cache.ErrNotFound || val == "" {
		return "", errors.RefreshTokenInvalid(code)
	}
	if err != nil {
		log.Errorf("exchange code failed, err: %s", err.Error())
		return "", errors.ServiceUnavailable()
	}
	_ = globalRefreshTokenCache.Delete(key)
	return val, nil
}

func getUserIdByRefreshToken(token string) (string, error) {
	res, err := globalRefreshTokenCache.Get(token)
	if err == cache.ErrNotFound || res == "" {
		return "", errors.RefreshTokenInvalid(token)
	}
	if err != nil {
		log.Errorf("get user by refresh token failed: %s", err.Error())
		return "", errors.ServiceUnavailable()
	}
	return res, nil
}
