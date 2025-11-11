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
	Address string `mapstructure:"Address"`
	Email   string `mapstructure:"Email"`
}

// UserInfo 用户信息结构体
type UserInfo struct {
	UserID             string            `json:"UserID"`
	Address            string            `json:"Address"`
	Email              string            `json:"Email"`
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
	// 允许通过 Address 或 Email 查询，至少提供一个
	var (
		userProfile    *dao.UserProfile
		err            error
		lookupAddress  string
		requestAddress = strings.ToLower(strings.TrimSpace(task.Request.Address))
		requestEmail   = strings.TrimSpace(task.Request.Email)
	)

	if requestAddress == "" && requestEmail == "" {
		log.Errorf("%s, neither address nor email provided", task.Request.RequestUUID)
		return nil, errors.MissingParams("Address or Email")
	}

	if requestAddress != "" {
		lookupAddress = requestAddress
		userProfile, err = db.GetUserProfileByAddress(lookupAddress)
	} else {
		userProfile, err = db.GetUserProfileByEmail(requestEmail)
		if err == nil && userProfile != nil {
			lookupAddress = strings.ToLower(strings.TrimSpace(userProfile.Address))
		}
	}
	if err != nil {
		log.Errorf("%s, failed to get user profile (addr=%s,email=%s): %v", task.Request.RequestUUID, lookupAddress, requestEmail, err)
		if lookupAddress != "" {
			return nil, errors.GetUserProfileFailed(lookupAddress)
		}
		return nil, errors.GetUserProfileFailed(requestEmail)
	}

	// 默认统计为 0；仅当有可用地址时查询统计
	winningRate := 0.00
	points := 0
	tokenAmount := 0
	totalGameCount := 0

	if lookupAddress != "" {
		if userStat, e := db.GetUserStatByAddress(lookupAddress); e == nil && userStat != nil {
			totalGameCount = int(userStat.TotalGameCount)
			if userStat.TotalGameCount > 0 {
				winningRate = float64(userStat.WinCount) / float64(userStat.TotalGameCount)
				winningRate = math.Round(winningRate*100) / 100
			}
		} else if e != nil {
			log.Errorf("%s, failed to get user stat for address %s: %v", task.Request.RequestUUID, lookupAddress, e)
		}

		if userToken, e := db.GetPlayerToken(c.Request.Context(), lookupAddress); e == nil && userToken != nil {
			points = int(userToken.Points)
			tokenAmount = int(userToken.TokenAmount)
		}

		if cardStats, e := db.GetCardStatsByAddress(lookupAddress); e == nil {
			cardStatInfo := db.GetCardStatsInfo(cardStats)
			level, currentLevelPoints, nextLevelPoints := calculateLevel(points)
			task.Response.UserInfo = UserInfo{
				UserID:             userProfile.UserID.String(),
				Address:            userProfile.Address,
				Email:              userProfile.Email,
				Name:               userProfile.Name,
				AvatarName:         userProfile.AvatarURL,
				AvatarURL:          "",
				BackgroundURL:      "",
				Points:             points,
				TokenAmount:        tokenAmount,
				OverallGame:        totalGameCount,
				WinningRate:        winningRate,
				Level:              level,
				CurrentLevelPoints: currentLevelPoints,
				NextLevelPoints:    nextLevelPoints,
				CardStatInfo:       cardStatInfo,
			}
		} else {
			// 无法获取卡牌统计，不影响主要信息
			level, currentLevelPoints, nextLevelPoints := calculateLevel(points)
			task.Response.UserInfo = UserInfo{
				UserID:             userProfile.UserID.String(),
				Address:            userProfile.Address,
				Email:              userProfile.Email,
				Name:               userProfile.Name,
				AvatarName:         userProfile.AvatarURL,
				AvatarURL:          "",
				BackgroundURL:      "",
				Points:             points,
				TokenAmount:        tokenAmount,
				OverallGame:        totalGameCount,
				WinningRate:        winningRate,
				Level:              level,
				CurrentLevelPoints: currentLevelPoints,
				NextLevelPoints:    nextLevelPoints,
				CardStatInfo:       []db.CardStatInfo{},
			}
		}
	} else {
		// 无地址（仅邮箱）情况下仅返回基础信息
		task.Response.UserInfo = UserInfo{
			UserID:        userProfile.UserID.String(),
			Address:       userProfile.Address,
			Email:         userProfile.Email,
			Name:          userProfile.Name,
			AvatarName:    userProfile.AvatarURL,
			AvatarURL:     "",
			BackgroundURL: "",
		}
	}

	if userProfile.AvatarURL != "" {
		if avatarURL, err := utils.GetPresignedImageURL(userProfile.AvatarURL); err == nil {
			task.Response.UserInfo.AvatarURL = avatarURL
		} else {
			log.Errorf("%s, failed to generate presigned avatar URL for %s: %v", task.Request.RequestUUID, userProfile.AvatarURL, err)
		}
	}

	if userProfile.BackgroundURL != "" {
		if backgroundURL, err := utils.GetPresignedImageURL(userProfile.BackgroundURL); err == nil {
			task.Response.UserInfo.BackgroundURL = backgroundURL
		} else {
			log.Errorf("%s, failed to generate presigned background URL for %s: %v", task.Request.RequestUUID, userProfile.BackgroundURL, err)
		}
	}

	log.Infof("%s, user profile retrieved successfully (addr=%s,email=%s)", task.Request.RequestUUID, lookupAddress, requestEmail)
	return task.Response, nil
}
