package events

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/CryptoElementals/common/log"
)

// EventType 定义事件类型
type EventType string

const (
	EventTypeNotification EventType = "notification"
	EventTypeDataChange   EventType = "data_change"
	EventTypeStatusUpdate EventType = "status_update"
	EventTypeError        EventType = "error"
	EventTypeHeartbeat    EventType = "heartbeat"
)

// Event 事件结构
type Event struct {
	Type        EventType              `json:"type"`
	Data        interface{}            `json:"data"`
	Timestamp   time.Time              `json:"timestamp"`
	RequestUUID string                 `json:"RequestUUID,omitempty"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
}

// SSEConnection SSE 连接
type SSEConnection struct {
	ID          string
	Writer      http.ResponseWriter
	Flusher     http.Flusher
	RequestUUID string
	Context     context.Context
	Cancel      context.CancelFunc
	mu          sync.Mutex
	closed      bool
}

// EventManager 事件管理器
type EventManager struct {
	connections map[string]*SSEConnection
	eventChan   chan Event
	mu          sync.RWMutex
}

var (
	globalEventManager *EventManager
	once               sync.Once
)

// GetEventManager 获取全局事件管理器
func GetEventManager() *EventManager {
	once.Do(func() {
		globalEventManager = &EventManager{
			connections: make(map[string]*SSEConnection),
			eventChan:   make(chan Event, 1000), // 缓冲通道
		}
		go globalEventManager.startEventLoop()
	})
	return globalEventManager
}

// startEventLoop 启动事件循环
func (em *EventManager) startEventLoop() {
	for event := range em.eventChan {
		em.broadcastEvent(event)
	}
}

// broadcastEvent 广播事件到所有连接
func (em *EventManager) broadcastEvent(event Event) {
	em.mu.RLock()
	defer em.mu.RUnlock()

	for _, conn := range em.connections {
		// 检查连接是否已关闭
		if conn.isClosed() {
			continue
		}

		// 检查上下文是否已取消
		select {
		case <-conn.Context.Done():
			em.removeConnection(conn.ID)
			continue
		default:
		}

		// 发送事件
		if err := conn.sendEvent(event); err != nil {
			log.Errorf("Failed to send event to connection %s: %v", conn.ID, err)
			em.removeConnection(conn.ID)
		}
	}
}

// AddConnection 添加 SSE 连接
func (em *EventManager) AddConnection(conn *SSEConnection) {
	em.mu.Lock()
	defer em.mu.Unlock()
	em.connections[conn.ID] = conn
	log.Infof("SSE connection added: %s, total connections: %d", conn.ID, len(em.connections))
}

// RemoveConnection 移除 SSE 连接
func (em *EventManager) RemoveConnection(connID string) {
	em.mu.Lock()
	defer em.mu.Unlock()
	em.removeConnection(connID)
}

// removeConnection 内部移除连接方法
func (em *EventManager) removeConnection(connID string) {
	if conn, exists := em.connections[connID]; exists {
		conn.close()
		delete(em.connections, connID)
		log.Infof("SSE connection removed: %s, total connections: %d", connID, len(em.connections))
	}
}

// PublishEvent 发布事件
func (em *EventManager) PublishEvent(eventType EventType, data interface{}, metadata map[string]interface{}) {
	event := Event{
		Type:      eventType,
		Data:      data,
		Timestamp: time.Now(),
		Metadata:  metadata,
	}

	select {
	case em.eventChan <- event:
		log.Debugf("Event published: %s", eventType)
	default:
		log.Warnf("Event channel full, dropping event: %s", eventType)
	}
}

// PublishEventToUser 向特定用户发布事件
func (em *EventManager) PublishEventToUser(userID string, eventType EventType, data interface{}, metadata map[string]interface{}) {
	em.mu.RLock()
	defer em.mu.RUnlock()

	event := Event{
		Type:      eventType,
		Data:      data,
		Timestamp: time.Now(),
		Metadata:  metadata,
	}

	for _, conn := range em.connections {
		if conn.isClosed() {
			continue
		}

		// 检查是否是目标用户的连接
		if userIDFromConn := getUserIDFromConnection(conn); userIDFromConn == userID {
			if err := conn.sendEvent(event); err != nil {
				log.Errorf("Failed to send user event to connection %s: %v", conn.ID, err)
			}
		}
	}
}

// GetConnectionCount 获取连接数量
func (em *EventManager) GetConnectionCount() int {
	em.mu.RLock()
	defer em.mu.RUnlock()
	return len(em.connections)
}

// NewSSEConnection 创建新的 SSE 连接
func NewSSEConnection(id string, writer http.ResponseWriter, flusher http.Flusher, requestUUID string) *SSEConnection {
	ctx, cancel := context.WithCancel(context.Background())
	return &SSEConnection{
		ID:          id,
		Writer:      writer,
		Flusher:     flusher,
		RequestUUID: requestUUID,
		Context:     ctx,
		Cancel:      cancel,
	}
}

// sendEvent 发送事件到客户端
func (conn *SSEConnection) sendEvent(event Event) error {
	conn.mu.Lock()
	defer conn.mu.Unlock()

	if conn.closed {
		return fmt.Errorf("connection is closed")
	}

	// 添加 RequestUUID 到事件
	if conn.RequestUUID != "" {
		event.RequestUUID = conn.RequestUUID
	}

	// 序列化事件
	jsonData, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("failed to marshal event: %v", err)
	}

	// 发送 SSE 格式数据
	eventStr := fmt.Sprintf("data: %s\n\n", string(jsonData))
	_, err = conn.Writer.Write([]byte(eventStr))
	if err != nil {
		return fmt.Errorf("failed to write event: %v", err)
	}

	conn.Flusher.Flush()
	return nil
}

// close 关闭连接
func (conn *SSEConnection) close() {
	conn.mu.Lock()
	defer conn.mu.Unlock()

	if !conn.closed {
		conn.closed = true
		conn.Cancel()
	}
}

// isClosed 检查连接是否已关闭
func (conn *SSEConnection) isClosed() bool {
	conn.mu.Lock()
	defer conn.mu.Unlock()
	return conn.closed
}

// getUserIDFromConnection 从连接中获取用户ID（需要根据实际认证机制实现）
func getUserIDFromConnection(conn *SSEConnection) string {
	// 这里需要根据你的认证机制来实现
	// 例如从 session、token 或请求头中获取用户ID
	if conn.RequestUUID != "" {
		// 临时实现，实际应该从认证信息中获取
		return conn.RequestUUID
	}
	return ""
}

// 便捷的事件发布函数
func PublishNotification(message string, metadata map[string]interface{}) {
	GetEventManager().PublishEvent(EventTypeNotification, map[string]interface{}{
		"message": message,
	}, metadata)
}

func PublishDataChange(dataType string, oldValue, newValue interface{}, metadata map[string]interface{}) {
	GetEventManager().PublishEvent(EventTypeDataChange, map[string]interface{}{
		"dataType": dataType,
		"oldValue": oldValue,
		"newValue": newValue,
	}, metadata)
}

func PublishStatusUpdate(status string, metadata map[string]interface{}) {
	GetEventManager().PublishEvent(EventTypeStatusUpdate, map[string]interface{}{
		"status": status,
	}, metadata)
}

func PublishError(error string, metadata map[string]interface{}) {
	GetEventManager().PublishEvent(EventTypeError, map[string]interface{}{
		"error": error,
	}, metadata)
}

// PublishEvent 通用事件发布函数
func PublishEvent(eventType EventType, data interface{}, metadata map[string]interface{}) {
	GetEventManager().PublishEvent(eventType, data, metadata)
}
