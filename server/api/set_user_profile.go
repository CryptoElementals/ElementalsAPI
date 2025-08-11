package api

import (
	"strings"

	"github.com/CryptoElementals/common/db"
	"github.com/CryptoElementals/common/errors"
	"github.com/CryptoElementals/common/log"
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
	// 从请求中获取用户地址（由中间件设置）
	address := task.Request.Address
	if address == "" {
		log.Errorf("%s, no address found in request", task.Request.RequestUUID)
		return nil, errors.MissingLoginCookie()
	}

	// 将地址转换为小写，确保与数据库中存储的格式一致
	lowercaseAddress := strings.ToLower(address)

	// 获取用户档案
	userProfile, err := db.GetUserProfileByAddress(lowercaseAddress)
	if err != nil {
		log.Errorf("%s, failed to get user profile for address %s: %v", task.Request.RequestUUID, lowercaseAddress, err)
		return nil, errors.GetUserProfileFailed(lowercaseAddress)
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

	log.Infof("%s, user profile updated successfully for address %s", task.Request.RequestUUID, lowercaseAddress)
	return task.Response, nil
}
