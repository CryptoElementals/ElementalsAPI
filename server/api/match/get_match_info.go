package match

import (
	"strings"

	"github.com/CryptoElementals/common/db"
	"github.com/CryptoElementals/common/log"
	"github.com/CryptoElementals/common/server/api"
	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
	"github.com/mitchellh/mapstructure"
)

const GET_MATCH_INFO_LABEL = "GetMatchInfo"

// GetMatchInfoRequest 请求结构体
type GetMatchInfoRequest struct {
	api.BaseRequest
	MatchID string `mapstructure:"MatchId" validate:"required"`
}

// MatchPlayer 匹配玩家信息
type MatchPlayer struct {
	Address   string `json:"Address"`   // 钱包地址
	Name      string `json:"Name"`      // 用户名
	AvatarURL string `json:"AvatarURL"` // 头像URL
	IsMyself  bool   `json:"IsMyself"`  // 是否是自己
	Confirmed bool   `json:"Confirmed"` // 是否已经确认Battle
}

// GetMatchInfoResponse 响应结构体
type GetMatchInfoResponse struct {
	api.BaseResponse
	Players []MatchPlayer `json:"Players"` // 双方的信息
}

type GetMatchInfoTask struct {
	Request  *GetMatchInfoRequest
	Response *GetMatchInfoResponse
}

// 解码请求
func NewGetMatchInfoRequest(data *map[string]interface{}) (*GetMatchInfoRequest, error) {
	req := &GetMatchInfoRequest{}
	err := mapstructure.Decode(*data, &req)
	if err != nil {
		return nil, err
	}
	req.BaseRequest.RequestUUID = (*data)["RequestUUID"].(string)
	return req, nil
}

func NewGetMatchInfoResponse(sessionId string) *GetMatchInfoResponse {
	return &GetMatchInfoResponse{
		BaseResponse: api.BaseResponse{
			Action:      GET_MATCH_INFO_LABEL + "Response",
			RequestUUID: sessionId,
		},
		Players: []MatchPlayer{},
	}
}

func NewGetMatchInfoTask(data *map[string]interface{}) (api.Task, error) {
	req, err := NewGetMatchInfoRequest(data)
	if err != nil {
		return nil, err
	}
	task := &GetMatchInfoTask{
		Request:  req,
		Response: NewGetMatchInfoResponse(req.BaseRequest.RequestUUID),
	}

	validate := validator.New()
	err = validate.Struct(task.Request)
	if err != nil {
		return nil, err
	}

	return task, nil
}

func (task *GetMatchInfoTask) Run(c *gin.Context) (api.Response, error) {
	// 获取玩家地址（从认证中间件设置的params中获取）
	_params, _ := c.Get("params")
	params, ok := _params.(*map[string]interface{})
	if !ok {
		task.Response.BaseResponse.RetCode = 1001
		task.Response.BaseResponse.Message = "Failed to parse parameters"
		return task.Response, nil
	}

	address, ok := (*params)["Address"].(string)
	if !ok || address == "" {
		task.Response.BaseResponse.RetCode = 1001
		task.Response.BaseResponse.Message = "Failed to get player address"
		return task.Response, nil
	}

	// 统一将地址转为小写
	address = strings.ToLower(address)
	log.Infof("[GetMatchInfo] Processing request for address: %s, matchId: %s", address, task.Request.MatchID)

	// 根据MatchID获取匹配记录
	matches, err := db.GetMatchesByMatchID(task.Request.MatchID)
	if err != nil {
		log.Infof("[GetMatchInfo] Error getting matches by matchId %s: %v", task.Request.MatchID, err)
		task.Response.BaseResponse.RetCode = 1002
		task.Response.BaseResponse.Message = "Match record does not exist"
		return task.Response, nil
	}

	// 检查是否找到匹配记录
	if len(matches) == 0 {
		log.Infof("[GetMatchInfo] No matches found for matchId: %s", task.Request.MatchID)
		task.Response.BaseResponse.RetCode = 1002
		task.Response.BaseResponse.Message = "Match record does not exist"
		return task.Response, nil
	}

	// 验证玩家是否是该匹配的参与者
	found := false
	for _, match := range matches {
		// 将数据库中的地址也转为小写进行比较
		matchAddress := strings.ToLower(match.Address)
		if matchAddress == address {
			found = true
			break
		}
	}

	if !found {
		log.Infof("[GetMatchInfo] Address %s is not a participant in match %s", address, task.Request.MatchID)
		task.Response.BaseResponse.RetCode = 1003
		task.Response.BaseResponse.Message = "You are not a participant in this match"
		return task.Response, nil
	}

	// 构建玩家信息列表
	var players []MatchPlayer
	for _, match := range matches {
		// 统一使用小写地址
		matchAddress := strings.ToLower(match.Address)

		// 获取用户档案信息
		userProfile, err := db.GetUserProfileByAddress(matchAddress)
		if err != nil {
			log.Infof("[GetMatchInfo] Failed to get user profile for address %s: %v", matchAddress, err)
			// 如果获取用户档案失败，使用默认值
			userProfile = nil
		}

		// 构建玩家信息
		player := MatchPlayer{
			Address:   matchAddress,
			IsMyself:  matchAddress == address,
			Confirmed: match.Status == "confirmed",
		}

		// 设置用户名和头像URL
		if userProfile != nil {
			player.Name = userProfile.Name
			player.AvatarURL = userProfile.AvatarURL
		} else {
			// 如果获取用户档案失败，使用地址作为默认用户名
			player.Name = matchAddress
			player.AvatarURL = ""
		}

		players = append(players, player)
	}

	task.Response.Players = players
	task.Response.BaseResponse.RetCode = 0
	task.Response.BaseResponse.Message = "Successfully retrieved match information"

	log.Infof("[GetMatchInfo] Successfully retrieved match info for matchId: %s, players count: %d", task.Request.MatchID, len(players))

	return task.Response, nil
}
