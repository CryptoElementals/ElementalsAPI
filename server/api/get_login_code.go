package api

import (
	"math/rand"
	"strconv"
	"time"

	"regexp"

	"github.com/CryptoElementals/common/log"
	"github.com/gin-contrib/sessions"
	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
	"github.com/mitchellh/mapstructure"
)

func init() {
	Register(GET_LOGIN_CODE_LABEL, NewGetLoginCodeTask, NOAUTH)
}

type GetLoginCodeRequest struct {
	BaseRequest
	Address string `mapstructure:"Address" validate:"required"`
}

type GetLoginCodeResponse struct {
	BaseResponse

	Nonce     int    `mapstructure:"Nonce"`
	LoginCode string `mapstructure:"LoginCode"`
}

type GetLoginCodeTask struct {
	Request  *GetLoginCodeRequest
	Response *GetLoginCodeResponse
}

func NewGetLoginCodeRequest(data *map[string]interface{}) (*GetLoginCodeRequest, error) {
	req := &GetLoginCodeRequest{}
	err := mapstructure.Decode(*data, &req)
	if err != nil {
		return nil, err
	}
	req.BaseRequest.RequestUUID = (*data)["RequestUUID"].(string)

	return req, nil
}

func NewGetLoginCodeResponse(sessionId string) *GetLoginCodeResponse {
	return &GetLoginCodeResponse{
		BaseResponse: BaseResponse{
			Action:      GET_LOGIN_CODE_LABEL + "Response",
			RequestUUID: sessionId,
		},
	}
}

func NewGetLoginCodeTask(data *map[string]interface{}) (Task, error) {
	req, err := NewGetLoginCodeRequest(data)
	if err != nil {
		return nil, err
	}
	task := &GetLoginCodeTask{
		Request:  req,
		Response: NewGetLoginCodeResponse(req.BaseRequest.RequestUUID),
	}

	validate := validator.New()
	err = validate.Struct(task.Request)
	if err != nil {
		return nil, err
	}
	return task, nil
}

func (task *GetLoginCodeTask) Run(c *gin.Context) (Response, error) {
	var nonce int
	// save nonce redis, set TTL
	session := sessions.Default(c)
	key := MakeAddrNonceKey(task.Request.Address)
	v := session.Get(key)
	if v == nil {
		r := rand.New(rand.NewSource(time.Now().UTC().UnixNano()))
		nonce = r.Intn(900000) + 100000
		session.Set(key, nonce)

		// when first time to login,
		// (1) save nonce to session
		// (2) generate session id
		// (3) save session to redis
		// (4) Set-Cookie cookie with key/value pair to response headers
		err := session.Save()
		if err != nil {
			log.Errorf("%s, save nonce to session failed, %s", task.Request.RequestUUID, err.Error())
			return nil, err
		}
	} else {
		nonce = v.(int)
	}

	reAddress := regexp.MustCompile("ADDRESS")
	reNonce := regexp.MustCompile("NONCE")

	// 打印日志
	log.Infof("Key: %s, type: %T", key, key)
	log.Infof("Nonce: %d, type: %T", nonce, nonce)

	str := codeTemplate
	str = reAddress.ReplaceAllString(str, task.Request.Address)
	str = reNonce.ReplaceAllString(str, strconv.Itoa(nonce))

	task.Response.Nonce = nonce
	task.Response.LoginCode = str
	return task.Response, nil
}
