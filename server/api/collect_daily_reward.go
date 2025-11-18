package api

import (
	"strings"

	"github.com/CryptoElementals/common/config"
	"github.com/CryptoElementals/common/db"
	cmnErrors "github.com/CryptoElementals/common/errors"
	"github.com/CryptoElementals/common/log"
	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
	"github.com/mitchellh/mapstructure"
	"gorm.io/gorm"

	dao "github.com/CryptoElementals/common/models"
)

func init() {
	Register(COLLECT_DAILY_REWARD_LABEL, NewCollectDailyRewardTask, COOKIEAUTH)
}

type CollectDailyRewardRequest struct {
	BaseRequest
	Address string `mapstructure:"Address"`
	Email   string `mapstructure:"Email"`
}

type CollectDailyRewardResponse struct {
	BaseResponse
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
		BaseResponse: BaseResponse{
			Action:      COLLECT_DAILY_REWARD_LABEL + "Response",
			RequestUUID: sessionId,
		},
	}
}

func NewCollectDailyRewardTask(data *map[string]interface{}) (Task, error) {
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

func (task *CollectDailyRewardTask) Run(c *gin.Context) (Response, error) {
	// 允许通过 Address 或 Email 领取，至少提供一个
	requestAddress := strings.ToLower(strings.TrimSpace(task.Request.Address))
	requestEmail := strings.TrimSpace(task.Request.Email)
	if requestAddress == "" && requestEmail == "" {
		log.Errorf("%s, neither address nor email provided", task.Request.RequestUUID)
		return nil, cmnErrors.MissingParams("Address or Email")
	}

	// Address 流程：保持原有逻辑（检查->发放Token->更新时间）
	if requestAddress != "" {
		collected, err := db.HasCollectedDailyReward(requestAddress)
		if err != nil {
			log.Errorf("%s, failed to check daily reward collection for address %s: %v", task.Request.RequestUUID, requestAddress, err)
			return nil, cmnErrors.GetUserProfileFailed(requestAddress)
		}
		if collected {
			log.Errorf("%s, user %s has already collected daily reward today", task.Request.RequestUUID, requestAddress)
			return nil, cmnErrors.ActionError("Daily reward already collected")
		}

		// 确保用户存在
		_, err = db.GetUserProfileByAddress(requestAddress)
		if err != nil {
			log.Errorf("%s, failed to get user profile for address %s: %v", task.Request.RequestUUID, requestAddress, err)
			return nil, cmnErrors.GetUserProfileFailed(requestAddress)
		}

		dailyRewardTokens := int32(config.GameParams.DailyRewardTokens)

		userToken, err := db.GetPlayerToken(c.Request.Context(), requestAddress)
		if err != nil && err != gorm.ErrRecordNotFound {
			log.Errorf("%s, failed to get user token for address %s: %v", task.Request.RequestUUID, requestAddress, err)
			return nil, cmnErrors.OperateDbFailed()
		}
		if userToken == nil {
			userToken = &dao.UserToken{
				WalletAddress: requestAddress,
				Points:        0,
				TokenAmount:   dailyRewardTokens,
			}
		} else {
			userToken.TokenAmount += dailyRewardTokens
		}
		if err = db.SaveUserToken(*userToken); err != nil {
			log.Errorf("%s, failed to save user token for address %s: %v", task.Request.RequestUUID, requestAddress, err)
			return nil, cmnErrors.OperateDbFailed()
		}

		if err = db.UpdateDailyRewardCollection(requestAddress); err != nil {
			log.Errorf("%s, failed to update daily reward collection for address %s: %v", task.Request.RequestUUID, requestAddress, err)
			return nil, cmnErrors.SaveUserProfileFailed()
		}
		log.Infof("%s, daily reward collected successfully for address %s, tokens: %d", task.Request.RequestUUID, requestAddress, dailyRewardTokens)
		return task.Response, nil
	}

	// Email 流程：直接按 Email 查询/更新（不通过 Email 解析 Address）
	collected, err := db.HasCollectedDailyRewardByEmail(requestEmail)
	if err != nil {
		log.Errorf("%s, failed to check daily reward collection for email %s: %v", task.Request.RequestUUID, requestEmail, err)
		return nil, cmnErrors.GetUserProfileFailed(requestEmail)
	}
	if collected {
		log.Errorf("%s, user %s has already collected daily reward today (email)", task.Request.RequestUUID, requestEmail)
		return nil, cmnErrors.ActionError("Daily reward already collected")
	}

	// 确保用户存在（按邮箱）
	if _, err = db.GetUserProfileByEmail(requestEmail); err != nil {
		log.Errorf("%s, failed to get user profile for email %s: %v", task.Request.RequestUUID, requestEmail, err)
		return nil, cmnErrors.GetUserProfileFailed(requestEmail)
	}

	// 目前Email 模式下仅更新领取时间，还没有发放token）
	if err = db.UpdateDailyRewardCollectionByEmail(requestEmail); err != nil {
		log.Errorf("%s, failed to update daily reward collection for email %s: %v", task.Request.RequestUUID, requestEmail, err)
		return nil, cmnErrors.SaveUserProfileFailed()
	}
	log.Infof("%s, daily reward collected successfully for email %s (no token distribution)", task.Request.RequestUUID, requestEmail)
	return task.Response, nil
}
