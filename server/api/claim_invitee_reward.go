package api

import (
	"strings"

	"github.com/CryptoElementals/common/db"
	cmnErrors "github.com/CryptoElementals/common/errors"
	"github.com/CryptoElementals/common/log"
	"github.com/CryptoElementals/common/server/invite"
	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
	"github.com/mitchellh/mapstructure"
	"gorm.io/gorm"
)

func init() {
	Register(CLAIM_INVITEE_REWARD_LABEL, NewClaimInviteeRewardTask, COOKIEAUTH)
}

type ClaimInviteeRewardRequest struct {
	BaseRequest
	PlayerID string `mapstructure:"PlayerID" validate:"required"`
}

type ClaimInviteeRewardResponse struct {
	BaseResponse
	Point int32 `json:"Point"`
}

type ClaimInviteeRewardTask struct {
	Request  *ClaimInviteeRewardRequest
	Response *ClaimInviteeRewardResponse
}

func NewClaimInviteeRewardRequest(data *map[string]interface{}) (*ClaimInviteeRewardRequest, error) {
	req := &ClaimInviteeRewardRequest{}
	if err := mapstructure.Decode(*data, &req); err != nil {
		return nil, err
	}
	req.BaseRequest.RequestUUID = (*data)["RequestUUID"].(string)
	return req, nil
}

func NewClaimInviteeRewardResponse(sessionId string) *ClaimInviteeRewardResponse {
	return &ClaimInviteeRewardResponse{
		BaseResponse: BaseResponse{
			Action:      CLAIM_INVITEE_REWARD_LABEL + "Response",
			RequestUUID: sessionId,
		},
	}
}

func NewClaimInviteeRewardTask(data *map[string]interface{}) (Task, error) {
	req, err := NewClaimInviteeRewardRequest(data)
	if err != nil {
		return nil, err
	}
	task := &ClaimInviteeRewardTask{
		Request:  req,
		Response: NewClaimInviteeRewardResponse(req.BaseRequest.RequestUUID),
	}
	validate := validator.New()
	if err := validate.Struct(task.Request); err != nil {
		return nil, err
	}
	return task, nil
}

func (task *ClaimInviteeRewardTask) Run(c *gin.Context) (Response, error) {
	playerID := strings.TrimSpace(task.Request.PlayerID)
	profile, err := db.GetUserProfileByPlayerID(playerID)
	if err != nil {
		return nil, cmnErrors.GetUserProfileFailed(playerID)
	}

	reward, err := db.GetUnclaimedInviteeReward(profile.PlayerID)
	if err != nil {
		return nil, cmnErrors.OperateDbFailed()
	}
	if reward == nil {
		return nil, cmnErrors.ActionError("Invite reward not available or already claimed")
	}

	if err := invite.CreditUserPointsAllEnvironments(profile.PlayerID, reward.Point, "invitee_initial_reward"); err != nil {
		log.Errorf("%s, credit invitee reward points failed: %v", task.Request.RequestUUID, err)
		return nil, cmnErrors.OperateDbFailed()
	}

	var marked bool
	err = db.Get().Transaction(func(tx *gorm.DB) error {
		var merr error
		marked, merr = db.MarkInviteeRewardClaimedTx(tx, profile.PlayerID, reward.Point)
		return merr
	})
	if err != nil {
		log.Errorf("%s, mark invitee reward failed: %v", task.Request.RequestUUID, err)
		return nil, cmnErrors.OperateDbFailed()
	}
	if !marked {
		return nil, cmnErrors.ActionError("Invite reward already claimed")
	}

	task.Response.Point = reward.Point
	return task.Response, nil
}
