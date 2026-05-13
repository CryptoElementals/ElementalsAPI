package api

import (
	"context"
	"strings"
	"time"

	"github.com/CryptoElementals/common/config"
	"github.com/CryptoElementals/common/db"
	cmnErrors "github.com/CryptoElementals/common/errors"
	"github.com/CryptoElementals/common/log"
	"github.com/CryptoElementals/common/rpc/client"
	"github.com/CryptoElementals/common/rpc/proto"
	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
	"github.com/mitchellh/mapstructure"
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
	// 统一流程：检查活动期间 -> 校验是否已领取 -> 判断第一天还是后续天 -> 发放并保存代币 -> 更新领取时间
	requestPlayerID := strings.TrimSpace(task.Request.PlayerID)
	if !config.GameParams.EnableDailyReward {
		log.Errorf("%s, daily reward disabled by config (player_id=%s)", task.Request.RequestUUID, requestPlayerID)
		return nil, cmnErrors.ActionError("Daily reward is not enabled")
	}
	profile, err := db.GetUserProfileByPlayerID(requestPlayerID)
	if err != nil {
		log.Errorf("%s, failed to get user profile by player_id=%s: %v", task.Request.RequestUUID, requestPlayerID, err)
		return nil, cmnErrors.GetUserProfileFailed(requestPlayerID)
	}

	// 检查活动是否在有效期内（使用UTC时间统一判断）
	now := time.Now().UTC()
	startDate, err := time.Parse("2006-01-02", config.GameParams.DailyRewardStartDate)
	if err != nil {
		log.Errorf("%s, invalid daily reward start date: %s, error: %v", task.Request.RequestUUID, config.GameParams.DailyRewardStartDate, err)
		return nil, cmnErrors.ActionError("Daily reward activity not configured")
	}
	endDate, err := time.Parse("2006-01-02", config.GameParams.DailyRewardEndDate)
	if err != nil {
		log.Errorf("%s, invalid daily reward end date: %s, error: %v", task.Request.RequestUUID, config.GameParams.DailyRewardEndDate, err)
		return nil, cmnErrors.ActionError("Daily reward activity not configured")
	}

	// 只比较日期部分，忽略时间，统一使用UTC时区
	nowDate := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)
	startDateOnly := time.Date(startDate.Year(), startDate.Month(), startDate.Day(), 0, 0, 0, 0, time.UTC)
	endDateOnly := time.Date(endDate.Year(), endDate.Month(), endDate.Day(), 0, 0, 0, 0, time.UTC)

	if nowDate.Before(startDateOnly) || nowDate.After(endDateOnly) {
		log.Errorf("%s, daily reward activity is not active. Current date: %s, Activity period: %s to %s",
			task.Request.RequestUUID, nowDate.Format("2006-01-02"), startDateOnly.Format("2006-01-02"), endDateOnly.Format("2006-01-02"))
		return nil, cmnErrors.ActionError("Daily reward activity is not active")
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

	// 判断是否是活动期间内第一次领取
	// 条件：用户从未在活动期间内领取过（从未领取过，或上次领取时间在活动开始日期之前）
	// 统一使用UTC时区进行比较
	isFirstTimeInActivity := false
	if profile.CollectedRewardAt == nil {
		// 从未领取过，这是活动期间内第一次
		isFirstTimeInActivity = true
	} else {
		// 检查上次领取时间是否在活动开始日期之前（转换为UTC进行比较）
		lastCollectedUTC := profile.CollectedRewardAt.UTC()
		lastCollectedDate := time.Date(lastCollectedUTC.Year(), lastCollectedUTC.Month(), lastCollectedUTC.Day(), 0, 0, 0, 0, time.UTC)
		if lastCollectedDate.Before(startDateOnly) {
			// 上次领取时间在活动开始之前，这是活动期间内第一次
			isFirstTimeInActivity = true
		}
	}

	// 根据是否是活动期间内第一次领取决定发放的token数量
	var dailyRewardTokens int32
	if isFirstTimeInActivity {
		dailyRewardTokens = int32(config.GameParams.FirstTimeRewardTokens)
		log.Infof("%s, player_id=%s collecting first time reward in activity: %d tokens", task.Request.RequestUUID, requestPlayerID, dailyRewardTokens)
	} else {
		dailyRewardTokens = int32(config.GameParams.DailyRewardTokensAfterFirst)
		log.Infof("%s, player_id=%s collecting daily reward: %d tokens", task.Request.RequestUUID, requestPlayerID, dailyRewardTokens)
	}

	lobbyClient := client.GetGlobalLobbyClient()
	if lobbyClient == nil {
		log.Errorf("%s, gRPC lobby client not initialized", task.Request.RequestUUID)
		return nil, cmnErrors.ActionError("gRPC lobby client not initialized")
	}
	if _, err = lobbyClient.CreditUserTokens(context.Background(), &proto.CreditUserTokensRequest{
		PlayerID: profile.PlayerID,
		Delta:    dailyRewardTokens,
		Reason:   "daily_reward",
	}); err != nil {
		log.Errorf("%s, failed to credit user token for player_id=%s: %v", task.Request.RequestUUID, requestPlayerID, err)
		return nil, cmnErrors.OperateDbFailed()
	}

	// 更新领取时间
	if err = db.UpdateDailyRewardCollectionByPlayerID(requestPlayerID); err != nil {
		log.Errorf("%s, failed to update daily reward collection for player_id=%s: %v", task.Request.RequestUUID, requestPlayerID, err)
		return nil, cmnErrors.SaveUserProfileFailed()
	}
	log.Infof("%s, daily reward collected successfully for player_id=%s, tokens: %d (isFirstTimeInActivity: %v)", task.Request.RequestUUID, requestPlayerID, dailyRewardTokens, isFirstTimeInActivity)
	return task.Response, nil
}
