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
	Register(COLLECT_NEW_USER_REWARD_LABEL, NewCollectNewUserRewardTask, COOKIEAUTH)
}

type CollectNewUserRewardRequest struct {
	BaseRequest
	PlayerID string `mapstructure:"PlayerID" validate:"required"`
}

type CollectNewUserRewardResponse struct {
	BaseResponse
	RewardAmount int32 `json:"RewardAmount"`
}

type CollectNewUserRewardTask struct {
	Request  *CollectNewUserRewardRequest
	Response *CollectNewUserRewardResponse
}

func NewCollectNewUserRewardRequest(data *map[string]interface{}) (*CollectNewUserRewardRequest, error) {
	req := &CollectNewUserRewardRequest{}
	if err := mapstructure.Decode(*data, &req); err != nil {
		return nil, err
	}
	req.BaseRequest.RequestUUID = (*data)["RequestUUID"].(string)
	return req, nil
}

func NewCollectNewUserRewardResponse(sessionId string) *CollectNewUserRewardResponse {
	return &CollectNewUserRewardResponse{
		BaseResponse: BaseResponse{
			Action:      COLLECT_NEW_USER_REWARD_LABEL + "Response",
			RequestUUID: sessionId,
		},
	}
}

func NewCollectNewUserRewardTask(data *map[string]interface{}) (Task, error) {
	req, err := NewCollectNewUserRewardRequest(data)
	if err != nil {
		return nil, err
	}
	task := &CollectNewUserRewardTask{
		Request:  req,
		Response: NewCollectNewUserRewardResponse(req.BaseRequest.RequestUUID),
	}

	validate := validator.New()
	if err := validate.Struct(task.Request); err != nil {
		return nil, err
	}

	return task, nil
}

func (task *CollectNewUserRewardTask) Run(c *gin.Context) (Response, error) {
	playerID := strings.TrimSpace(task.Request.PlayerID)
	if !config.GameParams.EnableNewUserReward {
		log.Errorf("%s, new user reward disabled by config (player_id=%s)", task.Request.RequestUUID, playerID)
		return nil, cmnErrors.ActionError("New user reward is not enabled")
	}
	rewardAmount := int32(config.GameParams.NewUserRewardTokens)
	if rewardAmount <= 0 {
		log.Errorf("%s, invalid new user reward tokens config: %d", task.Request.RequestUUID, rewardAmount)
		return nil, cmnErrors.ActionError("New user reward not configured")
	}

	err := db.Get().WithContext(c.Request.Context()).Transaction(func(tx *gorm.DB) error {
		profile, err := db.GetUserProfileByPlayerIDWithDB(playerID, tx)
		if err != nil {
			return err
		}

		marked, err := db.MarkNewUserRewardCollectedByPlayerIDTx(tx, playerID)
		if err != nil {
			return err
		}
		if !marked {
			return cmnErrors.ActionError("New user reward already collected")
		}

		res := tx.Model(&dao.UserToken{}).
			Where("player_id = ?", profile.PlayerID).
			Update("token_amount", gorm.Expr("token_amount + ?", rewardAmount))
		if res.Error != nil {
			return res.Error
		}
		if res.RowsAffected == 0 {
			userToken := dao.UserToken{
				PlayerId:    profile.PlayerID,
				Points:      0,
				TokenAmount: rewardAmount,
			}
			if err := tx.Create(&userToken).Error; err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		if customErr, ok := err.(cmnErrors.Error); ok {
			return nil, customErr
		}
		if err == gorm.ErrRecordNotFound {
			log.Errorf("%s, failed to get user profile by player_id=%s: %v", task.Request.RequestUUID, playerID, err)
			return nil, cmnErrors.GetUserProfileFailed(playerID)
		}
		log.Errorf("%s, failed to collect new user reward for player_id=%s: %v", task.Request.RequestUUID, playerID, err)
		return nil, cmnErrors.OperateDbFailed()
	}

	task.Response.RewardAmount = rewardAmount
	log.Infof("%s, new user reward collected successfully for player_id=%s, tokens=%d", task.Request.RequestUUID, playerID, rewardAmount)
	return task.Response, nil
}
