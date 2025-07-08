package match

import (
	"github.com/CryptoElementals/common/server/api"
	"github.com/CryptoElementals/common/server/services"
	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
	"github.com/mitchellh/mapstructure"
)

const CHECK_MATCH_STATUS_LABEL = "CheckMatchStatus"

// CheckMatchStatusRequest 请求结构体
type CheckMatchStatusRequest struct {
	api.BaseRequest
	Model string `mapstructure:"Model" validate:"required"`
}

// CheckMatchStatusResponse 响应结构体
type CheckMatchStatusResponse struct {
	api.BaseResponse
	Status     string `json:"status"`      // waiting, not_in_queue
	QueueCount int    `json:"queue_count"` // 当前队列中的玩家数量
}

type CheckMatchStatusTask struct {
	Request  *CheckMatchStatusRequest
	Response *CheckMatchStatusResponse
}

// 解码请求
func NewCheckMatchStatusRequest(data *map[string]interface{}) (*CheckMatchStatusRequest, error) {
	req := &CheckMatchStatusRequest{}
	err := mapstructure.Decode(*data, &req)
	if err != nil {
		return nil, err
	}
	req.BaseRequest.RequestUUID = (*data)["RequestUUID"].(string)
	return req, nil
}

func NewCheckMatchStatusResponse(sessionId string) *CheckMatchStatusResponse {
	return &CheckMatchStatusResponse{
		BaseResponse: api.BaseResponse{
			Action:      CHECK_MATCH_STATUS_LABEL + "Response",
			RequestUUID: sessionId,
		},
	}
}

func NewCheckMatchStatusTask(data *map[string]interface{}) (api.Task, error) {
	req, err := NewCheckMatchStatusRequest(data)
	if err != nil {
		return nil, err
	}
	task := &CheckMatchStatusTask{
		Request:  req,
		Response: NewCheckMatchStatusResponse(req.BaseRequest.RequestUUID),
	}

	validate := validator.New()
	err = validate.Struct(task.Request)
	if err != nil {
		return nil, err
	}

	return task, nil
}

func (task *CheckMatchStatusTask) Run(c *gin.Context) (api.Response, error) {
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

	// 创建匹配队列服务
	matchService := services.NewMatchQueueService()

	// 获取当前队列
	players, err := matchService.GetQueue(task.Request.Model)
	if err != nil {
		task.Response.BaseResponse.RetCode = 1002
		task.Response.BaseResponse.Message = "获取队列状态失败: " + err.Error()
		return task.Response, nil
	}

	// 检查玩家是否在队列中
	task.Response.QueueCount = len(players)
	inQueue := false
	for _, player := range players {
		if player.Address == address {
			inQueue = true
			break
		}
	}

	if inQueue {
		task.Response.Status = "waiting"
		task.Response.BaseResponse.Message = "玩家在匹配队列中等待"
	} else {
		task.Response.Status = "not_in_queue"
		task.Response.BaseResponse.Message = "玩家不在匹配队列中"
	}

	task.Response.BaseResponse.RetCode = 0
	return task.Response, nil
}
