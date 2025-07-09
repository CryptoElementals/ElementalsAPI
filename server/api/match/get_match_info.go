package match

import (
	"github.com/CryptoElementals/common/db"
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
		task.Response.BaseResponse.Message = "参数解析失败"
		return task.Response, nil
	}

	address, ok := (*params)["Address"].(string)
	if !ok || address == "" {
		task.Response.BaseResponse.RetCode = 1001
		task.Response.BaseResponse.Message = "未获取到玩家地址"
		return task.Response, nil
	}

	// 根据MatchID获取匹配记录
	matches, err := db.GetMatchesByMatchID(task.Request.MatchID)
	if err != nil {
		task.Response.BaseResponse.RetCode = 1002
		task.Response.BaseResponse.Message = "匹配记录不存在"
		return task.Response, nil
	}

	// 检查是否找到匹配记录
	if len(matches) == 0 {
		task.Response.BaseResponse.RetCode = 1002
		task.Response.BaseResponse.Message = "匹配记录不存在"
		return task.Response, nil
	}

	// 验证玩家是否是该匹配的参与者
	found := false
	for _, match := range matches {
		if match.Address == address {
			found = true
			break
		}
	}

	if !found {
		task.Response.BaseResponse.RetCode = 1003
		task.Response.BaseResponse.Message = "您不是该匹配的参与者"
		return task.Response, nil
	}

	// 构建玩家信息列表
	var players []MatchPlayer
	for _, match := range matches {
		player := MatchPlayer{
			Address:   match.Address,
			IsMyself:  match.Address == address,
			Confirmed: match.Status == "confirmed",
		}
		players = append(players, player)
	}

	task.Response.Players = players
	task.Response.BaseResponse.RetCode = 0
	task.Response.BaseResponse.Message = "获取匹配信息成功"

	return task.Response, nil
}
