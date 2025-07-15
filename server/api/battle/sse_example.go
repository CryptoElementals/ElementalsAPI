package battle

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/CryptoElementals/common/log"
	"github.com/CryptoElementals/common/server/api"
	"github.com/CryptoElementals/common/server/events"
	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
	"github.com/mitchellh/mapstructure"
)

const (
	SSE_EXAMPLE_LABEL = "SSEExample"
)

type SSEExampleRequest struct {
	api.BaseRequest
	EventTypes []string `mapstructure:"EventTypes" validate:"required"`              // 订阅的事件类型
	Duration   int      `mapstructure:"Duration" validate:"required,min=1,max=3600"` // 连接持续时间（秒）
}

type SSEExampleResponse struct {
	api.BaseResponse
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
		BaseResponse: api.BaseResponse{
			Action:      SSE_EXAMPLE_LABEL + "Response",
			RequestUUID: sessionId,
		},
	}
}

func NewSSEExampleTask(data *map[string]interface{}) (api.Task, error) {
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

// Run 实现普通的 HTTP 响应
func (task *SSEExampleTask) Run(c *gin.Context) (api.Response, error) {
	task.Response.Message = fmt.Sprintf("SSE Example Task - EventTypes: %v, Duration: %d", task.Request.EventTypes, task.Request.Duration)
	return task.Response, nil
}

// RunSSE 实现事件驱动的 SSE 流式响应
func (task *SSEExampleTask) RunSSE(ctx context.Context, c *gin.Context, writer http.ResponseWriter, flusher http.Flusher, requestUUID string) error {
	log.Infof("SSE Example started - EventTypes: %v, Duration: %d, RequestUUID: %s",
		task.Request.EventTypes, task.Request.Duration, requestUUID)

	// 发送开始事件
	startEvent := events.Event{
		Type: events.EventTypeStatusUpdate,
		Data: map[string]interface{}{
			"status":     "started",
			"eventTypes": task.Request.EventTypes,
			"duration":   task.Request.Duration,
		},
		RequestUUID: requestUUID,
	}
	if err := sendSSEEvent(writer, flusher, startEvent); err != nil {
		return err
	}

	// 启动只针对当前连接的事件模拟器
	done := make(chan struct{})
	task.startEventSimulator(ctx, writer, flusher, requestUUID, done)

	// 等待连接结束
	select {
	case <-ctx.Done():
		log.Infof("SSE connection closed by client - RequestUUID: %s", requestUUID)
	case <-time.After(time.Duration(task.Request.Duration) * time.Second):
		log.Infof("SSE connection timeout - RequestUUID: %s", requestUUID)
	case <-task.stopChan:
		log.Infof("SSE connection stopped manually - RequestUUID: %s", requestUUID)
	}

	// 通知模拟器退出
	close(done)

	// 发送结束事件
	endEvent := events.Event{
		Type: events.EventTypeStatusUpdate,
		Data: map[string]interface{}{
			"status": "completed",
		},
		RequestUUID: requestUUID,
	}
	if err := sendSSEEvent(writer, flusher, endEvent); err != nil {
		return err
	}

	return nil
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
				var event events.Event
				switch eventTypes[idx] {
				case "data_change":
					event = events.Event{
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
					event = events.Event{
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
					event = events.Event{
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
					event = events.Event{
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
				sendSSEEvent(writer, flusher, event)
			case <-ctx.Done():
				return
			case <-done:
				return
			}
		}
	}()
}

// EventListener 事件监听器
type EventListener struct {
	eventTypes   []string
	writer       http.ResponseWriter
	flusher      http.Flusher
	requestUUID  string
	stopChan     chan struct{}
	eventManager *events.EventManager
}

func NewEventListener(eventTypes []string, writer http.ResponseWriter, flusher http.Flusher, requestUUID string) *EventListener {
	return &EventListener{
		eventTypes:   eventTypes,
		writer:       writer,
		flusher:      flusher,
		requestUUID:  requestUUID,
		stopChan:     make(chan struct{}),
		eventManager: events.GetEventManager(),
	}
}

func (el *EventListener) Start() {
	// 在实际应用中，这里会监听特定的事件源
	// 例如：数据库变化、消息队列、外部API等

	// 这里我们使用全局事件管理器来演示
	// 在实际应用中，你可能需要实现更具体的事件监听逻辑

	log.Infof("Event listener started for types: %v, RequestUUID: %s", el.eventTypes, el.requestUUID)
}

func (el *EventListener) Stop() {
	close(el.stopChan)
	log.Infof("Event listener stopped for RequestUUID: %s", el.requestUUID)
}

// sendSSEEvent 发送 SSE 事件
func sendSSEEvent(writer http.ResponseWriter, flusher http.Flusher, event events.Event) error {
	jsonData, err := json.Marshal(event)
	if err != nil {
		return err
	}

	// SSE 格式：data: {json}\n\n
	eventStr := fmt.Sprintf("data: %s\n\n", string(jsonData))
	_, err = writer.Write([]byte(eventStr))
	if err != nil {
		return err
	}

	flusher.Flush()
	return nil
}
