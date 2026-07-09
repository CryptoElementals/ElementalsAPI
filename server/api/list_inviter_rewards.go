package api

import (
	"strconv"
	"strings"

	"github.com/CryptoElementals/common/db"
	cmnErrors "github.com/CryptoElementals/common/errors"
	"github.com/CryptoElementals/common/internal/playerlevel"
	"github.com/CryptoElementals/common/log"
	"github.com/CryptoElementals/common/server/invite"
	dao "github.com/CryptoElementals/common/models"
	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
	"github.com/mitchellh/mapstructure"
)

func init() {
	Register(LIST_INVITER_REWARDS_LABEL, NewListInviterRewardsTask, COOKIEAUTH)
}

type ListInviterRewardsRequest struct {
	BaseRequest
	PlayerID          string `mapstructure:"PlayerID" validate:"required"`
	InviteePlayerID   string `mapstructure:"InviteePlayerID"`
}

type InviterRewardItem struct {
	RewardID        uint   `json:"RewardID"`
	InviteePlayerID string `json:"InviteePlayerID"`
	InviteeUsername string `json:"InviteeUsername"`
	MilestoneLevel  int    `json:"MilestoneLevel"`
	Point           int32  `json:"Point"`
	Claimed         bool   `json:"Claimed"`
	Claimable       bool   `json:"Claimable"`
	InviteeLevel    int    `json:"InviteeLevel"`
}

type ListInviterRewardsResponse struct {
	BaseResponse
	Rewards []InviterRewardItem `json:"Rewards"`
}

type ListInviterRewardsTask struct {
	Request  *ListInviterRewardsRequest
	Response *ListInviterRewardsResponse
}

func NewListInviterRewardsRequest(data *map[string]interface{}) (*ListInviterRewardsRequest, error) {
	req := &ListInviterRewardsRequest{}
	if err := mapstructure.Decode(*data, &req); err != nil {
		return nil, err
	}
	req.BaseRequest.RequestUUID = (*data)["RequestUUID"].(string)
	return req, nil
}

func NewListInviterRewardsResponse(sessionId string) *ListInviterRewardsResponse {
	return &ListInviterRewardsResponse{
		BaseResponse: BaseResponse{
			Action:      LIST_INVITER_REWARDS_LABEL + "Response",
			RequestUUID: sessionId,
		},
		Rewards: []InviterRewardItem{},
	}
}

func NewListInviterRewardsTask(data *map[string]interface{}) (Task, error) {
	req, err := NewListInviterRewardsRequest(data)
	if err != nil {
		return nil, err
	}
	task := &ListInviterRewardsTask{
		Request:  req,
		Response: NewListInviterRewardsResponse(req.BaseRequest.RequestUUID),
	}
	validate := validator.New()
	if err := validate.Struct(task.Request); err != nil {
		return nil, err
	}
	return task, nil
}

func (task *ListInviterRewardsTask) Run(c *gin.Context) (Response, error) {
	inviterID, err := strconv.ParseInt(strings.TrimSpace(task.Request.PlayerID), 10, 64)
	if err != nil {
		return nil, cmnErrors.ParamsJudgeError("invalid player id")
	}
	profile, err := db.GetUserProfileByPlayerID(task.Request.PlayerID)
	if err != nil {
		return nil, cmnErrors.GetUserProfileFailed(task.Request.PlayerID)
	}
	if db.EffectiveServerType(profile) != dao.ServerTypeNormal {
		return task.Response, nil
	}
	var optionalInvitee int64
	if strings.TrimSpace(task.Request.InviteePlayerID) != "" {
		optionalInvitee, err = strconv.ParseInt(strings.TrimSpace(task.Request.InviteePlayerID), 10, 64)
		if err != nil {
			return nil, cmnErrors.ParamsJudgeError("invalid invitee player id")
		}
	}
	rows, err := db.ListInviterRewards(inviterID, optionalInvitee)
	if err != nil {
		log.Errorf("%s, list inviter rewards failed: %v", task.Request.RequestUUID, err)
		return nil, cmnErrors.OperateDbFailed()
	}
	items := make([]InviterRewardItem, 0, len(rows))
	for _, row := range rows {
		inviteeProfile, perr := db.GetUserProfileByPlayerIDInt(row.InviteePlayerID)
		inviteeServerType := dao.ServerTypeTrial
		if perr == nil && inviteeProfile != nil {
			inviteeServerType = db.EffectiveServerType(inviteeProfile)
		}
		points, perr := invite.PlayerPointsFromLobby(inviteeServerType, row.InviteePlayerID)
		if perr != nil {
			log.Errorf("%s, get invitee points failed: %v", task.Request.RequestUUID, perr)
			points = 0
		}
		items = append(items, InviterRewardItem{
			RewardID:        row.RewardID,
			InviteePlayerID: strconv.FormatInt(row.InviteePlayerID, 10),
			InviteeUsername: row.InviteeName,
			MilestoneLevel:  row.MilestoneLevel,
			Point:           row.Point,
			Claimed:         row.Claimed,
			Claimable:       row.Claimable,
			InviteeLevel:    playerlevel.CalculateLevel(points),
		})
	}
	task.Response.Rewards = items
	return task.Response, nil
}
