package api

import (
	"time"

	"github.com/CryptoElementals/common/rpc/client"
	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
	"github.com/mitchellh/mapstructure"
)

func init() {
	Register(HEALTH_CHECK_LABEL, NewHealthCheckTask, NOAUTH)
}

// HealthCheckRequest 请求结构体
type HealthCheckRequest struct {
	BaseRequest
	CheckConnection bool `mapstructure:"CheckConnection"` // 是否检查RoomServer连接
}

// HealthCheckResponse 响应结构体
type HealthCheckResponse struct {
	BaseResponse
	System     map[string]interface{} `json:"system"`
	Connection map[string]interface{} `json:"connection,omitempty"`
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
		BaseResponse: BaseResponse{
			Action:      HEALTH_CHECK_LABEL + "Response",
			RequestUUID: sessionId,
		},
	}
}

func NewHealthCheckTask(data *map[string]interface{}) (Task, error) {
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

func (task *HealthCheckTask) Run(c *gin.Context) (Response, error) {
	// 系统基本信息
	task.Response.System = map[string]interface{}{
		"status":    "healthy",
		"service":   "apiserver",
		"timestamp": time.Now().Format(time.RFC3339),
	}

	// 如果需要检查连接
	if task.Request.CheckConnection {
		healthMonitor := client.GetGlobalHealthMonitor()
		isHealthy := healthMonitor.CheckHealth()
		task.Response.Connection = healthMonitor.GetHealthStats()
		task.Response.Connection["current_status"] = isHealthy
	}

	task.Response.BaseResponse.RetCode = 0
	task.Response.BaseResponse.Message = "Health check completed"
	return task.Response, nil
}
