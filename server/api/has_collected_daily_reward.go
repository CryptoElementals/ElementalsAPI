package api

import (
	"strings"

	"github.com/CryptoElementals/common/db"
	"github.com/CryptoElementals/common/errors"
	"github.com/CryptoElementals/common/log"
	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
	"github.com/mitchellh/mapstructure"
)

func init() {
	Register(HAS_COLLECTED_DAILY_REWARD_LABEL, NewHasCollectedDailyRewardTask, COOKIEAUTH)
}

type HasCollectedDailyRewardRequest struct {
	BaseRequest
	Address string `mapstructure:"Address"`
	Email   string `mapstructure:"Email"`
}

type HasCollectedDailyRewardResponse struct {
	BaseResponse
	Collected bool `json:"Collected"`
}

type HasCollectedDailyRewardTask struct {
	Request  *HasCollectedDailyRewardRequest
	Response *HasCollectedDailyRewardResponse
}

// 将 map 类型的数据解码为 HasCollectedDailyRewardRequest 结构体，并提取 RequestUUID
func NewHasCollectedDailyRewardRequest(data *map[string]interface{}) (*HasCollectedDailyRewardRequest, error) {
	req := &HasCollectedDailyRewardRequest{}
	err := mapstructure.Decode(*data, &req)
	if err != nil {
		return nil, err
	}
	req.BaseRequest.RequestUUID = (*data)["RequestUUID"].(string)
	return req, nil
}

func NewHasCollectedDailyRewardResponse(sessionId string) *HasCollectedDailyRewardResponse {
	return &HasCollectedDailyRewardResponse{
		BaseResponse: BaseResponse{
			Action:      HAS_COLLECTED_DAILY_REWARD_LABEL + "Response",
			RequestUUID: sessionId,
		},
	}
}

func NewHasCollectedDailyRewardTask(data *map[string]interface{}) (Task, error) {
	req, err := NewHasCollectedDailyRewardRequest(data)
	if err != nil {
		return nil, err
	}
	task := &HasCollectedDailyRewardTask{
		Request:  req,
		Response: NewHasCollectedDailyRewardResponse(req.BaseRequest.RequestUUID),
	}

	validate := validator.New()
	err = validate.Struct(task.Request)
	if err != nil {
		return nil, err
	}

	return task, nil
}

func (task *HasCollectedDailyRewardTask) Run(c *gin.Context) (Response, error) {
	// 允许通过 Address 或 Email 查询，至少提供一个
	requestAddress := strings.ToLower(strings.TrimSpace(task.Request.Address))
	requestEmail := strings.TrimSpace(task.Request.Email)
	if requestAddress == "" && requestEmail == "" {
		log.Errorf("%s, neither address nor email provided", task.Request.RequestUUID)
		return nil, errors.MissingParams("Address or Email")
	}

	if requestAddress != "" {
		collected, err := db.HasCollectedDailyReward(requestAddress)
		if err != nil {
			log.Errorf("%s, failed to check daily reward collection for address %s: %v", task.Request.RequestUUID, requestAddress, err)
			return nil, errors.GetUserProfileFailed(requestAddress)
		}
		task.Response.Collected = collected
		log.Infof("%s, daily reward collection status checked (addr=%s): %v", task.Request.RequestUUID, requestAddress, collected)
		return task.Response, nil
	}
	collected, err := db.HasCollectedDailyRewardByEmail(requestEmail)
	if err != nil {
		log.Errorf("%s, failed to check daily reward collection for email %s: %v", task.Request.RequestUUID, requestEmail, err)
		return nil, errors.GetUserProfileFailed(requestEmail)
	}
	task.Response.Collected = collected
	log.Infof("%s, daily reward collection status checked (email=%s): %v", task.Request.RequestUUID, requestEmail, collected)
	return task.Response, nil
}
