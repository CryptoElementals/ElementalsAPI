package events

import (
	"context"
	"fmt"
	"io"
	"sync"
	"time"

	"github.com/CryptoElementals/common/log"
	"github.com/CryptoElementals/common/rpc/client"
	"github.com/CryptoElementals/common/rpc/proto"
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

// EventHandler 定义事件处理函数类型
type EventHandler func(*proto.Message)

// SSEClient 表示一个SSE客户端连接
type SSEClient struct {
	ID      string
	Handler EventHandler
	Topics  map[string]bool // 客户端订阅的主题集合
	Active  bool
	mu      sync.RWMutex
}

// GlobalEventManager 全局事件管理器
type GlobalEventManager struct {
	clients          map[string]*SSEClient         // clientID -> SSEClient
	topicClients     map[string]map[string]bool    // topic -> clientID -> bool
	subscribedTopics map[string]context.CancelFunc // topic -> cancelFunc
	topicStreams     map[string]chan struct{}      // topic -> reconnect signal
	mu               sync.RWMutex
	pubsubClient     proto.PubSubServiceClient
	ctx              context.Context
	cancel           context.CancelFunc
	reconnectChan    chan string // 重连信号通道
}

var (
	globalManager *GlobalEventManager
	managerOnce   sync.Once
)

// GetGlobalEventManager 获取全局事件管理器实例
func GetGlobalEventManager() *GlobalEventManager {
	managerOnce.Do(func() {
		globalManager = &GlobalEventManager{
			clients:          make(map[string]*SSEClient),
			topicClients:     make(map[string]map[string]bool),
			subscribedTopics: make(map[string]context.CancelFunc),
			topicStreams:     make(map[string]chan struct{}),
			reconnectChan:    make(chan string, 100), // 缓冲通道
		}
		globalManager.ctx, globalManager.cancel = context.WithCancel(context.Background())
	})
	return globalManager
}

// Initialize 初始化事件管理器
func (em *GlobalEventManager) Initialize() error {
	em.mu.Lock()
	defer em.mu.Unlock()

	// 获取全局PubSub客户端
	em.pubsubClient = client.GetGlobalPubSubClient()
	if em.pubsubClient == nil {
		return fmt.Errorf("gRPC PubSub客户端未初始化")
	}

	// 启动连接监控协程
	go em.startConnectionMonitor()

	log.Infof("全局事件管理器初始化成功")
	return nil
}

// startConnectionMonitor 启动连接监控
func (em *GlobalEventManager) startConnectionMonitor() {
	ticker := time.NewTicker(10 * time.Second) // 每10秒检查一次
	defer ticker.Stop()

	for {
		select {
		case <-em.ctx.Done():
			return
		case <-ticker.C:
			// 检查连接状态并重新订阅失效的topic
			em.checkAndReconnectTopics()
		case topic := <-em.reconnectChan:
			// 立即重连指定topic
			em.reconnectTopic(topic)
		}
	}
}

// checkAndReconnectTopics 检查并重连主题
func (em *GlobalEventManager) checkAndReconnectTopics() {
	em.mu.RLock()
	topics := make([]string, 0, len(em.subscribedTopics))
	for topic := range em.subscribedTopics {
		topics = append(topics, topic)
	}
	em.mu.RUnlock()

	// 检查每个topic的连接状态
	for _, topic := range topics {
		// 这里可以添加连接状态检查逻辑
		// 如果发现连接异常，发送重连信号
		select {
		case em.reconnectChan <- topic:
		default:
			// 通道已满，跳过这次重连
		}
	}
}

// reconnectTopic 重连指定主题
func (em *GlobalEventManager) reconnectTopic(topic string) {
	em.mu.Lock()
	defer em.mu.Unlock()

	// 检查topic是否还有客户端
	if clients, exists := em.topicClients[topic]; !exists || len(clients) == 0 {
		return
	}

	// 取消旧的订阅
	if cancel, exists := em.subscribedTopics[topic]; exists {
		cancel()
		delete(em.subscribedTopics, topic)
	}

	// 重新获取客户端并订阅
	em.pubsubClient = client.GetGlobalPubSubClient()
	if em.pubsubClient == nil {
		log.Errorf("重连时获取PubSub客户端失败，topic: %s", topic)
		return
	}

	// 重新订阅
	if err := em.subscribeToTopic(topic); err != nil {
		log.Errorf("重连topic失败: %s, 错误: %v", topic, err)
		// 1秒后重试
		go func() {
			time.Sleep(1 * time.Second)
			select {
			case em.reconnectChan <- topic:
			default:
			}
		}()
	} else {
		log.Infof("成功重连topic: %s", topic)
	}
}

// RegisterSSEClient 注册SSE客户端
func (em *GlobalEventManager) RegisterSSEClient(clientID string, handler EventHandler) *SSEClient {
	em.mu.Lock()
	defer em.mu.Unlock()

	client := &SSEClient{
		ID:      clientID,
		Handler: handler,
		Topics:  make(map[string]bool),
		Active:  true,
	}

	em.clients[clientID] = client
	log.Infof("注册SSE客户端: %s", clientID)
	return client
}

// UnregisterSSEClient 取消注册SSE客户端
func (em *GlobalEventManager) UnregisterSSEClient(clientID string) {
	em.mu.Lock()
	defer em.mu.Unlock()

	client, exists := em.clients[clientID]
	if !exists {
		return
	}

	// 标记客户端为非活跃
	client.Active = false

	// 从所有主题中移除该客户端
	for topic := range client.Topics {
		if topicClients, exists := em.topicClients[topic]; exists {
			delete(topicClients, clientID)

			// 如果该主题没有客户端了，取消订阅
			if len(topicClients) == 0 {
				em.unsubscribeFromTopic(topic)
				delete(em.topicClients, topic)
			}
		}
	}

	delete(em.clients, clientID)
	log.Infof("取消注册SSE客户端: %s", clientID)
}

// SubscribeToTopic 订阅主题
func (em *GlobalEventManager) SubscribeToTopic(clientID, topic string) error {
	em.mu.Lock()
	defer em.mu.Unlock()

	client, exists := em.clients[clientID]
	if !exists {
		return fmt.Errorf("客户端 %s 不存在", clientID)
	}

	// 添加客户端到主题
	if em.topicClients[topic] == nil {
		em.topicClients[topic] = make(map[string]bool)
	}
	em.topicClients[topic][clientID] = true
	client.Topics[topic] = true

	// 如果这是第一个订阅该主题的客户端，创建到RoomServer的订阅
	if len(em.topicClients[topic]) == 1 {
		if err := em.subscribeToTopic(topic); err != nil {
			// 订阅失败，清理状态
			delete(em.topicClients[topic], clientID)
			delete(client.Topics, topic)
			if len(em.topicClients[topic]) == 0 {
				delete(em.topicClients, topic)
			}
			return fmt.Errorf("订阅主题 %s 失败: %v", topic, err)
		}
	}

	log.Infof("客户端 %s 订阅主题 %s", clientID, topic)
	return nil
}

// UnsubscribeFromTopic 取消订阅主题
func (em *GlobalEventManager) UnsubscribeFromTopic(clientID, topic string) {
	em.mu.Lock()
	defer em.mu.Unlock()

	client, exists := em.clients[clientID]
	if !exists {
		return
	}

	// 从主题中移除客户端
	if topicClients, exists := em.topicClients[topic]; exists {
		delete(topicClients, clientID)
		delete(client.Topics, topic)

		// 如果该主题没有客户端了，取消订阅
		if len(topicClients) == 0 {
			em.unsubscribeFromTopic(topic)
			delete(em.topicClients, topic)
		}
	}

	log.Infof("客户端 %s 取消订阅主题 %s", clientID, topic)
}

// subscribeToTopic 订阅RoomServer主题
func (em *GlobalEventManager) subscribeToTopic(topic string) error {
	req := &proto.SubscribeRequest{
		Topic:        topic,
		SubscriberId: fmt.Sprintf("apiserver_global_%s", topic),
	}

	stream, err := em.pubsubClient.Subscribe(em.ctx, req)
	if err != nil {
		return fmt.Errorf("订阅主题失败: %v", err)
	}

	// 创建取消函数
	ctx, cancel := context.WithCancel(em.ctx)
	em.subscribedTopics[topic] = cancel

	// 启动事件接收协程
	go em.handleTopicEvents(ctx, topic, stream)

	log.Infof("成功订阅RoomServer主题: %s", topic)
	return nil
}

// unsubscribeFromTopic 取消订阅RoomServer主题
func (em *GlobalEventManager) unsubscribeFromTopic(topic string) {
	if cancel, exists := em.subscribedTopics[topic]; exists {
		cancel()
		delete(em.subscribedTopics, topic)
		log.Infof("取消订阅RoomServer主题: %s", topic)
	}
}

// handleTopicEvents 处理主题事件
func (em *GlobalEventManager) handleTopicEvents(ctx context.Context, topic string, stream proto.PubSubService_SubscribeClient) {
	defer func() {
		if r := recover(); r != nil {
			log.Errorf("处理主题 %s 事件时发生panic: %v", topic, r)
		}
		// 清理订阅状态
		em.mu.Lock()
		if cancel, exists := em.subscribedTopics[topic]; exists {
			cancel()
			delete(em.subscribedTopics, topic)
		}
		em.mu.Unlock()

		// 如果还有客户端订阅这个topic，触发重连
		em.mu.RLock()
		hasClients := len(em.topicClients[topic]) > 0
		em.mu.RUnlock()

		if hasClients {
			log.Warnf("主题 %s 连接断开，尝试重连", topic)
			select {
			case em.reconnectChan <- topic:
			default:
				log.Warnf("重连通道已满，跳过主题 %s 的重连请求", topic)
			}
		}
	}()

	for {
		select {
		case <-ctx.Done():
			log.Infof("主题 %s 事件处理协程退出", topic)
			return
		default:
			msg, err := stream.Recv()
			if err == io.EOF {
				log.Warnf("主题 %s 接收到EOF，连接可能已关闭", topic)
				return
			}
			if err != nil {
				log.Errorf("接收主题 %s 消息失败: %v", topic, err)
				return
			}

			// 转发事件给所有订阅该主题的客户端
			em.forwardEventToClients(topic, msg)
		}
	}
}

// forwardEventToClients 转发事件给客户端
func (em *GlobalEventManager) forwardEventToClients(topic string, msg *proto.Message) {
	em.mu.RLock()
	topicClients := em.topicClients[topic]
	em.mu.RUnlock()

	if topicClients == nil {
		return
	}

	// 并发转发给所有客户端
	var wg sync.WaitGroup
	for clientID := range topicClients {
		wg.Add(1)
		go func(cID string) {
			defer wg.Done()
			em.forwardToClient(cID, msg)
		}(clientID)
	}
	wg.Wait()
}

// forwardToClient 转发事件给特定客户端
func (em *GlobalEventManager) forwardToClient(clientID string, msg *proto.Message) {
	defer func() {
		if r := recover(); r != nil {
			log.Errorf("转发事件给客户端 %s 时发生panic: %v", clientID, r)
		}
	}()

	em.mu.RLock()
	client, exists := em.clients[clientID]
	em.mu.RUnlock()

	if !exists || !client.Active {
		return
	}

	// 调用客户端处理函数
	select {
	case <-time.After(5 * time.Second): // 超时保护
		log.Warnf("转发事件给客户端 %s 超时", clientID)
	default:
		client.Handler(msg)
	}
}

// Shutdown 关闭事件管理器
func (em *GlobalEventManager) Shutdown() {
	em.mu.Lock()
	defer em.mu.Unlock()

	// 取消所有订阅
	for topic, cancel := range em.subscribedTopics {
		cancel()
		log.Infof("取消订阅主题: %s", topic)
	}

	// 清理所有客户端
	for clientID := range em.clients {
		log.Infof("清理客户端: %s", clientID)
	}

	em.cancel()

	// 关闭重连通道
	close(em.reconnectChan)

	log.Infof("全局事件管理器已关闭")
}

// GetStats 获取统计信息
func (em *GlobalEventManager) GetStats() map[string]interface{} {
	em.mu.RLock()
	defer em.mu.RUnlock()

	return map[string]interface{}{
		"clients_count":           len(em.clients),
		"subscribed_topics_count": len(em.subscribedTopics),
		"topic_clients":           em.topicClients,
	}
}
