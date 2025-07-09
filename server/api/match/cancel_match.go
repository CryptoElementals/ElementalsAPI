package match

import (
	"github.com/CryptoElementals/common/db"
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

	// 验证匹配状态，只能取消未确认的匹配
	if len(matches) > 0 && matches[0].Status != "matched" {
		task.Response.BaseResponse.RetCode = 1004
		task.Response.BaseResponse.Message = "只能取消未确认的匹配"
		return task.Response, nil
	}

	// 将匹配状态设置为已取消
	err = db.UpdateMatchStatus(task.Request.MatchID, "cancelled")
	if err != nil {
		task.Response.BaseResponse.RetCode = 1005
		task.Response.BaseResponse.Message = "取消匹配失败"
		return task.Response, nil
	}

	task.Response.BaseResponse.RetCode = 0
	task.Response.BaseResponse.Message = "匹配已取消"

	return task.Response, nil
}
