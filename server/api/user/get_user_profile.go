package user

import (
	"strings"

	"github.com/CryptoElementals/common/db"
	"github.com/CryptoElementals/common/errors"
	"github.com/CryptoElementals/common/log"
	dao "github.com/CryptoElementals/common/models"
	"github.com/CryptoElementals/common/server/api"
	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
	"github.com/mitchellh/mapstructure"
)

const GET_USER_PROFILE_LABEL = "GetUserProfile"

type GetUserProfileRequest struct {
	api.BaseRequest
	Address string `mapstructure:"Address" validate:"required"`
}

// UserInfo 用户信息结构体
type UserInfo struct {
	Address      string            `json:"Address"`
	Name         string            `json:"Name"`
	AvatarURL    string            `json:"AvatarURL"`
	Points       int               `json:"Points"`
	TokenAmount  int               `json:"TokenAmount"`
	OverallGame  int               `json:"OverallGame"`
	WinningRate  float64           `json:"WinningRate"`
	CardStatInfo []db.CardStatInfo `json:"CardStatInfo"`
}

type GetUserProfileResponse struct {
	api.BaseResponse
	UserInfo UserInfo `json:"UserInfo"`
}

type GetUserProfileTask struct {
	Request  *GetUserProfileRequest
	Response *GetUserProfileResponse
}

// 将 map 类型的数据解码为 GetUserProfileRequest 结构体，并提取 RequestUUID
func NewGetUserProfileRequest(data *map[string]interface{}) (*GetUserProfileRequest, error) {
	req := &GetUserProfileRequest{}
	err := mapstructure.Decode(*data, &req)
	if err != nil {
		return nil, err
	}
	req.BaseRequest.RequestUUID = (*data)["RequestUUID"].(string)
	return req, nil
}

func NewGetUserProfileResponse(sessionId string) *GetUserProfileResponse {
	return &GetUserProfileResponse{
		BaseResponse: api.BaseResponse{
			Action:      GET_USER_PROFILE_LABEL + "Response",
			RequestUUID: sessionId,
		},
	}
}

func NewGetUserProfileTask(data *map[string]interface{}) (api.Task, error) {
	req, err := NewGetUserProfileRequest(data)
	if err != nil {
		return nil, err
	}
	task := &GetUserProfileTask{
		Request:  req,
		Response: NewGetUserProfileResponse(req.BaseRequest.RequestUUID),
	}

	validate := validator.New()
	err = validate.Struct(task.Request)
	if err != nil {
		return nil, err
	}

	return task, nil
}

func (task *GetUserProfileTask) Run(c *gin.Context) (api.Response, error) {
	// 从请求参数中获取用户地址
	address := task.Request.Address
	if address == "" {
		log.Errorf("%s, no address provided in request", task.Request.RequestUUID)
		return nil, errors.MissingParams("Address")
	}

	// 将地址转换为小写，确保与数据库中存储的格式一致
	lowercaseAddress := strings.ToLower(address)

	// 获取用户档案
	userProfile, err := db.GetUserProfileByAddress(lowercaseAddress)
	if err != nil {
		log.Errorf("%s, failed to get user profile for address %s: %v", task.Request.RequestUUID, lowercaseAddress, err)
		return nil, errors.GetUserProfileFailed(lowercaseAddress)
	}

	// 获取用户卡牌统计信息
	cardStats, err := db.GetCardStatsByAddress(lowercaseAddress)
	if err != nil {
		log.Errorf("%s, failed to get card stats for address %s: %v", task.Request.RequestUUID, lowercaseAddress, err)
		// 卡牌统计获取失败不影响主要功能，设置为空数组
		cardStats = []dao.CardStat{}
	}

	// 转换为API响应格式
	cardStatInfo := db.GetCardStatsInfo(cardStats)

	// 构建用户信息
	task.Response.UserInfo = UserInfo{
		Address:      userProfile.Address,
		Name:         userProfile.Name,
		AvatarURL:    userProfile.AvatarURL,
		Points:       userProfile.Points,
		TokenAmount:  userProfile.TokenAmount,
		OverallGame:  userProfile.OverallGame,
		WinningRate:  userProfile.WinningRate,
		CardStatInfo: cardStatInfo,
	}

	log.Infof("%s, user profile retrieved successfully for address %s", task.Request.RequestUUID, lowercaseAddress)
	return task.Response, nil
}
