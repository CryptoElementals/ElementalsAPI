package server

import (
	"context"

	"sync"
	"time"

	"github.com/google/uuid"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/CryptoElementals/common/log"
	"github.com/CryptoElementals/common/room_server/worker/types"
	pb "github.com/CryptoElementals/common/rpc/proto"
)

type PlayerManager interface {
	AddPlayer(address types.PlayerAddress) error
	RemovePlayer(address types.PlayerAddress)
}

type PubSub struct {
	pb.UnimplementedPubSubServiceServer
	mu            sync.RWMutex
	topics        map[string]*Topic
	subscribers   map[string]map[string]*Subscriber
	playerManager PlayerManager
}

type Topic struct {
	name           string
	subscribers    map[string]*Subscriber
	messageHistory []*pb.Message
	mu             sync.RWMutex
}

type Subscriber struct {
	id     string
	topic  string
	stream pb.PubSubService_SubscribeServer
	ctx    context.Context
	cancel context.CancelFunc
	mu     sync.Mutex
}

func NewPubSub() *PubSub {
	s := &PubSub{
		topics:      make(map[string]*Topic),
		subscribers: make(map[string]map[string]*Subscriber),
	}
	return s
}

func (s *PubSub) SetPlayerManager(playerManager PlayerManager) {
	s.playerManager = playerManager
}

func (s *PubSub) Publish(ctx context.Context, req *pb.PublishRequest) (*pb.PublishResponse, error) {
	if req.Topic == "" {
		return nil, status.Error(codes.InvalidArgument, "topic is required")
	}
	if req.Event == nil {
		return nil, status.Error(codes.InvalidArgument, "event is required")
	}

	s.mu.Lock()
	topic, exists := s.topics[req.Topic]
	if !exists {
		topic = &Topic{
			name:        req.Topic,
			subscribers: make(map[string]*Subscriber),
		}
		s.topics[req.Topic] = topic
	}
	s.mu.Unlock()

	message := &pb.Message{
		MessageId:   uuid.New().String(),
		Topic:       req.Topic,
		Event:       req.Event,
		Metadata:    req.Metadata,
		Timestamp:   time.Now().Unix(),
		PublisherId: "server", // 可以从ctx中获取实际的发布者ID
	}

	// 添加到消息历史
	topic.mu.Lock()
	topic.messageHistory = append(topic.messageHistory, message)
	// 保持最近1000条消息
	if len(topic.messageHistory) > 1000 {
		topic.messageHistory = topic.messageHistory[1:]
	}
	topic.mu.Unlock()

	// 广播给所有订阅者
	subscriberCount := 0
	topic.mu.RLock()
	for _, subscriber := range topic.subscribers {
		subscriber.mu.Lock()
		if subscriber.stream != nil {
			err := subscriber.stream.Send(message)
			if err != nil {
				log.Errorf("Failed to send message to subscriber %s: %v", subscriber.id, err)
				subscriber.cancel()
			} else {
				subscriberCount++
			}
		}
		subscriber.mu.Unlock()
	}
	topic.mu.RUnlock()

	return &pb.PublishResponse{
		MessageId:       message.MessageId,
		SubscriberCount: int32(subscriberCount),
		Success:         true,
	}, nil
}

