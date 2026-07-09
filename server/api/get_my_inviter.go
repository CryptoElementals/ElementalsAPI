package api

import (
	"strconv"
	"strings"

	"github.com/CryptoElementals/common/db"
	cmnErrors "github.com/CryptoElementals/common/errors"
	"github.com/CryptoElementals/common/internal/playerlevel"
	"github.com/CryptoElementals/common/log"
	"github.com/CryptoElementals/common/server/invite"
	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
	"github.com/mitchellh/mapstructure"
)

func init() {
	Register(GET_MY_INVITER_LABEL, NewGetMyInviterTask, COOKIEAUTH)
}

type GetMyInviterRequest struct {
	BaseRequest
	PlayerID string `mapstructure:"PlayerID" validate:"required"`
}

type GetMyInviterResponse struct {
	BaseResponse
	HasInviter bool   `json:"HasInviter"`
	PlayerID   string `json:"PlayerID"`
	Username   string `json:"Username"`
	Level      int    `json:"Level"`
	InvitedAt  string `json:"InvitedAt,omitempty"`
	InviteCode string `json:"InviteCode,omitempty"`
}

type GetMyInviterTask struct {
	Request  *GetMyInviterRequest
	Response *GetMyInviterResponse
}

func NewGetMyInviterRequest(data *map[string]interface{}) (*GetMyInviterRequest, error) {
	req := &GetMyInviterRequest{}
	if err := mapstructure.Decode(*data, &req); err != nil {
		return nil, err
	}
	req.BaseRequest.RequestUUID = (*data)["RequestUUID"].(string)
	return req, nil
}

func NewGetMyInviterResponse(sessionId string) *GetMyInviterResponse {
	return &GetMyInviterResponse{
		BaseResponse: BaseResponse{
			Action:      GET_MY_INVITER_LABEL + "Response",
			RequestUUID: sessionId,
		},
	}
}

func NewGetMyInviterTask(data *map[string]interface{}) (Task, error) {
	req, err := NewGetMyInviterRequest(data)
	if err != nil {
		return nil, err
	}
	task := &GetMyInviterTask{
		Request:  req,
		Response: NewGetMyInviterResponse(req.BaseRequest.RequestUUID),
	}
	validate := validator.New()
	if err := validate.Struct(task.Request); err != nil {
		return nil, err
	}
	return task, nil
}

func (task *GetMyInviterTask) Run(c *gin.Context) (Response, error) {
	inviteeID, err := strconv.ParseInt(strings.TrimSpace(task.Request.PlayerID), 10, 64)
	if err != nil {
		return nil, cmnErrors.ParamsJudgeError("invalid player id")
	}
	relation, err := db.GetInviterByInviteePlayerID(inviteeID)
	if err != nil {
		log.Errorf("%s, get inviter relation failed: %v", task.Request.RequestUUID, err)
		return nil, cmnErrors.OperateDbFailed()
	}
	if relation == nil {
		task.Response.HasInviter = false
		return task.Response, nil
	}
	inviterProfile, err := db.GetUserProfileByPlayerIDInt(relation.InviterPlayerID)
	if err != nil {
		return nil, cmnErrors.GetUserProfileFailed(strconv.FormatInt(relation.InviterPlayerID, 10))
	}
	points, perr := invite.PlayerPointsFromLobby(db.EffectiveServerType(inviterProfile), relation.InviterPlayerID)
	if perr != nil {
		log.Errorf("%s, get inviter points failed: %v", task.Request.RequestUUID, perr)
		points = 0
	}
	task.Response.HasInviter = true
	task.Response.PlayerID = strconv.FormatInt(relation.InviterPlayerID, 10)
	task.Response.Username = inviterProfile.Name
	task.Response.Level = playerlevel.CalculateLevel(points)
	task.Response.InvitedAt = relation.CreatedAt.UTC().Format("2006-01-02T15:04:05Z")
	task.Response.InviteCode = relation.InviteCode
	return task.Response, nil
}
