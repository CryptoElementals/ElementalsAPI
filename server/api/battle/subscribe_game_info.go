package battle

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/CryptoElementals/common/log"
	"github.com/CryptoElementals/common/rpc/proto"
	"github.com/CryptoElementals/common/server/api"
	"github.com/CryptoElementals/common/server/events"
	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
	"github.com/mitchellh/mapstructure"
)

const SUBSCRIBE_GAME_INFO_LABEL = "SubscribeGameInfo"

// SubscribeGameInfoRequest 请求结构体
type SubscribeGameInfoRequest struct {
	api.BaseRequest
	TempAddress string `mapstructure:"TempAddress" validate:"required"`     // 临时地址
	Duration    int    `mapstructure:"Duration" validate:"min=1,max=86400"` // 连接持续时间（秒）
}

// SubscribeGameInfoResponse 响应结构体
type SubscribeGameInfoResponse struct {
	api.BaseResponse
	Message string `json:"message"`
}

type SubscribeGameInfoTask struct {
	Request  *SubscribeGameInfoRequest
	Response *SubscribeGameInfoResponse
	mu       sync.Mutex
	stopChan chan struct{}
}

// 解码请求
func NewSubscribeGameInfoRequest(data *map[string]interface{}) (*SubscribeGameInfoRequest, error) {
	req := &SubscribeGameInfoRequest{}
	err := mapstructure.Decode(*data, &req)
	if err != nil {
		return nil, err
	}
	req.BaseRequest.RequestUUID = (*data)["RequestUUID"].(string)
	// 设置默认值为 86400秒（1天）
	if req.Duration == 0 {
		req.Duration = 86400
	}
	return req, nil
}

func NewSubscribeGameInfoResponse(sessionId string) *SubscribeGameInfoResponse {
	return &SubscribeGameInfoResponse{
		BaseResponse: api.BaseResponse{
			Action:      SUBSCRIBE_GAME_INFO_LABEL + "Response",
			RequestUUID: sessionId,
		},
	}
}

func NewSubscribeGameInfoTask(data *map[string]interface{}) (api.Task, error) {
	req, err := NewSubscribeGameInfoRequest(data)
	if err != nil {
		return nil, err
	}
	task := &SubscribeGameInfoTask{
		Request:  req,
		Response: NewSubscribeGameInfoResponse(req.BaseRequest.RequestUUID),
		stopChan: make(chan struct{}, 1),
	}

	validate := validator.New()
	err = validate.Struct(task.Request)
	if err != nil {
		return nil, err
	}

	return task, nil
}

// Run 实现普通的 HTTP 响应
func (task *SubscribeGameInfoTask) Run(c *gin.Context) (api.Response, error) {
	task.Response.Message = fmt.Sprintf("SubscribeGameInfo Task - TempAddress: %s, Duration: %d", task.Request.TempAddress, task.Request.Duration)
	return task.Response, nil
}

// RunSSE 实现事件驱动的 SSE 流式响应
func (task *SubscribeGameInfoTask) RunSSE(ctx context.Context, c *gin.Context, writer http.ResponseWriter, flusher http.Flusher, requestUUID string) error {
	// 获取玩家地址（从认证中间件设置的params中获取）
	_params, _ := c.Get("params")
	params, ok := _params.(*map[string]interface{})
	if !ok {
		return fmt.Errorf("parameter parsing failed")
	}

	address, ok := (*params)["Address"].(string)
	if !ok || address == "" {
		return fmt.Errorf("failed to get player address")
	}

	// 将地址转换为小写，确保与数据库中存储的格式一致
	address = strings.ToLower(address)
	temp_address := strings.ToLower(task.Request.TempAddress)

	// 组装 game_topic: address_tempaddress 格式
	game_topic := fmt.Sprintf("%s_%s", address, temp_address)

	// 发送开始事件
	startEvent := events.Event{
		Type: events.EventTypeStatusUpdate,
		Data: map[string]interface{}{
			"Status": "started",
		},
		Timestamp:   time.Now(),
		RequestUUID: requestUUID,
	}
	if err := sendSSEEvent(writer, flusher, startEvent); err != nil {
		return err
	}

	// 获取全局事件管理器
	eventManager := events.GetGlobalEventManager()

	// 注册SSE客户端
	clientID := fmt.Sprintf("%s_%s", requestUUID, game_topic)
	eventHandler := func(msg *proto.Message) {
		// 将RoomServer事件转换为SSE事件并发送
		sseEvent := task.convertRoomServerEventToSSE(msg, requestUUID)
		if err := sendSSEEvent(writer, flusher, sseEvent); err != nil {
			log.Errorf("发送SSE事件失败: %v", err)
		}
	}

	eventManager.RegisterSSEClient(clientID, eventHandler)
	defer eventManager.UnregisterSSEClient(clientID)

	// 订阅游戏主题
	if err := eventManager.SubscribeToTopic(clientID, game_topic); err != nil {
		log.Errorf("订阅主题失败: %v", err)
		errorEvent := events.Event{
			Type:        events.EventTypeError,
			Data:        map[string]interface{}{"error": fmt.Sprintf("订阅主题失败: %v", err)},
			Timestamp:   time.Now(),
			RequestUUID: requestUUID,
		}
		sendSSEEvent(writer, flusher, errorEvent)
		return err
	}
	defer eventManager.UnsubscribeFromTopic(clientID, game_topic)

	// 发送心跳保持连接
	ticker := time.NewTicker(60 * time.Second)
	defer ticker.Stop()

	// 等待连接结束
	for {
		select {
		case <-ctx.Done():
			log.Infof("SSE connection closed by client - RequestUUID: %s", requestUUID)
			return nil
		case <-time.After(time.Duration(task.Request.Duration) * time.Second):
			log.Infof("SSE connection timeout - RequestUUID: %s", requestUUID)
			return nil
		case <-task.stopChan:
			log.Infof("SSE connection stopped manually - RequestUUID: %s", requestUUID)
			return nil
		case <-ticker.C:
			// 发送心跳
			heartbeatEvent := events.Event{
				Type: events.EventTypeHeartbeat,
				Data: map[string]interface{}{
					"Timestamp": time.Now().Unix(),
				},
				Timestamp:   time.Now(),
				RequestUUID: requestUUID,
			}
			if err := sendSSEEvent(writer, flusher, heartbeatEvent); err != nil {
				log.Errorf("发送心跳失败: %v", err)
				return err
			}
		}
	}
}

