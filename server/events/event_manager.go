package events

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/CryptoElementals/common/log"
	"github.com/CryptoElementals/common/pubsub"
	"github.com/CryptoElementals/common/rpc/client"
	"github.com/CryptoElementals/common/rpc/proto"
	"github.com/CryptoElementals/common/stream"
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
	ID           string
	Handler      EventHandler
	Topics       map[string]bool // 客户端订阅的主题集合
	Active       bool
	ReceiverSelf *proto.PlayerAddress // 非空时仅转发 Receivers 包含该玩家的房间/大厅事件
	mu           sync.RWMutex
}

// GlobalEventManager 全局事件管理器
type GlobalEventManager struct {
	clients          map[string]*SSEClient         // clientID -> SSEClient
	topicClients     map[string]map[string]bool    // topic -> clientID -> bool
	subscribedTopics map[string]func()        // topic -> stop Redis stream reader
	topicStreams     map[string]chan struct{} // topic -> reconnect signal
	mu               sync.RWMutex
	eventStream      stream.Stream
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
			subscribedTopics: make(map[string]func()),
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

	em.eventStream = client.GetGlobalEventStream()
	if em.eventStream == nil {
		return fmt.Errorf("事件流未初始化（请先 redis.Init 再 InitGlobalClients）")
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
		// 检查连接是否真的断开
		if em.isTopicConnectionBroken(topic) {
			log.Warnf("检测到topic %s 连接断开，触发重连", topic)
			select {
			case em.reconnectChan <- topic:
				log.Debugf("发送重连信号给topic: %s", topic)
			default:
				// 通道已满，跳过这次重连
				log.Warnf("重连通道已满，跳过topic: %s", topic)
			}
		}
	}
}

// isTopicConnectionBroken 检查topic连接是否断开
func (em *GlobalEventManager) isTopicConnectionBroken(topic string) bool {
	em.mu.RLock()
	defer em.mu.RUnlock()

	// 检查是否有活跃的订阅
	if _, exists := em.subscribedTopics[topic]; !exists {
		// 没有订阅记录，说明连接已断开
		return true
	}

	// 有订阅记录，说明连接正常
	return false
}

// reconnectTopic 重连指定主题
func (em *GlobalEventManager) reconnectTopic(topic string) {
	em.mu.Lock()
	defer em.mu.Unlock()

	// 检查topic是否还有客户端
	if clients, exists := em.topicClients[topic]; !exists || len(clients) == 0 {
		return
	}

	log.Infof("开始重连topic: %s", topic)

	// 取消旧的订阅
	if stop, exists := em.subscribedTopics[topic]; exists {
		stop()
		delete(em.subscribedTopics, topic)
	}

	em.eventStream = client.GetGlobalEventStream()
	if em.eventStream == nil {
		log.Errorf("重连时事件流不可用，topic: %s", topic)
		return
	}

	// 重新订阅
	if err := em.subscribeToTopic(topic); err != nil {
		log.Errorf("重连topic失败: %s, 错误: %v", topic, err)
		// 5秒后重试
		go func() {
			time.Sleep(5 * time.Second)
			select {
			case em.reconnectChan <- topic:
			default:
				log.Warnf("重连通道已满，跳过topic: %s的重连请求", topic)
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

// RegisterSSEPlayerClient 注册带玩家身份的 SSE 客户端，仅转发 Receivers 包含该玩家的事件。
func (em *GlobalEventManager) RegisterSSEPlayerClient(clientID string, self *proto.PlayerAddress, handler EventHandler) *SSEClient {
	em.mu.Lock()
	defer em.mu.Unlock()

	sseClient := &SSEClient{
		ID:           clientID,
		Handler:      handler,
		Topics:       make(map[string]bool),
		Active:       true,
		ReceiverSelf: self,
	}

	em.clients[clientID] = sseClient
	log.Infof("注册SSE客户端(玩家): %s", clientID)
	return sseClient
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

// subscribeToTopic starts a Redis stream reader for topic ([pubsub.TopicRoom] or [pubsub.TopicLobby]).
func (em *GlobalEventManager) subscribeToTopic(topic string) error {
	if em.eventStream == nil {
		return fmt.Errorf("事件流未初始化")
	}
	ctx, cancel := context.WithCancel(em.ctx)
	sub := pubsub.NewStreamSubscriber(em.eventStream)
	msgCh, stopReader, err := sub.Subscribe(ctx, topic, pubsub.SubscribeOptions{})
	if err != nil {
		cancel()
		return fmt.Errorf("订阅主题失败: %v", err)
	}
	var stopOnce sync.Once
	stopAll := func() {
		stopOnce.Do(func() {
			stopReader()
			cancel()
		})
	}
	em.subscribedTopics[topic] = stopAll

	go func() {
		defer func() {
			if r := recover(); r != nil {
				log.Errorf("处理主题 %s 事件时发生panic: %v", topic, r)
			}
			stopAll()
			em.mu.Lock()
			delete(em.subscribedTopics, topic)
			em.mu.Unlock()
			em.mu.RLock()
			hasClients := len(em.topicClients[topic]) > 0
			em.mu.RUnlock()
			if hasClients {
				log.Warnf("主题 %s 流读取结束", topic)
			}
		}()
		for {
			select {
			case <-ctx.Done():
				log.Infof("主题 %s 事件处理协程退出", topic)
				return
			case msg, ok := <-msgCh:
				if !ok {
					return
				}
				em.forwardEventToClients(topic, msg)
			}
		}
	}()

	log.Infof("成功订阅流主题: %s", topic)
	return nil
}

// unsubscribeFromTopic stops the Redis reader for topic.
func (em *GlobalEventManager) unsubscribeFromTopic(topic string) {
	if stop, exists := em.subscribedTopics[topic]; exists {
		stop()
		delete(em.subscribedTopics, topic)
		log.Infof("取消订阅流主题: %s", topic)
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

	if client.ReceiverSelf != nil && !pubsub.EventTargetsReceiver(msg.GetEvent(), client.ReceiverSelf) {
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

	for topic, stop := range em.subscribedTopics {
		stop()
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
