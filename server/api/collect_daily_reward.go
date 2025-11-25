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
	PlayerID string `mapstructure:"PlayerID" validate:"required"`
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
	// 统一流程：基于 PlayerID 校验是否已领取 -> 发放并保存代币 -> 更新领取时间
	requestPlayerID := strings.TrimSpace(task.Request.PlayerID)
	profile, err := db.GetUserProfileByPlayerID(requestPlayerID)
	if err != nil {
		log.Errorf("%s, failed to get user profile by player_id=%s: %v", task.Request.RequestUUID, requestPlayerID, err)
		return nil, cmnErrors.GetUserProfileFailed(requestPlayerID)
	}

	// 校验当日是否已领取
	collected, err := db.HasCollectedDailyRewardByPlayerID(requestPlayerID)
	if err != nil {
		log.Errorf("%s, failed to check daily reward collection for player_id=%s: %v", task.Request.RequestUUID, requestPlayerID, err)
		return nil, cmnErrors.GetUserProfileFailed(requestPlayerID)
	}
	if collected {
		log.Errorf("%s, user %s has already collected daily reward today", task.Request.RequestUUID, requestPlayerID)
		return nil, cmnErrors.ActionError("Daily reward already collected")
	}

	// 发放 token
	dailyRewardTokens := int32(config.GameParams.DailyRewardTokens)
	var userToken *dao.UserToken
	userToken, err = db.GetPlayerToken(c.Request.Context(), profile.PlayerID)
	if err != nil && err != gorm.ErrRecordNotFound {
		log.Errorf("%s, failed to get user token for player_id=%s: %v", task.Request.RequestUUID, requestPlayerID, err)
		return nil, cmnErrors.OperateDbFailed()
	}
	if userToken == nil {
		userToken = &dao.UserToken{
			PlayerId:    profile.PlayerID,
			Points:      0,
			TokenAmount: dailyRewardTokens,
		}
	} else {
		userToken.TokenAmount += dailyRewardTokens
	}
	if err = db.SaveUserToken(*userToken); err != nil {
		log.Errorf("%s, failed to save user token for player_id=%s: %v", task.Request.RequestUUID, requestPlayerID, err)
		return nil, cmnErrors.OperateDbFailed()
	}

	// 更新领取时间
	if err = db.UpdateDailyRewardCollectionByPlayerID(requestPlayerID); err != nil {
		log.Errorf("%s, failed to update daily reward collection for player_id=%s: %v", task.Request.RequestUUID, requestPlayerID, err)
		return nil, cmnErrors.SaveUserProfileFailed()
	}
	log.Infof("%s, daily reward collected successfully for player_id=%s, tokens: %d", task.Request.RequestUUID, requestPlayerID, dailyRewardTokens)
	return task.Response, nil
}
