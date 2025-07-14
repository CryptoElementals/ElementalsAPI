package match

import (
	"strings"

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
		task.Response.BaseResponse.Message = "Parameter parsing failed"
		return task.Response, nil
	}

	address, ok := (*params)["Address"].(string)
	if !ok || address == "" {
		task.Response.BaseResponse.RetCode = 1001
		task.Response.BaseResponse.Message = "Failed to get player address"
		return task.Response, nil
	}

	// 将地址转换为小写，确保与数据库中存储的格式一致
	lowercaseAddress := strings.ToLower(address)

	// 验证游戏模式
	validModes := []string{"PvP", "Tournament"}
	modeValid := false
	for _, validMode := range validModes {
		if task.Request.Mode == validMode {
			modeValid = true
			break
		}
	}
	if !modeValid {
		task.Response.BaseResponse.RetCode = 1005
		task.Response.BaseResponse.Message = "Invalid game mode. Only PvP and Tournament are supported"
		return task.Response, nil
	}

	// 创建匹配队列服务
	matchService := services.NewMatchQueueService()

	// 离开匹配队列（传递小写地址）
	err := matchService.LeaveQueue(task.Request.Mode, lowercaseAddress)
	if err != nil {
		task.Response.BaseResponse.RetCode = 1002
		task.Response.BaseResponse.Message = "Failed to leave match queue: " + err.Error()
		return task.Response, nil
	}

	task.Response.BaseResponse.RetCode = 0
	task.Response.BaseResponse.Message = "Successfully left match queue"

	return task.Response, nil
}
