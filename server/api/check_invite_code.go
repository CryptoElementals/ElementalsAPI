package api

import (
	"strings"

	"github.com/CryptoElementals/common/db"
	cmnErrors "github.com/CryptoElementals/common/errors"
	"github.com/CryptoElementals/common/server/invite"
	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
	"github.com/mitchellh/mapstructure"
)

func init() {
	Register(CHECK_INVITE_CODE_LABEL, NewCheckInviteCodeTask, NOAUTH)
}

type CheckInviteCodeRequest struct {
	BaseRequest
	InviteCode string `mapstructure:"InviteCode" validate:"required"`
}

type CheckInviteCodeResponse struct {
	BaseResponse
	WarnCode             int    `json:"WarnCode"`
	WarnMessage          string `json:"WarnMessage"`
	InviteeInitialPoints int    `json:"InviteeInitialPoints"`
	InviteSlotsRemaining int    `json:"InviteSlotsRemaining"`
	MaxInviteesPerCode   int    `json:"MaxInviteesPerCode"`
}

type CheckInviteCodeTask struct {
	Request  *CheckInviteCodeRequest
	Response *CheckInviteCodeResponse
}

func NewCheckInviteCodeRequest(data *map[string]interface{}) (*CheckInviteCodeRequest, error) {
	req := &CheckInviteCodeRequest{}
	if err := mapstructure.Decode(*data, &req); err != nil {
		return nil, err
	}
	req.BaseRequest.RequestUUID = (*data)["RequestUUID"].(string)
	return req, nil
}

func NewCheckInviteCodeResponse(sessionId string) *CheckInviteCodeResponse {
	return &CheckInviteCodeResponse{
		BaseResponse: BaseResponse{
			Action:      CHECK_INVITE_CODE_LABEL + "Response",
			RequestUUID: sessionId,
		},
	}
}

func NewCheckInviteCodeTask(data *map[string]interface{}) (Task, error) {
	req, err := NewCheckInviteCodeRequest(data)
	if err != nil {
		return nil, err
	}
	task := &CheckInviteCodeTask{
		Request:  req,
		Response: NewCheckInviteCodeResponse(req.BaseRequest.RequestUUID),
	}
	validate := validator.New()
	if err := validate.Struct(task.Request); err != nil {
		return nil, err
	}
	return task, nil
}

func (task *CheckInviteCodeTask) Run(c *gin.Context) (Response, error) {
	rewardPoints := invite.InviteeInitialPoints()
	outcome, err := db.CheckInviteCodePreLogin(task.Request.InviteCode)
	if err != nil {
		return nil, cmnErrors.OperateDbFailed()
	}
	task.Response.WarnCode = int(outcome.WarnCode)
	if outcome.WarnCode != cmnErrors.WarnCodeOK {
		task.Response.WarnMessage = outcome.Message
	}
	task.Response.InviteeInitialPoints = rewardPoints

	code := strings.TrimSpace(strings.ToUpper(task.Request.InviteCode))
	inviter, ierr := db.GetInviterByInviteCode(code)
	if ierr != nil {
		return nil, cmnErrors.OperateDbFailed()
	}
	maxCount := 0
	if inviter != nil {
		maxCount = inviter.MaxInviteCount
	}
	task.Response.MaxInviteesPerCode = maxCount

	if outcome.WarnCode == cmnErrors.WarnCodeOK && inviter != nil {
		remaining := maxCount - inviter.InviteCount
		if remaining < 0 {
			remaining = 0
		}
		task.Response.InviteSlotsRemaining = remaining
	}
	return task.Response, nil
}
