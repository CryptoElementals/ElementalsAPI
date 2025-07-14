package match

import (
	"strings"

	"github.com/CryptoElementals/common/db"
	"github.com/CryptoElementals/common/log"
	dao "github.com/CryptoElementals/common/models"
	"github.com/CryptoElementals/common/server/api"
	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
	"github.com/mitchellh/mapstructure"
)

const CANCEL_MATCH_LABEL = "CancelMatch"

// CancelMatchRequest 请求结构体
type CancelMatchRequest struct {
	api.BaseRequest
	MatchID string `mapstructure:"MatchId" validate:"required"`
}

// CancelMatchResponse 响应结构体
type CancelMatchResponse struct {
	api.BaseResponse
}

type CancelMatchTask struct {
	Request  *CancelMatchRequest
	Response *CancelMatchResponse
}

// 解码请求
func NewCancelMatchRequest(data *map[string]interface{}) (*CancelMatchRequest, error) {
	req := &CancelMatchRequest{}
	err := mapstructure.Decode(*data, &req)
	if err != nil {
		return nil, err
	}
	req.BaseRequest.RequestUUID = (*data)["RequestUUID"].(string)
	return req, nil
}

func NewCancelMatchResponse(sessionId string) *CancelMatchResponse {
	return &CancelMatchResponse{
		BaseResponse: api.BaseResponse{
			Action:      CANCEL_MATCH_LABEL + "Response",
			RequestUUID: sessionId,
		},
	}
}

func NewCancelMatchTask(data *map[string]interface{}) (api.Task, error) {
	req, err := NewCancelMatchRequest(data)
	if err != nil {
		return nil, err
	}
	task := &CancelMatchTask{
		Request:  req,
		Response: NewCancelMatchResponse(req.BaseRequest.RequestUUID),
	}

	validate := validator.New()
	err = validate.Struct(task.Request)
	if err != nil {
		return nil, err
	}

	return task, nil
}

func (task *CancelMatchTask) Run(c *gin.Context) (api.Response, error) {
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
	log.Infof("[CancelMatch] Processing request for address: %s, matchId: %s", address, task.Request.MatchID)

	// 根据MatchID获取匹配记录
	matches, err := db.GetMatchesByMatchID(task.Request.MatchID)
	if err != nil {
		log.Infof("[CancelMatch] Error getting matches by matchId %s: %v", task.Request.MatchID, err)
		task.Response.BaseResponse.RetCode = 1002
		task.Response.BaseResponse.Message = "Match record does not exist"
		return task.Response, nil
	}

	// 验证玩家是否是该匹配的参与者
	found := false
	var currentPlayerMatch dao.GamePlayer
	for _, match := range matches {
		// 将数据库中的地址也转为小写进行比较
		matchAddress := strings.ToLower(match.Address)
		if matchAddress == address {
			found = true
			currentPlayerMatch = match
			break
		}
	}

	if !found {
		log.Infof("[CancelMatch] Address %s is not a participant in match %s", address, task.Request.MatchID)
		task.Response.BaseResponse.RetCode = 1003
		task.Response.BaseResponse.Message = "You are not a participant in this match"
		return task.Response, nil
	}

	// 检查当前玩家的状态，只有未确认的玩家才能取消
	switch currentPlayerMatch.Status {
	case "confirmed":
		log.Infof("[CancelMatch] Address %s has already confirmed, cannot cancel match %s", address, task.Request.MatchID)
		task.Response.BaseResponse.RetCode = 1004
		task.Response.BaseResponse.Message = "You have already confirmed, cannot cancel match"
		return task.Response, nil
	case "cancelled":
		log.Infof("[CancelMatch] Match %s has already been cancelled", task.Request.MatchID)
		task.Response.BaseResponse.RetCode = 1006
		task.Response.BaseResponse.Message = "Match has already been cancelled"
		return task.Response, nil
	case "matched":
		// 可以取消，继续执行
		log.Infof("[CancelMatch] Address %s can cancel match %s", address, task.Request.MatchID)
	default:
		log.Infof("[CancelMatch] Invalid match status for address %s in match %s: %s", address, task.Request.MatchID, currentPlayerMatch.Status)
		task.Response.BaseResponse.RetCode = 1007
		task.Response.BaseResponse.Message = "Invalid match status, cannot cancel"
		return task.Response, nil
	}

	// 将匹配状态设置为已取消
	err = db.UpdateMatchStatus(task.Request.MatchID, "cancelled")
	if err != nil {
		log.Infof("[CancelMatch] Failed to cancel match %s: %v", task.Request.MatchID, err)
		task.Response.BaseResponse.RetCode = 1005
		task.Response.BaseResponse.Message = "Failed to cancel match"
		return task.Response, nil
	}

	log.Infof("[CancelMatch] Successfully cancelled match %s by address %s", task.Request.MatchID, address)
	task.Response.BaseResponse.RetCode = 0
	task.Response.BaseResponse.Message = "Match cancelled successfully"

	return task.Response, nil
}
