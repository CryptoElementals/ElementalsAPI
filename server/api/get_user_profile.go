package api

import (
	"math"
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
	Register(GET_USER_PROFILE_LABEL, NewGetUserProfileTask, NOAUTH)
}

type GetUserProfileRequest struct {
	BaseRequest
	Address string `mapstructure:"Address" validate:"required"`
}

// UserInfo 用户信息结构体
type UserInfo struct {
	Address            string            `json:"Address"`
	Name               string            `json:"Name"`
	AvatarName         string            `json:"AvatarName"`
	AvatarURL          string            `json:"AvatarURL"`
	BackgroundURL      string            `json:"BackgroundURL"`
	Points             int               `json:"Points"`
	TokenAmount        int               `json:"TokenAmount"`
	OverallGame        int               `json:"OverallGame"`
	WinningRate        float64           `json:"WinningRate"`
	Level              int               `json:"Level"`
	CurrentLevelPoints int               `json:"CurrentLevelPoints"`
	NextLevelPoints    int               `json:"NextLevelPoints"`
	CardStatInfo       []db.CardStatInfo `json:"CardStatInfo"`
}

type GetUserProfileResponse struct {
	BaseResponse
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
		BaseResponse: BaseResponse{
			Action:      GET_USER_PROFILE_LABEL + "Response",
			RequestUUID: sessionId,
		},
	}
}

// Level 阈值配置
var levelThresholds = []int{
	0, 5000, 10000, 20500, 35000, 54500, 81000, 120000, 175000, 260000,
	410000, 650000, 1050000, 1700000, 2800000, 4600000, 7500000, 12300000,
	20500000, 35000000, 60000000,
}

// calculateLevel 根据积分计算等级、当前等级所需积分和下一级所需积分
func calculateLevel(points int) (level int, currentLevelPoints int, nextLevelPoints int) {
	// 如果积分为0，返回等级0，当前等级需要0积分，下一级需要5000积分
	if points == 0 {
		return 0, 0, levelThresholds[1]
	}

	// 查找当前等级
	for i, threshold := range levelThresholds {
		if points < threshold {
			// 找到第一个超过当前积分的阈值，等级为 i-1
			level = i - 1
			// 当前等级所需积分
			if level >= 0 {
				currentLevelPoints = levelThresholds[level]
			} else {
				currentLevelPoints = 0
			}
			// 下一级所需积分
			if i < len(levelThresholds) {
				nextLevelPoints = levelThresholds[i]
			} else {
				// 已经是最高等级
				nextLevelPoints = levelThresholds[len(levelThresholds)-1]
			}
			return
		}
	}

	// 如果积分超过所有阈值，返回最高等级
	level = len(levelThresholds) - 1
	currentLevelPoints = levelThresholds[level]
	nextLevelPoints = levelThresholds[len(levelThresholds)-1]
	return
}

func NewGetUserProfileTask(data *map[string]interface{}) (Task, error) {
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

func (task *GetUserProfileTask) Run(c *gin.Context) (Response, error) {
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

	userStat, err := db.GetUserStatByAddress(lowercaseAddress)
	if err != nil {
		log.Errorf("%s, failed to get user stat for address %s: %v", task.Request.RequestUUID, lowercaseAddress, err)
	}
	winningRate := 0.00
	if userStat.TotalGameCount > 0 {
		winningRate = float64(userStat.WinCount) / float64(userStat.TotalGameCount)
		// 保留2位小数
		winningRate = math.Round(winningRate*100) / 100
	}

	// 从 user_token 表获取积分和代币信息（优先使用该表数据）
	var (
		points      int
		tokenAmount int
	)

	userToken, err := db.GetPlayerToken(c.Request.Context(), lowercaseAddress)
	if err == nil && userToken != nil {
		points = int(userToken.Points)
		tokenAmount = int(userToken.TokenAmount)
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

	// 计算等级、当前等级所需积分和下一级所需积分
	level, currentLevelPoints, nextLevelPoints := calculateLevel(points)

	// 构建用户信息
	task.Response.UserInfo = UserInfo{
		Address:            userProfile.Address,
		Name:               userProfile.Name,
		AvatarName:         userProfile.AvatarURL,
		AvatarURL:          "",
		BackgroundURL:      "",
		Points:             points,
		TokenAmount:        tokenAmount,
		OverallGame:        int(userStat.TotalGameCount),
		WinningRate:        winningRate,
		Level:              level,
		CurrentLevelPoints: currentLevelPoints,
		NextLevelPoints:    nextLevelPoints,
		CardStatInfo:       cardStatInfo,
	}

	// 将文件名转换为预签名URL
	if userProfile.AvatarURL != "" {
		avatarURL, err := utils.GetPresignedImageURL(userProfile.AvatarURL)
		if err != nil {
			log.Errorf("%s, failed to generate presigned avatar URL for %s: %v", task.Request.RequestUUID, userProfile.AvatarURL, err)
			// 如果生成失败，保持原文件名
		} else {
			task.Response.UserInfo.AvatarURL = avatarURL
		}
	}

	if userProfile.BackgroundURL != "" {
		backgroundURL, err := utils.GetPresignedImageURL(userProfile.BackgroundURL)
		if err != nil {
			log.Errorf("%s, failed to generate presigned background URL for %s: %v", task.Request.RequestUUID, userProfile.BackgroundURL, err)
			// 如果生成失败，保持原文件名
		} else {
			task.Response.UserInfo.BackgroundURL = backgroundURL
		}
	}

	log.Infof("%s, user profile retrieved successfully for address %s", task.Request.RequestUUID, lowercaseAddress)
	return task.Response, nil
}