// convertRoomServerEventToSSE 将RoomServer事件转换为SSE事件
func (task *SubscribeGameInfoTask) convertRoomServerEventToSSE(msg *proto.Message, requestUUID string) events.Event {
	// 根据事件类型进行转换
	switch msg.Event.Type {
	case proto.EventType_TYPE_MATCHED:
		return events.Event{
			Type: events.EventTypeDataChange,
			Data: map[string]interface{}{
				"EventType": "matched",
			},
			Timestamp:   time.Now(),
			RequestUUID: requestUUID,
		}
	case proto.EventType_TYPE_PART_CONFIRMED:
		return events.Event{
			Type: events.EventTypeDataChange,
			Data: map[string]interface{}{
				"EventType": "partConfirmed",
			},
			Timestamp:   time.Now(),
			RequestUUID: requestUUID,
		}
	case proto.EventType_TYPE_GAME_CREATED:
		return events.Event{
			Type: events.EventTypeDataChange,
			Data: map[string]interface{}{
				"EventType": "gameCreated",
			},
			Timestamp:   time.Now(),
			RequestUUID: requestUUID,
		}
	case proto.EventType_TYPE_ROUND_READY:
		return events.Event{
			Type: events.EventTypeStatusUpdate,
			Data: map[string]interface{}{
				"EventType": "roundReady",
			},
			Timestamp:   time.Now(),
			RequestUUID: requestUUID,
		}
	case proto.EventType_TYPE_COMMITMENTS_ON_CHAIN:
		return events.Event{
			Type: events.EventTypeStatusUpdate,
			Data: map[string]interface{}{
				"EventType": "commitmentsOnChain",
			},
			Timestamp:   time.Now(),
			RequestUUID: requestUUID,
		}
	case proto.EventType_TYPE_CARDS_ON_CHAIN:
		return events.Event{
			Type: events.EventTypeStatusUpdate,
			Data: map[string]interface{}{
				"EventType": "cardsOnChain",
			},
			Timestamp:   time.Now(),
			RequestUUID: requestUUID,
		}
	case proto.EventType_TYPE_ROUND_COMPLETE:
		return events.Event{
			Type: events.EventTypeDataChange,
			Data: map[string]interface{}{
				"EventType": "roundComplete",
			},
			Timestamp:   time.Now(),
			RequestUUID: requestUUID,
		}
	case proto.EventType_TYPE_GAME_COMPLETE:
		return events.Event{
			Type: events.EventTypeStatusUpdate,
			Data: map[string]interface{}{
				"EventType": "gameComplete",
			},
			Timestamp:   time.Now(),
			RequestUUID: requestUUID,
		}
	case proto.EventType_TYPE_PLAYER_OFFLINE:
		return events.Event{
			Type: events.EventTypeStatusUpdate,
			Data: map[string]interface{}{
				"EventType": "playerOffline",
			},
			Timestamp:   time.Now(),
			RequestUUID: requestUUID,
		}
	case proto.EventType_TYPE_CONTINUE_CANCELED:
		return events.Event{
			Type: events.EventTypeError,
			Data: map[string]interface{}{
				"EventType": "continueCanceled",
			},
			Timestamp:   time.Now(),
			RequestUUID: requestUUID,
		}
	case proto.EventType_TYPE_KNOWN:
		fallthrough
	default:
		// 对于未知事件类型，直接转发原始数据
		jsonData, _ := json.Marshal(msg)
		return events.Event{
			Type: events.EventTypeDataChange,
			Data: map[string]interface{}{
				"EventType": "unknown",
				"RawData":   string(jsonData),
			},
			Timestamp:   time.Now(),
			RequestUUID: requestUUID,
		}
	}
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
