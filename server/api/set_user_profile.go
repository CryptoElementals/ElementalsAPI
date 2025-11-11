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
	Name    string `mapstructure:"Name" validate:"required,max=42"`
	Avatar  string `mapstructure:"Avatar" validate:"max=100"` // 文件名长度限制
	Address string `mapstructure:"Address"`
	Email   string `mapstructure:"Email"`
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
	if task.Request.Address != "" {
		lowercaseAddress := strings.ToLower(task.Request.Address)
		userProfile, err = db.GetUserProfileByAddress(lowercaseAddress)
		if err != nil {
			log.Errorf("%s, failed to get user profile for address %s: %v", task.Request.RequestUUID, lowercaseAddress, err)
			return nil, errors.GetUserProfileFailed(lowercaseAddress)
		}
	} else if task.Request.Email != "" {
		userProfile, err = db.GetUserProfileByEmail(task.Request.Email)
		if err != nil {
			log.Errorf("%s, failed to get user profile for email %s: %v", task.Request.RequestUUID, task.Request.Email, err)
			return nil, errors.GetUserProfileFailed(task.Request.Email)
		}
	} else {
		log.Errorf("%s, no address/email found in request", task.Request.RequestUUID)
		return nil, errors.MissingLoginCookie()
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

	log.Infof("%s, user profile updated successfully for user id %s", task.Request.RequestUUID, userProfile.UserID.String())
	return task.Response, nil
}
