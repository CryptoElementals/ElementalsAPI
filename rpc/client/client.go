package client

import (
	"context"
	"fmt"
	"io"
	"sync"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/connectivity"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/keepalive"

	"github.com/CryptoElementals/common/log"
	pb "github.com/CryptoElementals/common/rpc/proto"
)

// 全局gRPC客户端变量
var (
	globalRpcClient    pb.RpcServiceClient
	globalPubSubClient pb.PubSubServiceClient
	globalConn         *grpc.ClientConn
	globalMutex        sync.RWMutex
	initialized        bool
)

// InitGlobalClients 初始化全局gRPC客户端
func InitGlobalClients(serverAddress string) error {
	globalMutex.Lock()
	defer globalMutex.Unlock()

	if initialized {
		return nil // 已经初始化过了
	}

	// 设置连接选项，包括keepalive参数
	opts := []grpc.DialOption{
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithKeepaliveParams(keepalive.ClientParameters{
			Time:                30 * time.Second, // 每30秒发送keepalive ping
			Timeout:             5 * time.Second,  // 5秒超时
			PermitWithoutStream: true,             // 允许在没有活动流时发送keepalive ping
		}),
		grpc.WithDefaultCallOptions(
			grpc.MaxCallRecvMsgSize(4*1024*1024), // 4MB
			grpc.MaxCallSendMsgSize(4*1024*1024), // 4MB
		),
	}

	conn, err := grpc.NewClient(serverAddress, opts...)
	if err != nil {
		return fmt.Errorf("failed to connect to %s: %v", serverAddress, err)
	}

	globalConn = conn
	globalRpcClient = pb.NewRpcServiceClient(conn)
	globalPubSubClient = pb.NewPubSubServiceClient(conn)
	initialized = true

	// 启动健康检查协程
	go startHealthCheck(serverAddress)

	log.Infof("全局gRPC客户端初始化成功，连接到: %s", serverAddress)
	return nil
}

// startHealthCheck 启动健康检查
func startHealthCheck(serverAddress string) {
	ticker := time.NewTicker(30 * time.Second) // 每30秒检查一次
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			globalMutex.RLock()
			conn := globalConn
			globalMutex.RUnlock()

			if conn == nil {
				continue
			}

			// 检查连接状态
			state := conn.GetState()
			if state == connectivity.TransientFailure || state == connectivity.Shutdown {
				log.Warnf("gRPC连接状态异常: %v，尝试重新连接", state)

				// 重新连接
				if err := reconnectGlobalClients(serverAddress); err != nil {
					log.Errorf("重新连接失败: %v", err)
				} else {
					log.Infof("gRPC连接重新建立成功")
				}
			}
		}
	}
}

// reconnectGlobalClients 重新连接
func reconnectGlobalClients(serverAddress string) error {
	globalMutex.Lock()
	defer globalMutex.Unlock()

	// 关闭旧连接
	if globalConn != nil {
		globalConn.Close()
	}

	// 建立新连接
	opts := []grpc.DialOption{
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithKeepaliveParams(keepalive.ClientParameters{
			Time:                30 * time.Second,
			Timeout:             5 * time.Second,
			PermitWithoutStream: true,
		}),
	}

	conn, err := grpc.NewClient(serverAddress, opts...)
	if err != nil {
		return fmt.Errorf("failed to reconnect to %s: %v", serverAddress, err)
	}

	globalConn = conn
	globalRpcClient = pb.NewRpcServiceClient(conn)
	globalPubSubClient = pb.NewPubSubServiceClient(conn)

	return nil
}

// GetGlobalRpcClient 获取全局RPC客户端
func GetGlobalRpcClient() pb.RpcServiceClient {
	globalMutex.RLock()
	defer globalMutex.RUnlock()
	return globalRpcClient
}

// GetGlobalPubSubClient 获取全局PubSub客户端
func GetGlobalPubSubClient() pb.PubSubServiceClient {
	globalMutex.RLock()
	defer globalMutex.RUnlock()
	return globalPubSubClient
}

// CloseGlobalClients 关闭全局客户端连接
func CloseGlobalClients() error {
	globalMutex.Lock()
	defer globalMutex.Unlock()

	if globalConn != nil {
		err := globalConn.Close()
		globalConn = nil
		globalRpcClient = nil
		globalPubSubClient = nil
		initialized = false
		return err
	}
	return nil
}

