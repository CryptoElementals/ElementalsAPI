package match

import (
	"github.com/CryptoElementals/common/server/api"
	"github.com/CryptoElementals/common/server/services"
	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
	"github.com/mitchellh/mapstructure"
)

const LEAVE_QUEUE_LABEL = "LeaveQueue"

// LeaveQueueRequest 请求结构体
type LeaveQueueRequest struct {
	api.BaseRequest
	Mode string `mapstructure:"Mode" validate:"required"`
}

// LeaveQueueResponse 响应结构体
type LeaveQueueResponse struct {
	api.BaseResponse
	Success bool `json:"success"`
}

type LeaveQueueTask struct {
	Request  *LeaveQueueRequest
	Response *LeaveQueueResponse
}

// 解码请求
func NewLeaveQueueRequest(data *map[string]interface{}) (*LeaveQueueRequest, error) {
	req := &LeaveQueueRequest{}
	err := mapstructure.Decode(*data, &req)
	if err != nil {
		return nil, err
	}
	req.BaseRequest.RequestUUID = (*data)["RequestUUID"].(string)
	return req, nil
}

func NewLeaveQueueResponse(sessionId string) *LeaveQueueResponse {
	return &LeaveQueueResponse{
		BaseResponse: api.BaseResponse{
			Action:      LEAVE_QUEUE_LABEL + "Response",
			RequestUUID: sessionId,
		},
	}
}

func NewLeaveQueueTask(data *map[string]interface{}) (api.Task, error) {
	req, err := NewLeaveQueueRequest(data)
	if err != nil {
		return nil, err
	}
	task := &LeaveQueueTask{
		Request:  req,
		Response: NewLeaveQueueResponse(req.BaseRequest.RequestUUID),
	}

	validate := validator.New()
	err = validate.Struct(task.Request)
	if err != nil {
		return nil, err
	}

	return task, nil
}

func (task *LeaveQueueTask) Run(c *gin.Context) (api.Response, error) {
	// 获取玩家地址（从认证中间件设置的params中获取）
	_params, _ := c.Get("params")
	params, ok := _params.(*map[string]interface{})
	if !ok {
		task.Response.BaseResponse.RetCode = 1001
		task.Response.BaseResponse.Message = "参数解析失败"
		task.Response.Success = false
		return task.Response, nil
	}

	address, ok := (*params)["Address"].(string)
	if !ok || address == "" {
		task.Response.BaseResponse.RetCode = 1001
		task.Response.BaseResponse.Message = "未获取到玩家地址"
		task.Response.Success = false
		return task.Response, nil
	}

	// 创建匹配队列服务
	matchService := services.NewMatchQueueService()

	// 离开匹配队列（传递model和address）
	err := matchService.LeaveQueue(task.Request.Mode, address)
	if err != nil {
		task.Response.BaseResponse.RetCode = 1002
		task.Response.BaseResponse.Message = "离开匹配队列失败: " + err.Error()
		task.Response.Success = false
		return task.Response, nil
	}

	task.Response.Success = true
	task.Response.BaseResponse.RetCode = 0
	task.Response.BaseResponse.Message = "已成功离开匹配队列"

	return task.Response, nil
}
