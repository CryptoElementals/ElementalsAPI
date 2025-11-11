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

	// 统一流程：解析用户档案 -> 校验是否已领取 -> 发放并保存代币 -> 更新领取时间
	useAddress := requestAddress != ""

	// 读取档案
	var profile *dao.UserProfile
	var err error
	if useAddress {
		profile, err = db.GetUserProfileByAddress(requestAddress)
	} else {
		profile, err = db.GetUserProfileByEmail(requestEmail)
	}
	if err != nil {
		key := requestAddress
		if !useAddress {
			key = requestEmail
		}
		log.Errorf("%s, failed to get user profile for %s: %v", task.Request.RequestUUID, key, err)
		return nil, cmnErrors.GetUserProfileFailed(key)
	}

	// 校验当日是否已领取
	collected := false
	if useAddress {
		collected, err = db.HasCollectedDailyReward(requestAddress)
	} else {
		collected, err = db.HasCollectedDailyRewardByEmail(requestEmail)
	}
	if err != nil {
		key := requestAddress
		if !useAddress {
			key = requestEmail
		}
		log.Errorf("%s, failed to check daily reward collection for %s: %v", task.Request.RequestUUID, key, err)
		return nil, cmnErrors.GetUserProfileFailed(key)
	}
	if collected {
		key := requestAddress
		if !useAddress {
			key = requestEmail
		}
		log.Errorf("%s, user %s has already collected daily reward today", task.Request.RequestUUID, key)
		return nil, cmnErrors.ActionError("Daily reward already collected")
	}

	// 发放 token
	dailyRewardTokens := int32(config.GameParams.DailyRewardTokens)
	var userToken *dao.UserToken
	if useAddress {
		userToken, err = db.GetPlayerToken(c.Request.Context(), requestAddress)
	} else {
		userToken, err = db.GetPlayerTokenByEmail(c.Request.Context(), requestEmail)
	}
	if err != nil && err != gorm.ErrRecordNotFound {
		key := requestAddress
		if !useAddress {
			key = requestEmail
		}
		log.Errorf("%s, failed to get user token for %s: %v", task.Request.RequestUUID, key, err)
		return nil, cmnErrors.OperateDbFailed()
	}
	if userToken == nil {
		userToken = &dao.UserToken{
			UserID:      profile.UserID,
			Points:      0,
			TokenAmount: dailyRewardTokens,
		}
	} else {
		userToken.TokenAmount += dailyRewardTokens
	}
	if err = db.SaveUserToken(*userToken); err != nil {
		key := requestAddress
		if !useAddress {
			key = requestEmail
		}
		log.Errorf("%s, failed to save user token for %s: %v", task.Request.RequestUUID, key, err)
		return nil, cmnErrors.OperateDbFailed()
	}

	// 更新领取时间
	if useAddress {
		if err = db.UpdateDailyRewardCollection(requestAddress); err != nil {
			log.Errorf("%s, failed to update daily reward collection for address %s: %v", task.Request.RequestUUID, requestAddress, err)
			return nil, cmnErrors.SaveUserProfileFailed()
		}
		log.Infof("%s, daily reward collected successfully for address %s, tokens: %d", task.Request.RequestUUID, requestAddress, dailyRewardTokens)
	} else {
		if err = db.UpdateDailyRewardCollectionByEmail(requestEmail); err != nil {
			log.Errorf("%s, failed to update daily reward collection for email %s: %v", task.Request.RequestUUID, requestEmail, err)
			return nil, cmnErrors.SaveUserProfileFailed()
		}
		log.Infof("%s, daily reward collected successfully for email %s, tokens: %d", task.Request.RequestUUID, requestEmail, dailyRewardTokens)
	}
	return task.Response, nil
}
