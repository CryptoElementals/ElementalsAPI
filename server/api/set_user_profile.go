package api

import (
	"strings"

	"github.com/CryptoElementals/common/db"
	"github.com/CryptoElementals/common/errors"
	"github.com/CryptoElementals/common/log"
	dao "github.com/CryptoElementals/common/models"
	"github.com/CryptoElementals/common/utils"
	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
	"github.com/mitchellh/mapstructure"
)

func init() {
	Register(SET_USER_PROFILE_LABEL, NewSetUserProfileTask, COOKIEAUTH)
}

type SetUserProfileRequest struct {
	BaseRequest
	Name     string `mapstructure:"Name" validate:"required,max=42"`
	Avatar   string `mapstructure:"Avatar" validate:"max=100"` // 文件名长度限制
	PlayerID string `mapstructure:"PlayerID" validate:"required"`
}

type SetUserProfileResponse struct {
	BaseResponse
}

type SetUserProfileTask struct {
	Request  *SetUserProfileRequest
	Response *SetUserProfileResponse
}

// 将 map 类型的数据解码为 SetUserProfileRequest 结构体，并提取 RequestUUID
func NewSetUserProfileRequest(data *map[string]interface{}) (*SetUserProfileRequest, error) {
	req := &SetUserProfileRequest{}
	err := mapstructure.Decode(*data, &req)
	if err != nil {
		return nil, err
	}
	req.BaseRequest.RequestUUID = (*data)["RequestUUID"].(string)
	return req, nil
}

func NewSetUserProfileResponse(sessionId string) *SetUserProfileResponse {
	return &SetUserProfileResponse{
		BaseResponse: BaseResponse{
			Action:      SET_USER_PROFILE_LABEL + "Response",
			RequestUUID: sessionId,
		},
	}
}

func NewSetUserProfileTask(data *map[string]interface{}) (Task, error) {
	req, err := NewSetUserProfileRequest(data)
	if err != nil {
		return nil, err
	}
	task := &SetUserProfileTask{
		Request:  req,
		Response: NewSetUserProfileResponse(req.BaseRequest.RequestUUID),
	}

	validate := validator.New()
	err = validate.Struct(task.Request)
	if err != nil {
		return nil, err
	}

	return task, nil
}

func (task *SetUserProfileTask) Run(c *gin.Context) (Response, error) {
	// 从请求中获取用户身份（由中间件基于会话注入）
	var (
		userProfile *dao.UserProfile
		err         error
	)
	log.Infof("%s, set user profile request: %+v", task.Request.RequestUUID, task.Request)
	userProfile, err = db.GetUserProfileByPlayerID(strings.TrimSpace(task.Request.PlayerID))
	if err != nil || userProfile == nil {
		log.Errorf("%s, failed to get user profile by player_id=%s: %v", task.Request.RequestUUID, task.Request.PlayerID, err)
		return nil, errors.GetUserProfileFailed(task.Request.PlayerID)
	}

	// 更新用户档案
	userProfile.Name = task.Request.Name
	if task.Request.Avatar != "" {
		// 直接使用传入的文件名
		avatarFilename := task.Request.Avatar

		// 构造对应的背景文件名
		backgroundFilename := utils.GetBackgroundFilenameFromAvatarFilename(avatarFilename)

		// 存储文件名
		userProfile.AvatarURL = avatarFilename
		userProfile.BackgroundURL = backgroundFilename
	}

	// 保存到数据库
	err = db.UpdateUserProfile(userProfile)
	if err != nil {
		log.Errorf("%s, failed to update user profile: %v", task.Request.RequestUUID, err)
		return nil, errors.SaveUserProfileFailed()
	}

	log.Infof("%s, user profile updated successfully for player id %d", task.Request.RequestUUID, userProfile.PlayerID)
	return task.Response, nil
}
