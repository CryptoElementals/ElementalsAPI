package api

import (
	"strings"

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
	PlayerID string `mapstructure:"PlayerID" validate:"required"`
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

func (task *GetGameConfigTask) Run(c *gin.Context) (Response, error) {
	// 通过 PlayerID 解析玩家地址
	profile, err := db.GetUserProfileByPlayerID(strings.TrimSpace(task.Request.PlayerID))
	if err != nil || profile == nil || profile.Address == "" {
		task.Response.BaseResponse.RetCode = 1001
		task.Response.BaseResponse.Message = "Failed to get player address by player id"
		return task.Response, nil
	}
	address := strings.ToLower(profile.Address)

	// 获取并填充基础游戏配置
	policy := config.GameParams.KeygenPolicy
	task.Response.KeygenPolicy = policy

	if policy != 1 {
		return task.Response, nil
	}

	// 策略1：后端从数据库为该地址分配临时私钥与地址
	// 已绑定的直接返回
	if rec, err := db.GetDevTempKeyByAddress(address); err == nil && rec != nil {
		task.Response.TempPrivateKey = rec.TempPrivateKey
		task.Response.TempAddress = rec.TempAddress
		return task.Response, nil
	}
	// 未绑定则分配一个空闲记录
	rec, err := db.AssignNextAvailableDevTempKey(address)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			// 无可用临时密钥
			task.Response.RetCode = 8454
			task.Response.Message = "No available temporary key"
			return task.Response, nil
		}
		return nil, err
	}
	// 返回分配结果
	task.Response.TempPrivateKey = rec.TempPrivateKey
	task.Response.TempAddress = rec.TempAddress
	return task.Response, nil
}
