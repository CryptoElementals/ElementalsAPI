package api

import (
	"strconv"
	"strings"

	"github.com/CryptoElementals/common/db"
	cmnErrors "github.com/CryptoElementals/common/errors"
	"github.com/CryptoElementals/common/internal/invite"
	"github.com/CryptoElementals/common/internal/playerlevel"
	"github.com/CryptoElementals/common/log"
	serverinvite "github.com/CryptoElementals/common/server/invite"
	dao "github.com/CryptoElementals/common/models"
	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
	"github.com/mitchellh/mapstructure"
)

func init() {
	Register(LIST_INVITED_USERS_LABEL, NewListInvitedUsersTask, COOKIEAUTH)
}

type ListInvitedUsersRequest struct {
	BaseRequest
	PlayerID string `mapstructure:"PlayerID" validate:"required"`
	Limit    int    `mapstructure:"Limit"`
	Offset   int    `mapstructure:"Offset"`
}

type InvitedUserItem struct {
	PlayerID           string `json:"PlayerID"`
	Username           string `json:"Username"`
	Level              int    `json:"Level"`
	RewardStatus       string `json:"RewardStatus"`
	NextMilestoneLevel int    `json:"NextMilestoneLevel"`
}

type ListInvitedUsersResponse struct {
	BaseResponse
	Users []InvitedUserItem `json:"Users"`
}

type ListInvitedUsersTask struct {
	Request  *ListInvitedUsersRequest
	Response *ListInvitedUsersResponse
}

func NewListInvitedUsersRequest(data *map[string]interface{}) (*ListInvitedUsersRequest, error) {
	req := &ListInvitedUsersRequest{}
	if err := mapstructure.Decode(*data, &req); err != nil {
		return nil, err
	}
	req.BaseRequest.RequestUUID = (*data)["RequestUUID"].(string)
	return req, nil
}

func NewListInvitedUsersResponse(sessionId string) *ListInvitedUsersResponse {
	return &ListInvitedUsersResponse{
		BaseResponse: BaseResponse{
			Action:      LIST_INVITED_USERS_LABEL + "Response",
			RequestUUID: sessionId,
		},
		Users: []InvitedUserItem{},
	}
}

func NewListInvitedUsersTask(data *map[string]interface{}) (Task, error) {
	req, err := NewListInvitedUsersRequest(data)
	if err != nil {
		return nil, err
	}
	task := &ListInvitedUsersTask{
		Request:  req,
		Response: NewListInvitedUsersResponse(req.BaseRequest.RequestUUID),
	}
	validate := validator.New()
	if err := validate.Struct(task.Request); err != nil {
		return nil, err
	}
	return task, nil
}

func (task *ListInvitedUsersTask) Run(c *gin.Context) (Response, error) {
	playerIDStr := strings.TrimSpace(task.Request.PlayerID)
	inviterID, err := strconv.ParseInt(playerIDStr, 10, 64)
	if err != nil {
		return nil, cmnErrors.ParamsJudgeError("invalid player id")
	}
	profile, err := db.GetUserProfileByPlayerID(playerIDStr)
	if err != nil {
		return nil, cmnErrors.GetUserProfileFailed(playerIDStr)
	}
	if db.EffectiveServerType(profile) != dao.ServerTypeNormal {
		return task.Response, nil
	}
	rows, err := db.ListUserInviteRelationsByInviter(inviterID, task.Request.Limit, task.Request.Offset)
	if err != nil {
		log.Errorf("%s, list invited users failed: %v", task.Request.RequestUUID, err)
		return nil, cmnErrors.OperateDbFailed()
	}
	items := make([]InvitedUserItem, 0, len(rows))
	for _, row := range rows {
		inviteeProfile, perr := db.GetUserProfileByPlayerIDInt(row.InviteePlayerID)
		inviteeServerType := dao.ServerTypeTrial
		if perr == nil && inviteeProfile != nil {
			inviteeServerType = db.EffectiveServerType(inviteeProfile)
		}
		points, perr := serverinvite.PlayerPointsFromLobby(inviteeServerType, row.InviteePlayerID)
		if perr != nil {
			log.Errorf("%s, get invitee points failed: %v", task.Request.RequestUUID, perr)
			points = 0
		}
		inviteeLevel := playerlevel.CalculateLevel(points)
		milestones, merr := db.ListInviterMilestoneRewardsForInvitee(inviterID, row.InviteePlayerID)
		if merr != nil {
			return nil, cmnErrors.OperateDbFailed()
		}
		rewardStatus, nextMilestoneLevel := invite.DeriveInviterRewardStatus(inviteeLevel, milestones)
		items = append(items, InvitedUserItem{
			PlayerID:           strconv.FormatInt(row.InviteePlayerID, 10),
			Username:           row.InviteeName,
			Level:              inviteeLevel,
			RewardStatus:       string(rewardStatus),
			NextMilestoneLevel: nextMilestoneLevel,
		})
	}
	task.Response.Users = items
	return task.Response, nil
}
