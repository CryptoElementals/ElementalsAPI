package api

import (
	"github.com/CryptoElementals/common/db"
	"github.com/CryptoElementals/common/log"
	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
	"github.com/mitchellh/mapstructure"
)

func init() {
	Register(IS_USER_LOGGED_IN_LABEL, NewIsUserLoggedInTask, VERIFYAUTH)
}

type IsUserLoggedInRequest struct {
	BaseRequest
	RefreshToken string `mapstructure:"RefreshToken" validate:"required"`
}

type IsUserLoggedInResponse struct {
	BaseResponse
	UserLoggedIn bool   `json:"UserLoggedIn"`
	PlayerID     string `json:"PlayerID,omitempty"`
	Address      string `json:"Address,omitempty"`
	Email        string `json:"Email,omitempty"`
}

type IsUserLoggedInTask struct {
	Request  *IsUserLoggedInRequest
	Response *IsUserLoggedInResponse
}

// 将 map 类型的数据解码为 IsUserLoggedInRequest 结构体，并提取 RequestUUID
func NewIsUserLoggedInRequest(data *map[string]interface{}) (*IsUserLoggedInRequest, error) {
	req := &IsUserLoggedInRequest{}
	err := mapstructure.Decode(*data, &req)
	if err != nil {
		return nil, err
	}
	req.BaseRequest.RequestUUID = (*data)["RequestUUID"].(string)

	return req, nil
}

func NewIsUserLoggedInResponse(sessionId string) *IsUserLoggedInResponse {
	return &IsUserLoggedInResponse{
		BaseResponse: BaseResponse{
			Action:      IS_USER_LOGGED_IN_LABEL + "Response",
			RequestUUID: sessionId,
		},
	}
}

func NewIsUserLoggedInTask(data *map[string]interface{}) (Task, error) {
	req, err := NewIsUserLoggedInRequest(data)
	if err != nil {
		return nil, err
	}
	task := &IsUserLoggedInTask{
		Request:  req,
		Response: NewIsUserLoggedInResponse(req.BaseRequest.RequestUUID),
	}

	validate := validator.New()
	err = validate.Struct(task.Request)
	if err != nil {
		return nil, err
	}

	return task, nil
}

func (task *IsUserLoggedInTask) Run(c *gin.Context) (Response, error) {
	refreshToken := task.Request.RefreshToken

	// 验证 RefreshToken 是否有效
	playerID, err := getUserIdByRefreshToken(refreshToken)
	if err != nil {
		// RefreshToken 无效或过期
		log.Infof("%s, refresh token invalid or expired: %s", task.Request.RequestUUID, refreshToken)
		task.Response.UserLoggedIn = false
		task.Response.PlayerID = ""
		task.Response.Address = ""
		task.Response.Email = ""
		return task.Response, nil
	}

	// RefreshToken 有效，基于 player_id 查用户档案，返回 Address/Email
	profile, err := db.GetUserProfileByPlayerID(playerID)
	if err != nil || profile == nil {
		log.Infof("%s, user profile not found for player_id: %s", task.Request.RequestUUID, playerID)
		task.Response.UserLoggedIn = false
		return task.Response, nil
	}
	log.Infof("%s, user logged in: player_id=%s addr=%s email=%s", task.Request.RequestUUID, playerID, profile.Address, profile.Email)
	task.Response.UserLoggedIn = true
	task.Response.PlayerID = playerID
	task.Response.Address = profile.Address
	task.Response.Email = profile.Email
	return task.Response, nil
}
