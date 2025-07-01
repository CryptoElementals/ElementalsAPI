package login

import (
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/CryptoElementals/common/errors"
	"github.com/CryptoElementals/common/log"
	"github.com/CryptoElementals/common/server/api"
	"github.com/CryptoElementals/common/wallet"
	"github.com/ethereum/go-ethereum/common"
	"github.com/gin-contrib/sessions"
	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"
	"github.com/mitchellh/mapstructure"
)

const codeTemplate = "Welcome to DILL!\n\nThis request will not trigger a blockchain transaction or cost any gas fees. It is only used to authorise logging into DILL.\n\nYour authentication status will reset after 12 hours.\n\nWallet address:\nADDRESS\n\nNonce:\nNONCE"

const (
	GET_LOGIN_CODE_LABEL = "GetLoginCode"
	LOGIN_DILL_LABEL     = "LoginDill"
)

var globalSessionMaxAge int

func UseWalletLogin(sessionMaxAge int) {
	globalSessionMaxAge = sessionMaxAge
	api.Register(GET_LOGIN_CODE_LABEL, NewGetLoginCodeTask, api.NOAUTH)
	api.Register(LOGIN_DILL_LABEL, NewLoginDillTask, api.VERIFYAUTH)
}

type LoginDillRequest struct {
	api.BaseRequest
	Signature string `mapstructure:"Signature" validate:"required"`
	Address   string `mapstructure:"Address" validate:"required"`
	Nonce     int    `mapstructure:"Nonce" validate:"required"`
}

type LoginDillResponse struct {
	api.BaseResponse
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
	data := strings.Replace(codeTemplate, "ADDRESS", task.Request.Address, -1)
	data = strings.Replace(data, "NONCE", strconv.Itoa(task.Request.Nonce), -1)

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

	//2 generate cookie
	//创建并保存 Cookie 到 Session 中
	var cookieValue string

	retryCount := 10
	isSet := false
	for i := 0; i < retryCount; i++ {
		cookieValue = uuid.NewString()
		v := session.Get(cookieValue)
		if v == nil {
			//生成唯一的session_id，值为用户地址
			session.Set(cookieValue, task.Request.Address)
			//把 session 的修改写入redis
			err := session.Save()
			if err == nil {
				isSet = true
				break
			} else {
				log.Errorf("%s, save cookie-address to session failed, %s", task.Request.RequestUUID, err.Error())
			}
		}
	}

	if !isSet {
		task.Response.BaseResponse.RetCode = int(errors.SaveSessionFailed().Code())
		task.Response.BaseResponse.Message = errors.SaveSessionFailed().Message()
		return task.Response, err
	}

	cookie := &http.Cookie{
		Name:     "login_dill",
		Value:    cookieValue,
		Expires:  time.Now().UTC().Add(time.Duration(globalSessionMaxAge) * time.Second), // 设置超时时间为12小时 测试用15天
		HttpOnly: true,                                                                   // 仅允许通过 HTTP 访问
		Secure:   false,                                                                  // 仅在 HTTPS 连接中传输
	}

	//cookieValue 是通过 Set-Cookie HTTP 响应头直接返回给客户端的（不是task.Response）
	http.SetCookie(c.Writer, cookie)

	// 删除一次性的nonce，退出当前登录需要重新生成nonce，之前的session-id无法再使用，会慢慢过期
	session.Delete(key)
	err = session.Save()
	if err != nil {
		log.Errorf("%s, delete nonce from session failed, %s", task.Request.RequestUUID, err.Error())
	}

	return task.Response, nil
}

func verifySign(message string, signature string, addr string) (bool, error) {

	signatureBytes := common.Hex2Bytes(strings.TrimPrefix(signature, "0x"))
	addrBytes := common.Hex2Bytes(strings.TrimPrefix(addr, "0x"))

	return wallet.EthVerify(message, signatureBytes, addrBytes)
}
