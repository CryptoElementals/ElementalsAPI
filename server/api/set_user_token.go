package api

import (
	"strings"

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
	Register(SET_USER_TOKEN_LABEL, NewSetUserTokenTask, NOAUTH)
}

// SetUserTokenRequest 前端联调接口，待删除
// 注意：PlayerID 只能是 2010551417005150208 或 6300839975063552
type SetUserTokenRequest struct {
	BaseRequest
	PlayerID string `mapstructure:"PlayerID" validate:"required"`
	Token    int32  `mapstructure:"Token" validate:"required"`
}

type SetUserTokenResponse struct {
	BaseResponse
}

type SetUserTokenTask struct {
	Request  *SetUserTokenRequest
	Response *SetUserTokenResponse
}

func NewSetUserTokenRequest(data *map[string]interface{}) (*SetUserTokenRequest, error) {
	req := &SetUserTokenRequest{}
	if err := mapstructure.Decode(*data, req); err != nil {
		return nil, err
	}
	// 透传 RequestUUID
	if v, ok := (*data)["RequestUUID"].(string); ok {
		req.BaseRequest.RequestUUID = v
	}
	return req, nil
}

func NewSetUserTokenResponse(sessionId string) *SetUserTokenResponse {
	return &SetUserTokenResponse{
		BaseResponse: BaseResponse{
			Action:      SET_USER_TOKEN_LABEL + "Response",
			RequestUUID: sessionId,
		},
	}
}

func NewSetUserTokenTask(data *map[string]interface{}) (Task, error) {
	req, err := NewSetUserTokenRequest(data)
	if err != nil {
		return nil, err
	}
	task := &SetUserTokenTask{
		Request:  req,
		Response: NewSetUserTokenResponse(req.BaseRequest.RequestUUID),
	}

	validate := validator.New()
	if err := validate.Struct(task.Request); err != nil {
		return nil, err
	}

	return task, nil
}

func (task *SetUserTokenTask) Run(c *gin.Context) (Response, error) {
	requestPlayerID := strings.TrimSpace(task.Request.PlayerID)

	// 只允许两个指定的 playerId
	if requestPlayerID != "2010551417005150208" && requestPlayerID != "6300839975063552" {
		log.Errorf("%s, SetUserToken not allowed for player_id=%s", task.Request.RequestUUID, requestPlayerID)
		return nil, cmnErrors.ActionError("player_id not allowed")
	}

	profile, err := db.GetUserProfileByPlayerID(requestPlayerID)
	if err != nil {
		log.Errorf("%s, failed to get user profile by player_id=%s: %v", task.Request.RequestUUID, requestPlayerID, err)
		return nil, cmnErrors.GetUserProfileFailed(requestPlayerID)
	}

	targetToken := task.Request.Token

	var userToken *dao.UserToken
	userToken, err = db.GetPlayerToken(c.Request.Context(), profile.PlayerID)
	if err != nil && err != gorm.ErrRecordNotFound {
		log.Errorf("%s, failed to get user token for player_id=%s: %v", task.Request.RequestUUID, requestPlayerID, err)
		return nil, cmnErrors.OperateDbFailed()
	}

	if userToken == nil {
		userToken = &dao.UserToken{
			PlayerId:    profile.PlayerID,
			Points:      0,
			TokenAmount: targetToken,
		}
	} else {
		userToken.TokenAmount = targetToken
	}

	if err = db.SaveUserToken(*userToken); err != nil {
		log.Errorf("%s, failed to save user token for player_id=%s: %v", task.Request.RequestUUID, requestPlayerID, err)
		return nil, cmnErrors.OperateDbFailed()
	}

	log.Infof("%s, SetUserToken success for player_id=%s, new token amount=%d", task.Request.RequestUUID, requestPlayerID, targetToken)
	return task.Response, nil
}