func (s *PubSub) Subscribe(req *pb.SubscribeRequest, stream pb.PubSubService_SubscribeServer) error {
	if req.Topic == "" {
		return status.Error(codes.InvalidArgument, "topic is required")
	}
	if req.SubscriberId == "" {
		req.SubscriberId = uuid.New().String()
	}
	addr := types.PlayerAddress{}
	err := addr.Parse(req.Topic)
	if err != nil {
		return status.Error(codes.InvalidArgument, "topic is invalid, topic should be in 'walletAddress_temporaryAddress' format")
	}
	err = s.playerManager.AddPlayer(addr)
	if err != nil {
		return status.Error(codes.InvalidArgument, "failed to add player: "+err.Error())
	}

	s.mu.Lock()
	topic, exists := s.topics[req.Topic]
	if !exists {
		topic = &Topic{
			name:        req.Topic,
			subscribers: make(map[string]*Subscriber),
		}
		s.topics[req.Topic] = topic
	}
	s.mu.Unlock()

	ctx, cancel := context.WithCancel(stream.Context())
	subscriber := &Subscriber{
		id:     req.SubscriberId,
		topic:  req.Topic,
		stream: stream,
		ctx:    ctx,
		cancel: cancel,
	}

	// 注册订阅者
	topic.mu.Lock()
	topic.subscribers[req.SubscriberId] = subscriber
	topic.mu.Unlock()

	s.mu.Lock()
	if s.subscribers[req.Topic] == nil {
		s.subscribers[req.Topic] = make(map[string]*Subscriber)
	}
	s.subscribers[req.Topic][req.SubscriberId] = subscriber
	s.mu.Unlock()

	log.Infof("Subscriber %s subscribed to topic %s", req.SubscriberId, req.Topic)

	// 等待连接关闭
	<-ctx.Done()

	// 清理订阅者
	s.Unsubscribe(context.Background(), &pb.UnsubscribeRequest{
		Topic:        req.Topic,
		SubscriberId: req.SubscriberId,
	})

	log.Infof("Subscriber %s unsubscribed from topic %s", req.SubscriberId, req.Topic)
	return nil
}

func (s *PubSub) Unsubscribe(ctx context.Context, req *pb.UnsubscribeRequest) (*pb.UnsubscribeResponse, error) {
	if req.Topic == "" || req.SubscriberId == "" {
		return nil, status.Error(codes.InvalidArgument, "topic and subscriber_id are required")
	}

	addr := types.PlayerAddress{}
	err := addr.Parse(req.Topic)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "topic is invalid, topic should be in 'walletAddress_temporaryAddress' format")
	}
	s.playerManager.RemovePlayer(addr)
	s.mu.Lock()
	defer s.mu.Unlock()

	// 从主题中移除订阅者
	if topic, exists := s.topics[req.Topic]; exists {
		topic.mu.Lock()
		if subscriber, exists := topic.subscribers[req.SubscriberId]; exists {
			subscriber.cancel()
			delete(topic.subscribers, req.SubscriberId)
		}
		topic.mu.Unlock()
	}

	// 从全局订阅者映射中移除
	if subscribers, exists := s.subscribers[req.Topic]; exists {
		if subscriber, exists := subscribers[req.SubscriberId]; exists {
			subscriber.cancel()
			delete(subscribers, req.SubscriberId)
		}
	}

	return &pb.UnsubscribeResponse{
		Success: true,
	}, nil
}

func (s *PubSub) ListTopics(ctx context.Context, req *pb.ListTopicsRequest) (*pb.ListTopicsResponse, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var topics []*pb.TopicInfo
	for _, topic := range s.topics {
		topic.mu.RLock()
		topicInfo := &pb.TopicInfo{
			Name:            topic.name,
			SubscriberCount: int32(len(topic.subscribers)),
			MessageCount:    int64(len(topic.messageHistory)),
		}
		if len(topic.messageHistory) > 0 {
			topicInfo.LastMessageTime = topic.messageHistory[len(topic.messageHistory)-1].Timestamp
		}
		topic.mu.RUnlock()
		topics = append(topics, topicInfo)
	}

	return &pb.ListTopicsResponse{
		Topics: topics,
	}, nil
}

func (s *PubSub) GetSubscriberCount(ctx context.Context, req *pb.GetSubscriberCountRequest) (*pb.GetSubscriberCountResponse, error) {
	if req.Topic == "" {
		return nil, status.Error(codes.InvalidArgument, "topic is required")
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	if topic, exists := s.topics[req.Topic]; exists {
		topic.mu.RLock()
		count := len(topic.subscribers)
		topic.mu.RUnlock()
		return &pb.GetSubscriberCountResponse{
			Topic:           req.Topic,
			SubscriberCount: int32(count),
			Success:         true,
		}, nil
	}

	return &pb.GetSubscriberCountResponse{
		Topic:   req.Topic,
		Success: false,
		Error:   "topic not found",
	}, nil
}
