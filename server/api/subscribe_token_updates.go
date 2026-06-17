package api

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/CryptoElementals/common/log"
	"github.com/CryptoElementals/common/pubsub"
	"github.com/CryptoElementals/common/rpc/client"
	"github.com/CryptoElementals/common/rpc/proto"
	"github.com/CryptoElementals/common/server/sse"
	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
	"github.com/mitchellh/mapstructure"
)

func init() {
	Register(SUBSCRIBE_TOKEN_UPDATES_LABEL, NewSubscribeTokenUpdatesTask, COOKIEAUTH)
}

type SubscribeTokenUpdatesRequest struct {
	BaseRequest
	Duration int    `mapstructure:"Duration" validate:"min=1,max=86400"`
	PlayerID string `mapstructure:"PlayerID" validate:"required"`
}

type SubscribeTokenUpdatesResponse struct {
	BaseResponse
	Message string `json:"message"`
}

type SubscribeTokenUpdatesTask struct {
	Request  *SubscribeTokenUpdatesRequest
	Response *SubscribeTokenUpdatesResponse
	mu       sync.Mutex
	stopChan chan struct{}
}

var (
	subscribeTokenBusMu   sync.Mutex
	subscribeTokenBuses   = make(map[string]client.EventBus)
	subscribeTokenBusErrs = make(map[string]error)
)

func getSubscribeTokenEventBus(serverType string) (client.EventBus, error) {
	subscribeTokenBusMu.Lock()
	defer subscribeTokenBusMu.Unlock()
	if bus, ok := subscribeTokenBuses[serverType]; ok {
		return bus, subscribeTokenBusErrs[serverType]
	}
	eventStream := client.EventStreamForType(serverType)
	if eventStream == nil {
		err := fmt.Errorf("event stream is nil for server type %q", serverType)
		subscribeTokenBusErrs[serverType] = err
		return nil, err
	}
	bus := client.NewTokenEventBus(pubsub.NewStreamSubscriber(eventStream))
	subscribeTokenBuses[serverType] = bus
	return bus, subscribeTokenBusErrs[serverType]
}

func NewSubscribeTokenUpdatesRequest(data *map[string]interface{}) (*SubscribeTokenUpdatesRequest, error) {
	req := &SubscribeTokenUpdatesRequest{}
	if err := mapstructure.Decode(*data, &req); err != nil {
		return nil, err
	}
	req.BaseRequest.RequestUUID = (*data)["RequestUUID"].(string)
	if req.Duration == 0 {
		req.Duration = 86400
	}
	return req, nil
}

func NewSubscribeTokenUpdatesResponse(sessionId string) *SubscribeTokenUpdatesResponse {
	return &SubscribeTokenUpdatesResponse{
		BaseResponse: BaseResponse{
			Action:      SUBSCRIBE_TOKEN_UPDATES_LABEL + "Response",
			RequestUUID: sessionId,
		},
	}
}

func NewSubscribeTokenUpdatesTask(data *map[string]interface{}) (Task, error) {
	req, err := NewSubscribeTokenUpdatesRequest(data)
	if err != nil {
		return nil, err
	}
	task := &SubscribeTokenUpdatesTask{
		Request:  req,
		Response: NewSubscribeTokenUpdatesResponse(req.BaseRequest.RequestUUID),
		stopChan: make(chan struct{}, 1),
	}
	validate := validator.New()
	if err := validate.Struct(task.Request); err != nil {
		return nil, err
	}
	return task, nil
}

