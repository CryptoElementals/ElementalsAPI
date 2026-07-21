package api

import (
	"strings"
	"time"

	"github.com/CryptoElementals/common/db"
	cmnErrors "github.com/CryptoElementals/common/errors"
	"github.com/CryptoElementals/common/internal/playerlevel"
	"github.com/CryptoElementals/common/log"
	"github.com/CryptoElementals/common/server/invite"
	dao "github.com/CryptoElementals/common/models"
	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
	"github.com/mitchellh/mapstructure"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

func init() {
	Register(CLAIM_INVITER_REWARD_LABEL, NewClaimInviterRewardTask, COOKIEAUTH)
}

type ClaimInviterRewardRequest struct {
	BaseRequest
	PlayerID string `mapstructure:"PlayerID" validate:"required"`
	RewardID uint   `mapstructure:"RewardID" validate:"required"`
}

type ClaimInviterRewardResponse struct {
	BaseResponse
	Point int32 `json:"Point"`
}

type ClaimInviterRewardTask struct {
	Request  *ClaimInviterRewardRequest
	Response *ClaimInviterRewardResponse
}

func NewClaimInviterRewardRequest(data *map[string]interface{}) (*ClaimInviterRewardRequest, error) {
	req := &ClaimInviterRewardRequest{}
	if err := mapstructure.Decode(*data, &req); err != nil {
		return nil, err
	}
	req.BaseRequest.RequestUUID = (*data)["RequestUUID"].(string)
	return req, nil
}

func NewClaimInviterRewardResponse(sessionId string) *ClaimInviterRewardResponse {
	return &ClaimInviterRewardResponse{
		BaseResponse: BaseResponse{
			Action:      CLAIM_INVITER_REWARD_LABEL + "Response",
			RequestUUID: sessionId,
		},
	}
}

func NewClaimInviterRewardTask(data *map[string]interface{}) (Task, error) {
	req, err := NewClaimInviterRewardRequest(data)
	if err != nil {
		return nil, err
	}
	task := &ClaimInviterRewardTask{
		Request:  req,
		Response: NewClaimInviterRewardResponse(req.BaseRequest.RequestUUID),
	}
	validate := validator.New()
	if err := validate.Struct(task.Request); err != nil {
		return nil, err
	}
	return task, nil
}

func (task *ClaimInviterRewardTask) Run(c *gin.Context) (Response, error) {
	playerID := strings.TrimSpace(task.Request.PlayerID)
	profile, err := db.GetUserProfileByPlayerID(playerID)
	if err != nil {
		return nil, cmnErrors.GetUserProfileFailed(playerID)
	}
	if db.EffectiveServerType(profile) != dao.ServerTypeNormal {
		return nil, cmnErrors.ActionError("Invite rewards require a normal account")
	}

	row, err := db.GetInviterRewardByID(task.Request.RewardID)
	if err != nil || row == nil || row.InviterPlayerID != profile.PlayerID || row.ClaimedAt != nil {
		return nil, cmnErrors.ActionError("Reward not found or already claimed")
	}
	inviteeProfile, perr := db.GetUserProfileByPlayerIDInt(row.InviteePlayerID)
	if perr != nil {
		return nil, cmnErrors.ActionError("Invitee not found")
	}
	points, perr := invite.PlayerPointsFromLobby(db.EffectiveServerType(inviteeProfile), row.InviteePlayerID)
	if perr != nil {
		return nil, cmnErrors.OperateDbFailed()
	}
	if playerlevel.CalculateLevel(points) < row.MilestoneLevel {
		return nil, cmnErrors.ActionError("Invitee level requirement not met")
	}

		if err := invite.CreditUserPointsForServerType(dao.ServerTypeNormal, profile.PlayerID, row.Point, "inviter_milestone_reward"); err != nil {
		log.Errorf("%s, credit inviter reward failed: %v", task.Request.RequestUUID, err)
		return nil, cmnErrors.OperateDbFailed()
	}

	err = db.Get().Transaction(func(tx *gorm.DB) error {
		var locked dao.UserInviterReward
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("id = ? AND inviter_player_id = ? AND claimed_at IS NULL", task.Request.RewardID, profile.PlayerID).
			First(&locked).Error; err != nil {
			if err == gorm.ErrRecordNotFound {
				return cmnErrors.ActionError("Reward not found or already claimed")
			}
			return err
		}
		now := time.Now().UTC()
		return tx.Model(&locked).Update("claimed_at", now).Error
	})
	if err != nil {
		if customErr, ok := err.(cmnErrors.Error); ok {
			return nil, customErr
		}
		log.Errorf("%s, mark inviter reward failed: %v", task.Request.RequestUUID, err)
		return nil, cmnErrors.OperateDbFailed()
	}

	task.Response.Point = row.Point
	return task.Response, nil
}
