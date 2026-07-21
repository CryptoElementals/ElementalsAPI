package api

import (
	"strconv"
	"strings"

	"github.com/CryptoElementals/common/db"
	cmnErrors "github.com/CryptoElementals/common/errors"
	"github.com/CryptoElementals/common/log"
	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
	"github.com/mitchellh/mapstructure"
)

func init() {
	Register(GET_INVITEE_INITIAL_REWARD_LABEL, NewGetInviteeInitialRewardTask, COOKIEAUTH)
}

type GetInviteeInitialRewardRequest struct {
	BaseRequest
	PlayerID string `mapstructure:"PlayerID" validate:"required"`
}

type GetInviteeInitialRewardResponse struct {
	BaseResponse
	HasReward bool  `json:"HasReward"`
	Claimed   bool  `json:"Claimed"`
	Point     int32 `json:"Point"`
}

type GetInviteeInitialRewardTask struct {
	Request  *GetInviteeInitialRewardRequest
	Response *GetInviteeInitialRewardResponse
}

func NewGetInviteeInitialRewardRequest(data *map[string]interface{}) (*GetInviteeInitialRewardRequest, error) {
	req := &GetInviteeInitialRewardRequest{}
	if err := mapstructure.Decode(*data, &req); err != nil {
		return nil, err
	}
	req.BaseRequest.RequestUUID = (*data)["RequestUUID"].(string)
	return req, nil
}

func NewGetInviteeInitialRewardResponse(sessionId string) *GetInviteeInitialRewardResponse {
	return &GetInviteeInitialRewardResponse{
		BaseResponse: BaseResponse{
			Action:      GET_INVITEE_INITIAL_REWARD_LABEL + "Response",
			RequestUUID: sessionId,
		},
	}
}

func NewGetInviteeInitialRewardTask(data *map[string]interface{}) (Task, error) {
	req, err := NewGetInviteeInitialRewardRequest(data)
	if err != nil {
		return nil, err
	}
	task := &GetInviteeInitialRewardTask{
		Request:  req,
		Response: NewGetInviteeInitialRewardResponse(req.BaseRequest.RequestUUID),
	}
	validate := validator.New()
	if err := validate.Struct(task.Request); err != nil {
		return nil, err
	}
	return task, nil
}

func (task *GetInviteeInitialRewardTask) Run(c *gin.Context) (Response, error) {
	playerID := strings.TrimSpace(task.Request.PlayerID)
	inviteeID, err := strconv.ParseInt(playerID, 10, 64)
	if err != nil {
		return nil, cmnErrors.ParamsJudgeError("invalid player id")
	}
	status, err := db.GetInviteeInitialRewardStatus(inviteeID)
	if err != nil {
		log.Errorf("%s, get invitee initial reward failed for player_id=%s: %v", task.Request.RequestUUID, playerID, err)
		return nil, cmnErrors.OperateDbFailed()
	}
	task.Response.HasReward = status.HasReward
	task.Response.Claimed = status.Claimed
	task.Response.Point = status.Point
	return task.Response, nil
}
