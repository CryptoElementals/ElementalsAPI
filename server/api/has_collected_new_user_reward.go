package api

import (
	"strings"

	"github.com/CryptoElementals/common/config"
	"github.com/CryptoElementals/common/db"
	"github.com/CryptoElementals/common/errors"
	"github.com/CryptoElementals/common/log"
	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
	"github.com/mitchellh/mapstructure"
)

func init() {
	Register(HAS_COLLECTED_NEW_USER_REWARD_LABEL, NewHasCollectedNewUserRewardTask, COOKIEAUTH)
}

type HasCollectedNewUserRewardRequest struct {
	BaseRequest
	PlayerID string `mapstructure:"PlayerID" validate:"required"`
}

type HasCollectedNewUserRewardResponse struct {
	BaseResponse
	Collected bool `json:"Collected"`
}

type HasCollectedNewUserRewardTask struct {
	Request  *HasCollectedNewUserRewardRequest
	Response *HasCollectedNewUserRewardResponse
}

func NewHasCollectedNewUserRewardRequest(data *map[string]interface{}) (*HasCollectedNewUserRewardRequest, error) {
	req := &HasCollectedNewUserRewardRequest{}
	if err := mapstructure.Decode(*data, &req); err != nil {
		return nil, err
	}
	req.BaseRequest.RequestUUID = (*data)["RequestUUID"].(string)
	return req, nil
}

func NewHasCollectedNewUserRewardResponse(sessionId string) *HasCollectedNewUserRewardResponse {
	return &HasCollectedNewUserRewardResponse{
		BaseResponse: BaseResponse{
			Action:      HAS_COLLECTED_NEW_USER_REWARD_LABEL + "Response",
			RequestUUID: sessionId,
		},
	}
}

func NewHasCollectedNewUserRewardTask(data *map[string]interface{}) (Task, error) {
	req, err := NewHasCollectedNewUserRewardRequest(data)
	if err != nil {
		return nil, err
	}
	task := &HasCollectedNewUserRewardTask{
		Request:  req,
		Response: NewHasCollectedNewUserRewardResponse(req.BaseRequest.RequestUUID),
	}

	validate := validator.New()
	if err := validate.Struct(task.Request); err != nil {
		return nil, err
	}

	return task, nil
}

func (task *HasCollectedNewUserRewardTask) Run(c *gin.Context) (Response, error) {
	playerID := strings.TrimSpace(task.Request.PlayerID)
	if !config.GameParams.EnableNewUserReward {
		// Feature off: report as already handled so clients can hide the claim UI without calling collect.
		task.Response.Collected = true
		log.Infof("%s, new user reward disabled by config; returning Collected=true (player_id=%s)", task.Request.RequestUUID, playerID)
		return task.Response, nil
	}
	collected, err := db.HasCollectedNewUserRewardByPlayerID(playerID)
	if err != nil {
		log.Errorf("%s, failed to check new user reward collection for player_id=%s: %v", task.Request.RequestUUID, playerID, err)
		return nil, errors.GetUserProfileFailed(playerID)
	}
	task.Response.Collected = collected
	log.Infof("%s, new user reward collection status checked (player_id=%s): %v", task.Request.RequestUUID, playerID, collected)
	return task.Response, nil
}
