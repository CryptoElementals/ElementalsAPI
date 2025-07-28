package client

import (
	"fmt"
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
	ticker := time.NewTicker(10 * time.Second) // 改为每10秒检查一次
	defer ticker.Stop()

	var consecutiveFailures int
	const maxRetries = 5

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
			switch state {
			case connectivity.Ready:
				// 连接正常，重置失败计数
				if consecutiveFailures > 0 {
					log.Infof("gRPC连接恢复正常")
					consecutiveFailures = 0
				}
			case connectivity.TransientFailure, connectivity.Shutdown:
				consecutiveFailures++
				log.Warnf("gRPC连接状态异常: %v，连续失败次数: %d/%d", state, consecutiveFailures, maxRetries)

				if consecutiveFailures >= maxRetries {
					log.Errorf("连续失败次数达到上限，尝试重新连接")

					// 重新连接
					if err := reconnectGlobalClients(serverAddress); err != nil {
						log.Errorf("重新连接失败: %v", err)
					} else {
						log.Infof("gRPC连接重新建立成功")
						consecutiveFailures = 0
					}
				}
			case connectivity.Connecting:
				log.Debugf("gRPC连接正在重连中...")
			case connectivity.Idle:
				log.Debugf("gRPC连接处于空闲状态")
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
		grpc.WithDefaultCallOptions(
			grpc.MaxCallRecvMsgSize(4*1024*1024), // 4MB
			grpc.MaxCallSendMsgSize(4*1024*1024), // 4MB
		),
	}

	conn, err := grpc.NewClient(serverAddress, opts...)
	if err != nil {
		return fmt.Errorf("failed to reconnect to %s: %v", serverAddress, err)
	}

	globalConn = conn
	globalRpcClient = pb.NewRpcServiceClient(conn)
	globalPubSubClient = pb.NewPubSubServiceClient(conn)

	log.Infof("gRPC连接重新建立成功到: %s", serverAddress)
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

// Client 统一的客户端接口，组合了 RPC 和 PubSub 客户端
type Client struct {
	conn         *grpc.ClientConn
	PubSubClient *PubSubClient
	RpcClient    *RpcClient
}

// NewClient 创建新的统一客户端
func NewClient(addr string) (*Client, error) {
	conn, err := grpc.NewClient(addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, err
	}
	return &Client{
		conn:         conn,
		PubSubClient: NewPubSubClient(conn),
		RpcClient:    NewRpcClient(conn),
	}, nil
}

// Close 关闭客户端连接
func (c *Client) Close() error {
	if c.PubSubClient != nil {
		c.PubSubClient.Close()
	}
	if c.conn != nil {
		return c.conn.Close()
	}
	return nil
}