type PubSubClient struct {
	client pb.PubSubServiceClient
	conn   *grpc.ClientConn
	mu     sync.RWMutex
	subs   map[string]context.CancelFunc
}

func NewPubSubClient(serverAddr string) (*PubSubClient, error) {
	conn, err := grpc.Dial(serverAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, fmt.Errorf("failed to connect: %v", err)
	}

	client := pb.NewPubSubServiceClient(conn)
	return &PubSubClient{
		client: client,
		conn:   conn,
		subs:   make(map[string]context.CancelFunc),
	}, nil
}

func (c *PubSubClient) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	// 取消所有订阅
	for _, cancel := range c.subs {
		cancel()
	}

	return c.conn.Close()
}

func (c *PubSubClient) Publish(topic string, event *pb.Event, metadata map[string]string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	req := &pb.PublishRequest{
		Topic:    topic,
		Event:    event,
		Metadata: metadata,
	}

	resp, err := c.client.Publish(ctx, req)
	if err != nil {
		return fmt.Errorf("failed to publish: %v", err)
	}

	fmt.Printf("Published message %s to topic %s (subscribers: %d)\n",
		resp.MessageId, topic, resp.SubscriberCount)
	return nil
}

func (c *PubSubClient) Subscribe(topic, subscriberID string, evtChan chan *pb.Event, errChan chan error) error {
	ctx, cancel := context.WithCancel(context.Background())

	c.mu.Lock()
	c.subs[topic] = cancel
	c.mu.Unlock()

	req := &pb.SubscribeRequest{
		Topic:        topic,
		SubscriberId: subscriberID,
	}

	stream, err := c.client.Subscribe(ctx, req)
	if err != nil {
		cancel()
		return fmt.Errorf("failed to subscribe: %v", err)
	}

	log.Infof("Subscribed to topic %s with ID %s\n", topic, subscriberID)
	go func() {
		defer close(evtChan)
		defer close(errChan)
		for {
			msg, err := stream.Recv()
			if err == io.EOF || ctx.Err() != nil {
				return
			}
			if err != nil {
				errChan <- err
				return
			}
			select {
			case <-ctx.Done():
				return
			case evtChan <- msg.Event:
			}
		}
	}()
	return nil
}

func (c *PubSubClient) Unsubscribe(topic, subscriberID string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	req := &pb.UnsubscribeRequest{
		Topic:        topic,
		SubscriberId: subscriberID,
	}

	resp, err := c.client.Unsubscribe(ctx, req)
	if err != nil {
		return fmt.Errorf("failed to unsubscribe: %v", err)
	}

	if resp.Success {
		fmt.Printf("Unsubscribed from topic %s\n", topic)
	} else {
		fmt.Printf("Failed to unsubscribe from topic %s: %s\n", topic, resp.Error)
	}

	// 取消本地订阅
	c.mu.Lock()
	if cancel, exists := c.subs[topic]; exists {
		cancel()
		delete(c.subs, topic)
	}
	c.mu.Unlock()

	return nil
}

func (c *PubSubClient) ListTopics(pattern string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	req := &pb.ListTopicsRequest{
		Pattern: pattern,
	}

	resp, err := c.client.ListTopics(ctx, req)
	if err != nil {
		return fmt.Errorf("failed to list topics: %v", err)
	}

	fmt.Printf("Topics (%d):\n", len(resp.Topics))
	for _, topic := range resp.Topics {
		fmt.Printf("  - %s (subscribers: %d, messages: %d)\n",
			topic.Name, topic.SubscriberCount, topic.MessageCount)
		if topic.LastMessageTime > 0 {
			fmt.Printf("    Last message: %s\n",
				time.Unix(topic.LastMessageTime, 0).Format("2006-01-02 15:04:05"))
		}
	}

	return nil
}

func (c *PubSubClient) GetSubscriberCount(topic string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	req := &pb.GetSubscriberCountRequest{
		Topic: topic,
	}

	resp, err := c.client.GetSubscriberCount(ctx, req)
	if err != nil {
		return fmt.Errorf("failed to get subscriber count: %v", err)
	}

	if resp.Success {
		fmt.Printf("Topic %s has %d subscribers\n", topic, resp.SubscriberCount)
	} else {
		fmt.Printf("Failed to get subscriber count for topic %s: %s\n", topic, resp.Error)
	}

	return nil
}
