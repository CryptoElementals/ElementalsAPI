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
	// 从请求中获取用户地址（由中间件设置）
	address := task.Request.Address
	if address == "" {
		log.Errorf("%s, no address found in request", task.Request.RequestUUID)
		return nil, errors.MissingLoginCookie()
	}

	// 将地址转换为小写，确保与数据库中存储的格式一致
	lowercaseAddress := strings.ToLower(address)

	// 检查用户是否已领取今日奖励
	collected, err := db.HasCollectedDailyReward(lowercaseAddress)
	if err != nil {
		log.Errorf("%s, failed to check daily reward collection for address %s: %v", task.Request.RequestUUID, lowercaseAddress, err)
		return nil, errors.GetUserProfileFailed(lowercaseAddress)
	}

	task.Response.Collected = collected
	log.Infof("%s, daily reward collection status checked for address %s: %v", task.Request.RequestUUID, lowercaseAddress, collected)
	return task.Response, nil
}
