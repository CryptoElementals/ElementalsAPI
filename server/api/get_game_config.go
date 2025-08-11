package api

import (
	"github.com/CryptoElementals/common/config"
	"github.com/CryptoElementals/common/db"
	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
	"github.com/mitchellh/mapstructure"
	"gorm.io/gorm"
)

func init() {
	Register(GET_GAME_CONFIG_LABEL, NewGetGameConfigTask, COOKIEAUTH)
}

type GetGameConfigRequest struct {
	BaseRequest
	Address string `mapstructure:"Address" validate:"required"`
}

type GetGameConfigResponse struct {
	BaseResponse
	KeygenPolicy   uint   `json:"KeygenPolicy"`
	TempPrivateKey string `json:"TempPrivateKey,omitempty"`
	TempAddress    string `json:"TempAddress,omitempty"`
}

type GetGameConfigTask struct {
	Request  *GetGameConfigRequest
	Response *GetGameConfigResponse
}

func NewGetGameConfigRequest(data *map[string]interface{}) (*GetGameConfigRequest, error) {
	req := &GetGameConfigRequest{}
	err := mapstructure.Decode(*data, &req)
	if err != nil {
		return nil, err
	}
	req.BaseRequest.RequestUUID = (*data)["RequestUUID"].(string)
	return req, nil
}

func NewGetGameConfigResponse(sessionId string) *GetGameConfigResponse {
	return &GetGameConfigResponse{
		BaseResponse: BaseResponse{
			Action:      GET_GAME_CONFIG_LABEL + "Response",
			RequestUUID: sessionId,
		},
	}
}

func NewGetGameConfigTask(data *map[string]interface{}) (Task, error) {
	req, err := NewGetGameConfigRequest(data)
	if err != nil {
		return nil, err
	}
	task := &GetGameConfigTask{
		Request:  req,
		Response: NewGetGameConfigResponse(req.BaseRequest.RequestUUID),
	}

	validate := validator.New()
	err = validate.Struct(task.Request)
	if err != nil {
		return nil, err
	}
	return task, nil
}

func (t *GetGameConfigTask) Run(c *gin.Context) (Response, error) {
	policy := config.GameParams.KeygenPolicy
	t.Response.KeygenPolicy = policy
	if policy != 1 {
		return t.Response, nil
	}

	// 策略1：后端从数据库为该地址分配临时私钥与地址
	addr := t.Request.Address
	// 已绑定的直接返回
	if rec, err := db.GetDevTempKeyByAddress(addr); err == nil && rec != nil {
		t.Response.TempPrivateKey = rec.TempPrivateKey
		t.Response.TempAddress = rec.TempAddress
		return t.Response, nil
	}
	// 未绑定则分配一个空闲记录
	rec, err := db.AssignNextAvailableDevTempKey(addr)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			// 无可用临时密钥
			t.Response.RetCode = 8454
			t.Response.Message = "No available temporary key"
			return t.Response, nil
		}
		return nil, err
	}
	// 返回分配结果
	t.Response.TempPrivateKey = rec.TempPrivateKey
	t.Response.TempAddress = rec.TempAddress
	return t.Response, nil
}