func (task *SubscribeTokenUpdatesTask) Run(c *gin.Context) (Response, error) {
	playerIDStr := strings.TrimSpace(task.Request.PlayerID)
	if playerIDStr == "" {
		return nil, fmt.Errorf("player id is empty")
	}
	playerID, err := strconv.ParseInt(playerIDStr, 10, 64)
	if err != nil {
		return nil, fmt.Errorf("invalid player id: %v", err)
	}

	serverType := ServerTypeFromGin(c)
	eventBus, err := getSubscribeTokenEventBus(serverType)
	if err != nil {
		log.Errorf("failed to initialize token event bus: %v", err)
		errorEvent := sse.Event{
			Type:        sse.EventTypeError,
			Data:        map[string]interface{}{"error": fmt.Sprintf("failed to initialize token event bus: %v", err)},
			Timestamp:   time.Now(),
			RequestUUID: task.Request.RequestUUID,
		}
		_ = sse.Write(c.Writer, c.Writer.(http.Flusher), errorEvent)
		return nil, err
	}

	self := &proto.PlayerAddress{Id: playerID}
	subscriberID := client.SubscriberID{Address: self, ClientID: task.Request.RequestUUID}
	msgCh, errCh := eventBus.RegisterSubscriber(subscriberID)
	defer eventBus.UnregisterSubscriber(subscriberID)

	connectedEvent := sse.Event{
		Type: sse.EventTypeNotification,
		Data: map[string]interface{}{
			"Status": "connected",
		},
		Timestamp:   time.Now(),
		RequestUUID: task.Request.RequestUUID,
	}
	if err := sse.Write(c.Writer, c.Writer.(http.Flusher), connectedEvent); err != nil {
		return nil, err
	}

	ticker := time.NewTicker(20 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-c.Request.Context().Done():
			log.Infof("token SSE connection closed by client - RequestUUID: %s", task.Request.RequestUUID)
			return task.Response, nil
		case <-time.After(time.Duration(task.Request.Duration) * time.Second):
			log.Infof("token SSE connection timeout - RequestUUID: %s", task.Request.RequestUUID)
			return task.Response, nil
		case <-task.stopChan:
			log.Infof("token SSE connection stopped manually - RequestUUID: %s", task.Request.RequestUUID)
			return task.Response, nil
		case <-ticker.C:
			heartbeatEvent := sse.Event{
				Type:        sse.EventTypeHeartbeat,
				Data:        map[string]interface{}{},
				Timestamp:   time.Now(),
				RequestUUID: task.Request.RequestUUID,
			}
			if err := sse.Write(c.Writer, c.Writer.(http.Flusher), heartbeatEvent); err != nil {
				log.Errorf("failed to send token SSE heartbeat: %v", err)
			}
		case msg, ok := <-msgCh:
			if !ok {
				log.Infof("token event stream closed - RequestUUID: %s", task.Request.RequestUUID)
				return task.Response, nil
			}
			sseEvent := task.convertTokenEventToSSE(msg, task.Request.RequestUUID)
			if err := sse.Write(c.Writer, c.Writer.(http.Flusher), sseEvent); err != nil {
				log.Errorf("failed to send token SSE event: %v", err)
			}
		case err, ok := <-errCh:
			if !ok {
				return task.Response, nil
			}
			if err == nil {
				continue
			}
			log.Errorf("token event bus subscriber error: %v", err)
			errorEvent := sse.Event{
				Type:        sse.EventTypeError,
				Data:        map[string]interface{}{"error": fmt.Sprintf("token event bus subscriber error: %v", err)},
				Timestamp:   time.Now(),
				RequestUUID: task.Request.RequestUUID,
			}
			_ = sse.Write(c.Writer, c.Writer.(http.Flusher), errorEvent)
			return nil, err
		}
	}
}

func (task *SubscribeTokenUpdatesTask) convertTokenEventToSSE(msg *proto.Message, requestUUID string) sse.Event {
	if msg == nil || msg.GetEvent() == nil {
		return sse.Event{
			Type:        sse.EventTypeError,
			Data:        map[string]interface{}{"error": "nil token event"},
			Timestamp:   time.Now(),
			RequestUUID: requestUUID,
		}
	}
	messageID := msg.GetEvent().GetMessageId()
	if msg.GetEvent().GetType() == proto.EventType_TYPE_TOKEN_UPDATED {
		return sse.Event{
			Type: sse.EventTypeDataChange,
			Data: map[string]interface{}{
				"MessageID": messageID,
				"EventType": "tokenUpdated",
				"Message":   msg.GetEvent().GetTokenUpdated(),
			},
			Timestamp:   time.Now(),
			RequestUUID: requestUUID,
		}
	}
	return sse.Event{
		Type: sse.EventTypeError,
		Data: map[string]interface{}{
			"MessageID": messageID,
			"error":     fmt.Sprintf("unexpected event type: %s", msg.GetEvent().GetType().String()),
		},
		Timestamp:   time.Now(),
		RequestUUID: requestUUID,
	}
}
