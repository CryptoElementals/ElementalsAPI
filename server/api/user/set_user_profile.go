package user

import (
	"strings"

	"github.com/CryptoElementals/common/db"
	"github.com/CryptoElementals/common/errors"
	"github.com/CryptoElementals/common/log"
	"github.com/CryptoElementals/common/server/api"
	"github.com/CryptoElementals/common/utils"
	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
	"github.com/mitchellh/mapstructure"
)

const SET_USER_PROFILE_LABEL = "SetUserProfile"

type SetUserProfileRequest struct {
	api.BaseRequest
	Name   string `mapstructure:"Name" validate:"required,max=42"`
	Avatar string `mapstructure:"Avatar" validate:"max=100"` // 文件名长度限制
}

type SetUserProfileResponse struct {
	api.BaseResponse
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
		BaseResponse: api.BaseResponse{
			Action:      SET_USER_PROFILE_LABEL + "Response",
			RequestUUID: sessionId,
		},
	}
}

func NewSetUserProfileTask(data *map[string]interface{}) (api.Task, error) {
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

func (task *SetUserProfileTask) Run(c *gin.Context) (api.Response, error) {
	// 从请求参数中获取用户地址（由中间件设置）
	_params, _ := c.Get("params")
	params, ok := _params.(*map[string]interface{})
	if !ok {
		log.Errorf("%s, params assert failed", task.Request.RequestUUID)
		return nil, errors.MissingLoginCookie()
	}

	address, ok := (*params)["Address"].(string)
	if !ok || address == "" {
		log.Errorf("%s, no address found in params", task.Request.RequestUUID)
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

// RegisterUserApis 注册用户相关API
func RegisterUserApis() {
	api.Register(SET_USER_PROFILE_LABEL, NewSetUserProfileTask, api.COOKIEAUTH)
	api.Register(GET_USER_PROFILE_LABEL, NewGetUserProfileTask, api.NOAUTH)
	api.Register(HAS_COLLECTED_DAILY_REWARD_LABEL, NewHasCollectedDailyRewardTask, api.COOKIEAUTH)
	api.Register(COLLECT_DAILY_REWARD_LABEL, NewCollectDailyRewardTask, api.COOKIEAUTH)
}
