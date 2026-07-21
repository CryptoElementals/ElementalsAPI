package api

import (
	"context"
	"math"
	"strconv"
	"strings"

	"github.com/CryptoElementals/common/db"
	"github.com/CryptoElementals/common/errors"
	"github.com/CryptoElementals/common/internal/playerlevel"
	"github.com/CryptoElementals/common/log"
	dao "github.com/CryptoElementals/common/models"
	"github.com/CryptoElementals/common/rpc/client"
	"github.com/CryptoElementals/common/rpc/proto"
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
	PlayerID string `mapstructure:"PlayerID" validate:"required"`
}

// UserInfo 用户信息结构体
type UserInfo struct {
	PlayerID           string            `json:"PlayerID"`
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
	ServerType         string            `json:"ServerType"`
	InviteCode         string            `json:"InviteCode"`
	InvitedCount       int               `json:"InvitedCount"`
	MaxInviteesPerCode int               `json:"MaxInviteesPerCode"`
	InviteEnabled      bool              `json:"InviteEnabled"`
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

// calculateLevel 根据积分计算等级、当前等级所需积分和下一级所需积分
func calculateLevel(points int) (level int, currentLevelPoints int, nextLevelPoints int) {
	return playerlevel.CalculateLevelDetail(points)
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
	var (
		userProfile *dao.UserProfile
		err         error
	)
	playerID := strings.TrimSpace(task.Request.PlayerID)
	userProfile, err = db.GetUserProfileByPlayerID(playerID)
	if err != nil {
		log.Errorf("%s, failed to get user profile by player_id=%s: %v", task.Request.RequestUUID, playerID, err)
		return nil, errors.GetUserProfileFailed(playerID)
	}
	lookupAddress := strings.ToLower(strings.TrimSpace(userProfile.Address))

	// 默认统计为 0；仅当有可用地址时查询统计
	winningRate := 0.00
	points := 0
	tokenAmount := 0
	totalGameCount := 0

	// 统一基于 PlayerID 查询统计、代币、卡牌统计
	if userStat, e := db.GetUserStatByPlayerID(playerID); e == nil && userStat != nil {
		totalGameCount = int(userStat.TotalGameCount)
		if userStat.TotalGameCount > 0 {
			winningRate = float64(userStat.WinCount) / float64(userStat.TotalGameCount)
			winningRate = math.Round(winningRate*100) / 100
		}
	} else if e != nil {
		log.Errorf("%s, failed to get user stat by player_id=%s: %v", task.Request.RequestUUID, playerID, e)
	}

	lobbyClient := client.LobbyClientForType(db.EffectiveServerType(userProfile))
	if lobbyClient != nil {
		if userToken, e := lobbyClient.GetPlayerToken(context.Background(), &proto.GetPlayerTokenRequest{Id: userProfile.PlayerID}); e == nil && userToken != nil {
			points = int(userToken.GetPoints())
			tokenAmount = int(userToken.GetTokens())
		}
	}

	cardStatInfo := []db.CardStatInfo{}
	if cardStats, e := db.GetCardStatsByPlayerID(playerID); e == nil {
		cardStatInfo = db.GetCardStatsInfo(cardStats)
	}
	level, currentLevelPoints, nextLevelPoints := calculateLevel(points)
	serverType := db.EffectiveServerType(userProfile)
	inviteEnabled := serverType == dao.ServerTypeNormal
	inviteCode := ""
	invitedCount := 0
	maxInviteesPerCode := 0
	if inviteEnabled {
		if row, ierr := db.GetInviteCodeByPlayerID(userProfile.PlayerID); ierr == nil && row != nil {
			inviteCode = row.InviteCode
			invitedCount = row.InviteCount
			maxInviteesPerCode = row.MaxInviteCount
		}
	}
	task.Response.UserInfo = UserInfo{
		PlayerID:           strconv.FormatInt(userProfile.PlayerID, 10),
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
		ServerType:         serverType,
		InviteCode:         inviteCode,
		InvitedCount:       invitedCount,
		MaxInviteesPerCode: maxInviteesPerCode,
		InviteEnabled:      inviteEnabled,
		CardStatInfo:       cardStatInfo,
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

	log.Infof("%s, user profile retrieved successfully (player_id=%s, addr=%s)", task.Request.RequestUUID, playerID, lookupAddress)
	return task.Response, nil
}
