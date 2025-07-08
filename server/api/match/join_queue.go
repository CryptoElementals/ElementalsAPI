package match

import (
	"github.com/CryptoElementals/common/server/api"
	"github.com/CryptoElementals/common/server/services"
	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
	"github.com/mitchellh/mapstructure"
)

const JOIN_QUEUE_LABEL = "JoinQueue"

// JoinQueueRequest 请求结构体
type JoinQueueRequest struct {
	api.BaseRequest
	Model     string `mapstructure:"Model" validate:"required"`
	PublicKey string `mapstructure:"PublicKey" validate:"required"`
}

// JoinQueueResponse 响应结构体
type JoinQueueResponse struct {
	api.BaseResponse
}

type JoinQueueTask struct {
	Request  *JoinQueueRequest
	Response *JoinQueueResponse
}

// 解码请求
func NewJoinQueueRequest(data *map[string]interface{}) (*JoinQueueRequest, error) {
	req := &JoinQueueRequest{}
	err := mapstructure.Decode(*data, &req)
	if err != nil {
		return nil, err
	}
	req.BaseRequest.RequestUUID = (*data)["RequestUUID"].(string)
	return req, nil
}

func NewJoinQueueResponse(sessionId string) *JoinQueueResponse {
	return &JoinQueueResponse{
		BaseResponse: api.BaseResponse{
			Action:      JOIN_QUEUE_LABEL + "Response",
			RequestUUID: sessionId,
		},
	}
}

func NewJoinQueueTask(data *map[string]interface{}) (api.Task, error) {
	req, err := NewJoinQueueRequest(data)
	if err != nil {
		return nil, err
	}
	task := &JoinQueueTask{
		Request:  req,
		Response: NewJoinQueueResponse(req.BaseRequest.RequestUUID),
	}

	validate := validator.New()
	err = validate.Struct(task.Request)
	if err != nil {
		return nil, err
	}

	return task, nil
}

func (task *JoinQueueTask) Run(c *gin.Context) (api.Response, error) {
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

	// 加入匹配队列（传递model、address和publickey）
	err := matchService.JoinQueue(task.Request.Model, address, task.Request.PublicKey)
	if err != nil {
		task.Response.BaseResponse.RetCode = 1002
		task.Response.BaseResponse.Message = "加入匹配队列失败: " + err.Error()
		return task.Response, nil
	}

	// 暂时不进行匹配，只返回成功
	task.Response.BaseResponse.RetCode = 0
	task.Response.BaseResponse.Message = "已成功加入匹配队列"

	return task.Response, nil
}

// RegisterMatchApis 注册匹配相关API
func RegisterMatchApis() {
	api.Register(JOIN_QUEUE_LABEL, NewJoinQueueTask, api.COOKIEAUTH)
	api.Register(CHECK_MATCH_STATUS_LABEL, NewCheckMatchStatusTask, api.COOKIEAUTH)
	api.Register(LEAVE_QUEUE_LABEL, NewLeaveQueueTask, api.COOKIEAUTH)
}
