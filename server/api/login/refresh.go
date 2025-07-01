package login

import (
	"github.com/CryptoElementals/common/errors"
	"github.com/CryptoElementals/common/log"
	"github.com/CryptoElementals/common/redis"
	"github.com/CryptoElementals/common/server/api"
	"github.com/gin-contrib/sessions"
	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"
	"github.com/mitchellh/mapstructure"
)

type RefreshDillRequest struct {
	api.BaseRequest
	RefreshToken string
}

type RefreshDillResponse struct {
	api.BaseResponse
	RefreshToken          string
	RefreshTokenExpiresIn int // by second
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
		BaseResponse: api.BaseResponse{
			Action:      REFRESH_LABEL + "Response",
			RequestUUID: sessionId,
		},
	}
}

func NewRefreshDillTask(data *map[string]interface{}) (api.Task, error) {
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

func (task *RefreshDillTask) Run(c *gin.Context) (api.Response, error) {
	// 验证 nonce 是否存在于 Session 中
	session := sessions.Default(c)
	session.Options(sessions.Options{
		MaxAge: globalSessionMaxAge,
	})

	refreshToken := task.Request.RefreshToken
	addr, err := getAddrByRefreshToken(refreshToken)
	if err != nil {
		return nil, err
	}

	err = redis.SetWithExpire(refreshToken, addr, globalRefreshTokenMaxAge)
	if err != nil {
		log.Errorf("update refresh failed, err: %s", err.Error())
	}
	//2 generate session
	session.Set(SESSION_ADDR_KEY, addr)
	err = session.Save()
	if err != nil {
		log.Errorf("%s, delete nonce from session failed, %s", task.Request.RequestUUID, err.Error())
	}
	task.Response.RefreshToken = refreshToken
	task.Response.RefreshTokenExpiresIn = globalRefreshTokenMaxAge
	return task.Response, nil
}

func saveRefreshToken(addr string) (string, error) {
	token := uuid.NewString()
	_, err := redis.Exist(token)
	if err == nil {
		return "", errors.SaveRefreshTokenFailed()
	}
	if err != redis.ErrNotFound {
		log.Errorf("get refresh token failed: %s", err.Error())
		return "", errors.SaveRefreshTokenFailed()
	}

	err = redis.SetWithExpire(token, addr, globalRefreshTokenMaxAge)
	return token, err
}

func getAddrByRefreshToken(token string) (string, error) {
	res, err := redis.Get(token)
	if err == redis.ErrNotFound || res == "" {
		return "", errors.RefreshTokenInvalid(token)
	}
	if err != nil {
		log.Errorf("get addr by refresh token failed: %s", err.Error())
		return "", errors.ServiceUnavailable()
	}
	return res, nil
}
