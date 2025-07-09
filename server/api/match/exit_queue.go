package match

import (
	"github.com/CryptoElementals/common/server/api"
	"github.com/CryptoElementals/common/server/services"
	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
	"github.com/mitchellh/mapstructure"
)

const EXIT_QUEUE_LABEL = "ExitQueue"

// ExitQueueRequest 请求结构体
type ExitQueueRequest struct {
	api.BaseRequest
	Mode string `mapstructure:"Mode" validate:"required"`
}

// ExitQueueResponse 响应结构体
type ExitQueueResponse struct {
	api.BaseResponse
}

type ExitQueueTask struct {
	Request  *ExitQueueRequest
	Response *ExitQueueResponse
}

// 解码请求
func NewExitQueueRequest(data *map[string]interface{}) (*ExitQueueRequest, error) {
	req := &ExitQueueRequest{}
	err := mapstructure.Decode(*data, &req)
	if err != nil {
		return nil, err
	}
	req.BaseRequest.RequestUUID = (*data)["RequestUUID"].(string)
	return req, nil
}

func NewExitQueueResponse(sessionId string) *ExitQueueResponse {
	return &ExitQueueResponse{
		BaseResponse: api.BaseResponse{
			Action:      EXIT_QUEUE_LABEL + "Response",
			RequestUUID: sessionId,
		},
	}
}

func NewExitQueueTask(data *map[string]interface{}) (api.Task, error) {
	req, err := NewExitQueueRequest(data)
	if err != nil {
		return nil, err
	}
	task := &ExitQueueTask{
		Request:  req,
		Response: NewExitQueueResponse(req.BaseRequest.RequestUUID),
	}

	validate := validator.New()
	err = validate.Struct(task.Request)
	if err != nil {
		return nil, err
	}

	return task, nil
}

func (task *ExitQueueTask) Run(c *gin.Context) (api.Response, error) {
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

	// 离开匹配队列（传递model和address）
	err := matchService.LeaveQueue(task.Request.Mode, address)
	if err != nil {
		task.Response.BaseResponse.RetCode = 1002
		task.Response.BaseResponse.Message = "离开匹配队列失败: " + err.Error()
		return task.Response, nil
	}

	task.Response.BaseResponse.RetCode = 0
	task.Response.BaseResponse.Message = "已成功离开匹配队列"

	return task.Response, nil
}
