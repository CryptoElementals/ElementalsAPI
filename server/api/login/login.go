package login

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/CryptoElementals/common/errors"
	"github.com/CryptoElementals/common/log"
	"github.com/CryptoElementals/common/server/api"
	"github.com/CryptoElementals/common/wallet"
	"github.com/ethereum/go-ethereum/common"
	"github.com/gin-contrib/sessions"
	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
	"github.com/mitchellh/mapstructure"
)

const codeTemplate = `Welcome to DILL!

This request will not trigger a blockchain transaction or cost any gas fees. It is only used to authorise logging into DILL.

Your authentication status will reset after 12 hours.

Wallet address:
ADDRESS

Nonce:
NONCE`

const (
	GET_LOGIN_CODE_LABEL = "GetLoginCode"
	LOGIN_DILL_LABEL     = "LoginDill"
	REFRESH_LABEL        = "Refresh"
	SESSION_ADDR_KEY     = "addr"
)

var globalSessionMaxAge int
var globalRefreshTokenMaxAge int

func SetTokenExpire(sessionMaxAge, refreshTokenMaxAge int) {
	globalSessionMaxAge = sessionMaxAge
	globalRefreshTokenMaxAge = refreshTokenMaxAge
}

func init() {
	api.Register(GET_LOGIN_CODE_LABEL, NewGetLoginCodeTask, api.NOAUTH)
	api.Register(LOGIN_DILL_LABEL, NewLoginDillTask, api.VERIFYAUTH)
	api.Register(REFRESH_LABEL, NewRefreshDillTask, api.VERIFYAUTH)
}

type LoginDillRequest struct {
	api.BaseRequest
	Signature string `mapstructure:"Signature" validate:"required"`
	Address   string `mapstructure:"Address" validate:"required"`
	Nonce     int    `mapstructure:"Nonce" validate:"required"`
}

type LoginDillResponse struct {
	api.BaseResponse
	RefreshToken          string
	RefreshTokenExpiresIn int // by second
}

type LoginDillTask struct {
	Request  *LoginDillRequest
	Response *LoginDillResponse
}

// 将 map 类型的数据解码为 LoginDillRequest 结构体，并提取 RequestUUID
func NewLoginDillRequest(data *map[string]interface{}) (*LoginDillRequest, error) {
	req := &LoginDillRequest{}
	err := mapstructure.Decode(*data, &req)
	if err != nil {
		return nil, err
	}
	req.BaseRequest.RequestUUID = (*data)["RequestUUID"].(string)

	return req, nil
}

func NewLoginDillResponse(sessionId string) *LoginDillResponse {
	return &LoginDillResponse{
		BaseResponse: api.BaseResponse{
			Action:      LOGIN_DILL_LABEL + "Response",
			RequestUUID: sessionId,
		},
	}
}

func NewLoginDillTask(data *map[string]interface{}) (api.Task, error) {
	req, err := NewLoginDillRequest(data)
	if err != nil {
		return nil, err
	}
	task := &LoginDillTask{
		Request:  req,
		Response: NewLoginDillResponse(req.BaseRequest.RequestUUID), //respose里加上request的uuid，与cookieValue两回事
	}

	validate := validator.New()
	err = validate.Struct(task.Request)
	if err != nil {
		return nil, err
	}

	return task, nil
}

func (task *LoginDillTask) Run(c *gin.Context) (api.Response, error) {
	// 验证 nonce 是否存在于 Session 中
	session := sessions.Default(c)
	session.Options(sessions.Options{
		MaxAge: globalSessionMaxAge,
	})
	key := api.MakeAddrNonceKey(task.Request.Address)
	v := session.Get(key)
	if v == nil {
		log.Errorf("%s, key %s has no nonce", task.Request.RequestUUID, key)
		myErr := errors.AddrHasNoNonce(task.Request.Address)
		task.Response.SetRetCode(int(myErr.Code()))
		task.Response.SetMessage(myErr.String())
		return task.Response, nil
	}
	nonce := v.(int)

	if nonce != task.Request.Nonce {
		log.Errorf("%s, nonce %s != task.Request.Nonce %s", task.Request.RequestUUID, nonce, task.Request.Nonce)
		myErr := errors.NonceInvalid(strconv.Itoa(task.Request.Nonce))
		task.Response.SetRetCode(int(myErr.Code()))
		task.Response.SetMessage(myErr.String())
		return task.Response, nil
	}

	//1 verify signature
	//构造一个签名验证用的原始消息（message），用来验证用户提交的签名是否有效
	data := strings.ReplaceAll(codeTemplate, "ADDRESS", task.Request.Address)
	data = strings.ReplaceAll(data, "NONCE", strconv.Itoa(task.Request.Nonce))

	// 验证签名是否合法
	ok, err := verifySign(data, task.Request.Signature, task.Request.Address)
	if err != nil || !ok {
		if err != nil {
			log.Errorf("%s, verifySign failed, err: %s", task.Request.RequestUUID, err.Error())
		} else {
			log.Errorf("%s, verifySign failed, ok: %v", task.Request.RequestUUID, ok)
		}
		task.Response.BaseResponse.RetCode = int(errors.SignatureInvalid().Code())
		task.Response.BaseResponse.Message = errors.SignatureInvalid().Message()
		return task.Response, err
	}
	var refreshToken string
	//2 generate refresh token
	err = withRetry(10, func(retryTime int) error {
		return saveRefreshToken(task.Request.Address)
	})
	if err != nil {
		log.Errorf("save refresh token failed, err: %v", err)
		task.Response.BaseResponse.RetCode = int(errors.SaveRefreshTokenFailed().Code())
		task.Response.BaseResponse.Message = errors.SaveRefreshTokenFailed().Message()
		return task.Response, err
	}

	//3 generate session object
	err = saveSession(task.Request.RequestUUID, task.Request.Address, session)
	if err != nil {
		log.Errorf("save access token failed, err: %v", err)
		task.Response.BaseResponse.RetCode = int(errors.SaveSessionFailed().Code())
		task.Response.BaseResponse.Message = errors.SaveSessionFailed().Message()
		return task.Response, err
	}

	// 删除一次性的nonce，退出当前登录需要重新生成nonce，之前的session-id无法再使用，会慢慢过期
	session.Delete(key)
	err = session.Save()
	if err != nil {
		log.Errorf("%s, delete nonce from session failed, %s", task.Request.RequestUUID, err.Error())
	}
	task.Response.RefreshToken = refreshToken
	task.Response.RefreshTokenExpiresIn = globalRefreshTokenMaxAge
	return task.Response, nil
}

func verifySign(message string, signature string, addr string) (bool, error) {
	signatureBytes := common.Hex2Bytes(strings.TrimPrefix(signature, "0x"))
	addrBytes := common.Hex2Bytes(strings.TrimPrefix(addr, "0x"))
	return wallet.EthVerify(message, signatureBytes, addrBytes)
}

func saveSession(requestID, addr string, session sessions.Session) error {
	if addr != "" {
		session.Set(SESSION_ADDR_KEY, addr)
	}
	//把 session 写入redis
	err := session.Save()
	if err != nil {
		return fmt.Errorf("%s, save cookie-address to session failed, %s", requestID, err.Error())
	}
	return nil
}

func withRetry(retryCount int, do func(retryTime int) error) error {
	var err error
	for i := 0; i < retryCount; i++ {
		err = do(i)
		if err == nil {
			return nil
		}
	}
	// return the last error
	return err
}
