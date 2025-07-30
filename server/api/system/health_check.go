package system

import (
	"time"

	"github.com/CryptoElementals/common/server/api"
	"github.com/CryptoElementals/common/server/events"
	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
	"github.com/mitchellh/mapstructure"
)

const HEALTH_CHECK_LABEL = "HealthCheck"

// HealthCheckRequest 请求结构体
type HealthCheckRequest struct {
	api.BaseRequest
	CheckConnection bool `mapstructure:"CheckConnection"` // 是否检查RoomServer连接
}

// HealthCheckResponse 响应结构体
type HealthCheckResponse struct {
	api.BaseResponse
	System     map[string]interface{} `json:"system"`
	Connection map[string]interface{} `json:"connection,omitempty"`
	EventStats map[string]interface{} `json:"event_stats,omitempty"`
}

type HealthCheckTask struct {
	Request  *HealthCheckRequest
	Response *HealthCheckResponse
}

func NewHealthCheckRequest(data *map[string]interface{}) (*HealthCheckRequest, error) {
	req := &HealthCheckRequest{}
	err := mapstructure.Decode(*data, &req)
	if err != nil {
		return nil, err
	}
	req.BaseRequest.RequestUUID = (*data)["RequestUUID"].(string)
	return req, nil
}

func NewHealthCheckResponse(sessionId string) *HealthCheckResponse {
	return &HealthCheckResponse{
		BaseResponse: api.BaseResponse{
			Action:      HEALTH_CHECK_LABEL + "Response",
			RequestUUID: sessionId,
		},
	}
}

func NewHealthCheckTask(data *map[string]interface{}) (api.Task, error) {
	req, err := NewHealthCheckRequest(data)
	if err != nil {
		return nil, err
	}
	task := &HealthCheckTask{
		Request:  req,
		Response: NewHealthCheckResponse(req.BaseRequest.RequestUUID),
	}

	validate := validator.New()
	err = validate.Struct(task.Request)
	if err != nil {
		return nil, err
	}

	return task, nil
}

func (task *HealthCheckTask) Run(c *gin.Context) (api.Response, error) {
	// 系统基本信息
	task.Response.System = map[string]interface{}{
		"status":    "healthy",
		"service":   "apiserver",
		"timestamp": time.Now().Format(time.RFC3339),
	}

	// 如果需要检查连接
	if task.Request.CheckConnection {
		healthMonitor := events.GetGlobalHealthMonitor()

		// 执行实时健康检查
		isHealthy := healthMonitor.CheckHealth()

		// 获取连接统计信息
		task.Response.Connection = healthMonitor.GetHealthStats()
		task.Response.Connection["current_status"] = isHealthy

		// 获取事件管理器统计
		eventManager := events.GetGlobalEventManager()
		task.Response.EventStats = eventManager.GetStats()
	}

	task.Response.BaseResponse.RetCode = 0
	task.Response.BaseResponse.Message = "Health check completed"
	return task.Response, nil
}

// RegisterSystemApis 注册系统相关API
func RegisterSystemApis() {
	api.Register(HEALTH_CHECK_LABEL, NewHealthCheckTask, api.NOAUTH)
	api.Register(GET_CARDS_LABEL, NewGetAllCardsTask, api.NOAUTH)
	api.Register(LIST_AVATARS_LABEL, NewListAvatarsTask, api.NOAUTH)
}
