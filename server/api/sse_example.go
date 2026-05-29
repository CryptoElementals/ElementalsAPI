package api

import (
	"context"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/CryptoElementals/common/log"
	"github.com/CryptoElementals/common/server/sse"
	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
	"github.com/mitchellh/mapstructure"
)

func init() {
	Register(SSE_EXAMPLE_LABEL, NewSSEExampleTask, NOAUTH)
}

type SSEExampleRequest struct {
	BaseRequest
	EventTypes []string `mapstructure:"EventTypes" validate:"required"`              // 订阅的事件类型
	Duration   int      `mapstructure:"Duration" validate:"required,min=1,max=3600"` // 连接持续时间（秒）
}

type SSEExampleResponse struct {
	BaseResponse
	Message string `json:"message"`
}

type SSEExampleTask struct {
	Request  *SSEExampleRequest
	Response *SSEExampleResponse
	mu       sync.Mutex
	stopChan chan struct{}
}

func NewSSEExampleRequest(data *map[string]interface{}) (*SSEExampleRequest, error) {
	req := &SSEExampleRequest{}
	err := mapstructure.Decode(*data, &req)
	if err != nil {
		return nil, err
	}
	req.BaseRequest.RequestUUID = (*data)["RequestUUID"].(string)
	return req, nil
}

func NewSSEExampleResponse(sessionId string) *SSEExampleResponse {
	return &SSEExampleResponse{
		BaseResponse: BaseResponse{
			Action:      SSE_EXAMPLE_LABEL + "Response",
			RequestUUID: sessionId,
		},
	}
}

func NewSSEExampleTask(data *map[string]interface{}) (Task, error) {
	req, err := NewSSEExampleRequest(data)
	if err != nil {
		return nil, err
	}
	task := &SSEExampleTask{
		Request:  req,
		Response: NewSSEExampleResponse(req.BaseRequest.RequestUUID),
		stopChan: make(chan struct{}),
	}

	validate := validator.New()
	err = validate.Struct(task.Request)
	if err != nil {
		return nil, err
	}

	return task, nil
}

// Run 实现事件驱动的 SSE 流式响应
func (task *SSEExampleTask) Run(c *gin.Context) (Response, error) {
	log.Infof("SSE Example started - EventTypes: %v, Duration: %d, RequestUUID: %s",
		task.Request.EventTypes, task.Request.Duration, task.Request.RequestUUID)

	// 发送开始事件
	startEvent := sse.Event{
		Type: sse.EventTypeStatusUpdate,
		Data: map[string]interface{}{
			"status":     "started",
			"eventTypes": task.Request.EventTypes,
			"duration":   task.Request.Duration,
		},
		RequestUUID: task.Request.RequestUUID,
	}
	if err := sse.Write(c.Writer, c.Writer.(http.Flusher), startEvent); err != nil {
		return nil, err
	}

	// 启动只针对当前连接的事件模拟器
	done := make(chan struct{})
	task.startEventSimulator(c.Request.Context(), c.Writer, c.Writer.(http.Flusher), task.Request.RequestUUID, done)

	// 等待连接结束
	select {
	case <-c.Request.Context().Done():
		log.Infof("SSE connection closed by client - RequestUUID: %s", task.Request.RequestUUID)
	case <-time.After(time.Duration(task.Request.Duration) * time.Second):
		log.Infof("SSE connection timeout - RequestUUID: %s", task.Request.RequestUUID)
	case <-task.stopChan:
		log.Infof("SSE connection stopped manually - RequestUUID: %s", task.Request.RequestUUID)
	}

	// 通知模拟器退出
	close(done)

	// 发送结束事件
	endEvent := sse.Event{
		Type: sse.EventTypeStatusUpdate,
		Data: map[string]interface{}{
			"status": "completed",
		},
		RequestUUID: task.Request.RequestUUID,
	}
	if err := sse.Write(c.Writer, c.Writer.(http.Flusher), endEvent); err != nil {
		return nil, err
	}

	return task.Response, nil
}

// startEventSimulator 只推送给当前连接
func (task *SSEExampleTask) startEventSimulator(ctx context.Context, writer http.ResponseWriter, flusher http.Flusher, requestUUID string, done chan struct{}) {
	go func() {
		eventTypes := []string{"data_change", "status_update", "notification", "error"}
		ticker := time.NewTicker(1 * time.Second)
		defer ticker.Stop()
		count := 0
		for {
			select {
			case <-ticker.C:
				count++
				idx := (count - 1) % len(eventTypes)
				now := time.Now().Format("2006-01-02 15:04:05")
				var event sse.Event
				switch eventTypes[idx] {
				case "data_change":
					event = sse.Event{
						Type: "data_change",
						Data: map[string]interface{}{
							"dataType": "user_balance",
							"oldValue": count - 1,
							"newValue": count,
						},
						RequestUUID: requestUUID,
						Metadata: map[string]interface{}{
							"user_id": "user123",
							"source":  "sse_example",
							"count":   count,
							"time":    now,
						},
					}
				case "status_update":
					event = sse.Event{
						Type: "status_update",
						Data: map[string]interface{}{
							"status": fmt.Sprintf("status_%d", count),
						},
						RequestUUID: requestUUID,
						Metadata: map[string]interface{}{
							"component": "sse_example",
							"count":     count,
							"time":      now,
						},
					}
				case "notification":
					event = sse.Event{
						Type: "notification",
						Data: map[string]interface{}{
							"message": fmt.Sprintf("Example notification #%d", count),
						},
						RequestUUID: requestUUID,
						Metadata: map[string]interface{}{
							"priority": "normal",
							"source":   "sse_example",
							"count":    count,
							"time":     now,
						},
					}
				case "error":
					event = sse.Event{
						Type: "error",
						Data: map[string]interface{}{
							"error": fmt.Sprintf("Example error #%d", count),
						},
						RequestUUID: requestUUID,
						Metadata: map[string]interface{}{
							"severity": "warning",
							"source":   "sse_example",
							"count":    count,
							"time":     now,
						},
					}
				}
				sse.Write(writer, flusher, event)
			case <-ctx.Done():
				return
			case <-done:
				return
			}
		}
	}()
}
