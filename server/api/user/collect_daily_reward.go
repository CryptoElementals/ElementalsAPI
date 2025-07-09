package user

import (
	"strings"

	"github.com/CryptoElementals/common/db"
	"github.com/CryptoElementals/common/errors"
	"github.com/CryptoElementals/common/log"
	"github.com/CryptoElementals/common/server/api"
	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
	"github.com/mitchellh/mapstructure"
)

const COLLECT_DAILY_REWARD_LABEL = "CollectDailyReward"

type CollectDailyRewardRequest struct {
	api.BaseRequest
}

type CollectDailyRewardResponse struct {
	api.BaseResponse
	Success bool `json:"Success"`
}

type CollectDailyRewardTask struct {
	Request  *CollectDailyRewardRequest
	Response *CollectDailyRewardResponse
}

// 将 map 类型的数据解码为 CollectDailyRewardRequest 结构体，并提取 RequestUUID
func NewCollectDailyRewardRequest(data *map[string]interface{}) (*CollectDailyRewardRequest, error) {
	req := &CollectDailyRewardRequest{}
	err := mapstructure.Decode(*data, &req)
	if err != nil {
		return nil, err
	}
	req.BaseRequest.RequestUUID = (*data)["RequestUUID"].(string)
	return req, nil
}

func NewCollectDailyRewardResponse(sessionId string) *CollectDailyRewardResponse {
	return &CollectDailyRewardResponse{
		BaseResponse: api.BaseResponse{
			Action:      COLLECT_DAILY_REWARD_LABEL + "Response",
			RequestUUID: sessionId,
		},
	}
}

func NewCollectDailyRewardTask(data *map[string]interface{}) (api.Task, error) {
	req, err := NewCollectDailyRewardRequest(data)
	if err != nil {
		return nil, err
	}
	task := &CollectDailyRewardTask{
		Request:  req,
		Response: NewCollectDailyRewardResponse(req.BaseRequest.RequestUUID),
	}

	validate := validator.New()
	err = validate.Struct(task.Request)
	if err != nil {
		return nil, err
	}

	return task, nil
}

func (task *CollectDailyRewardTask) Run(c *gin.Context) (api.Response, error) {
	// 从请求参数中获取用户地址（由中间件设置）
	_params, _ := c.Get("params")
	params, ok := _params.(*map[string]interface{})
	if !ok {
		log.Errorf("%s, params assert failed", task.Request.RequestUUID)
		return nil, errors.MissingLoginCookie()
	}

	address, ok := (*params)["Address"].(string)
	if !ok || address == "" {
		log.Errorf("%s, no address found in params", task.Request.RequestUUID)
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

	// 如果已经领取过今日奖励，返回错误
	if collected {
		log.Errorf("%s, user %s has already collected daily reward today", task.Request.RequestUUID, lowercaseAddress)
		return nil, errors.ActionError("今日奖励已领取")
	}

	// 更新用户每日奖励领取时间
	err = db.UpdateDailyRewardCollection(lowercaseAddress)
	if err != nil {
		log.Errorf("%s, failed to update daily reward collection for address %s: %v", task.Request.RequestUUID, lowercaseAddress, err)
		return nil, errors.SaveUserProfileFailed()
	}

	task.Response.Success = true
	log.Infof("%s, daily reward collected successfully for address %s", task.Request.RequestUUID, lowercaseAddress)
	return task.Response, nil
}
