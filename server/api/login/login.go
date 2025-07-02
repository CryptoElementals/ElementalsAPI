package login

import (
	goerrs "errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/CryptoElementals/common/cache"
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

const codeTemplateTemplate = `Welcome to %s!

This request will not trigger a blockchain transaction or cost any gas fees. It is only used to authorise logging into %s.

Your authentication status will reset after 12 hours.

Wallet address:
ADDRESS

Nonce:
NONCE`

var codeTemplate = fmt.Sprintf(codeTemplateTemplate, "DILL", "DILL")

const (
	GET_LOGIN_CODE_LABEL = "GetLoginCode"
	LOGIN_DILL_LABEL     = "LoginDill"
	REFRESH_LABEL        = "Refresh"
	SESSION_ADDR_KEY     = "addr"
)

var globalSessionMaxAge int
var globalRefreshTokenMaxAge int
var globalRefreshTokenCache cache.Cache

func InitLoginApi(sessionMaxAge, refreshTokenMaxAge int, serviceName string, refreshTokenCache cache.Cache) error {
	if sessionMaxAge == 0 {
		return goerrs.New("sessionMaxAge is zero")
	}
	if refreshTokenMaxAge == 0 {
		return goerrs.New("refreshTokenMaxAge is zero")
	}
	if serviceName == "" {
		return goerrs.New("serviceName is empty")
	}
	if refreshTokenCache == nil {
		return goerrs.New("refreshTokenCache is empty")
	}
	globalSessionMaxAge = sessionMaxAge
	globalRefreshTokenMaxAge = refreshTokenMaxAge
	codeTemplate = fmt.Sprintf(codeTemplateTemplate, serviceName, serviceName)
	globalRefreshTokenCache = refreshTokenCache
	api.Register(GET_LOGIN_CODE_LABEL, NewGetLoginCodeTask, api.NOAUTH)
	api.Register(LOGIN_DILL_LABEL, NewLoginDillTask, api.VERIFYAUTH)
	api.Register(REFRESH_LABEL, NewRefreshDillTask, api.VERIFYAUTH)
	return nil
}

func GetSigningData(addr string, nonce int) string {
	data := strings.ReplaceAll(codeTemplate, "ADDRESS", addr)
	data = strings.ReplaceAll(data, "NONCE", strconv.Itoa(nonce))
	return data
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
		return nil, errors.AddrHasNoNonce(task.Request.Address)
	}
	nonce := v.(int)

	if nonce != task.Request.Nonce {
		log.Errorf("%s, nonce %s != task.Request.Nonce %s", task.Request.RequestUUID, nonce, task.Request.Nonce)
		return nil, errors.NonceInvalid(strconv.Itoa(task.Request.Nonce))
	}

	//1 verify signature
	//构造一个签名验证用的原始消息（message），用来验证用户提交的签名是否有效
	data := GetSigningData(task.Request.Address, task.Request.Nonce)

	// 验证签名是否合法
	ok, err := verifySign(data, task.Request.Signature, task.Request.Address)
	if err != nil || !ok {
		if err != nil {
			log.Errorf("%s, verifySign failed, err: %s", task.Request.RequestUUID, err.Error())
		} else {
			log.Errorf("%s, verifySign failed, ok: %v", task.Request.RequestUUID, ok)
		}
		return nil, err
	}
	var refreshToken string
	//2 generate refresh token
	err = withRetry(10, func(retryTime int) error {
		var saveErr error
		refreshToken, saveErr = saveRefreshToken(task.Request.Address)
		return saveErr
	})
	if err != nil {
		log.Errorf("save refresh token failed, err: %v", err)
		return nil, err
	}

	//3 set address to session object
	if task.Request.Address != "" {
		session.Set(SESSION_ADDR_KEY, task.Request.Address)
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
