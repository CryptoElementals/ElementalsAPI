package client

import (
	"fmt"
	"sync"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/connectivity"
	"google.golang.org/grpc/credentials/insecure"

	"github.com/CryptoElementals/common/log"
	pb "github.com/CryptoElementals/common/rpc/proto"
)

func defaultGRPCDialOptions() []grpc.DialOption {
	return []grpc.DialOption{
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithDefaultCallOptions(
			grpc.MaxCallRecvMsgSize(4*1024*1024),
			grpc.MaxCallSendMsgSize(4*1024*1024),
		),
	}
}

// 全局gRPC客户端变量（room + lobby）
var (
	globalRoomAddr     string
	globalLobbyAddr    string
	globalConn         *grpc.ClientConn
	globalLobbyConn    *grpc.ClientConn
	globalRpcClient    pb.RpcServiceClient
	globalLobbyClient  pb.LobbyServiceClient
	globalPubSubClient      pb.PubSubServiceClient
	globalLobbyPubSubClient pb.PubSubServiceClient
	globalMutex             sync.RWMutex
	initialized        bool
)

// dialGlobalLocked opens room and lobby connections. Caller must hold globalMutex (write lock).
func dialGlobalLocked() error {
	if globalConn != nil {
		_ = globalConn.Close()
		globalConn = nil
	}
	if globalLobbyConn != nil {
		_ = globalLobbyConn.Close()
		globalLobbyConn = nil
	}
	roomConn, err := grpc.NewClient(globalRoomAddr, defaultGRPCDialOptions()...)
	if err != nil {
		return fmt.Errorf("dial room %s: %w", globalRoomAddr, err)
	}
	lobbyConn, err := grpc.NewClient(globalLobbyAddr, defaultGRPCDialOptions()...)
	if err != nil {
		_ = roomConn.Close()
		return fmt.Errorf("dial lobby %s: %w", globalLobbyAddr, err)
	}
	globalConn = roomConn
	globalLobbyConn = lobbyConn
	globalRpcClient = pb.NewRpcServiceClient(roomConn)
	globalLobbyClient = pb.NewLobbyServiceClient(lobbyConn)
	globalPubSubClient = pb.NewPubSubServiceClient(roomConn)
	globalLobbyPubSubClient = pb.NewPubSubServiceClient(lobbyConn)
	return nil
}

// InitGlobalClients connects to room (Rpc + PubSub) and lobby (LobbyService).
func InitGlobalClients(roomAddress, lobbyAddress string) error {
	globalMutex.Lock()
	defer globalMutex.Unlock()

	if initialized {
		return nil
	}
	globalRoomAddr = roomAddress
	globalLobbyAddr = lobbyAddress
	if err := dialGlobalLocked(); err != nil {
		return err
	}
	initialized = true

	go startHealthCheck()

	log.Infof("全局gRPC客户端初始化成功，room=%s lobby=%s", roomAddress, lobbyAddress)
	return nil
}

// startHealthCheck 启动健康检查（监控 room 连接；失败时重连 room+lobby）
func startHealthCheck() {
	ticker := time.NewTicker(10 * 60 * time.Second)
	defer ticker.Stop()

	var consecutiveFailures int
	const maxRetries = 3

	for range ticker.C {
		globalMutex.RLock()
		conn := globalConn
		globalMutex.RUnlock()

		if conn == nil {
			continue
		}

		state := conn.GetState()
		switch state {
		case connectivity.Ready:
			if consecutiveFailures > 0 {
				log.Infof("gRPC连接恢复正常")
				consecutiveFailures = 0
			}
		case connectivity.TransientFailure, connectivity.Shutdown:
			consecutiveFailures++
			log.Warnf("gRPC连接状态异常: %v，连续失败次数: %d/%d", state, consecutiveFailures, maxRetries)

			if consecutiveFailures >= maxRetries {
				log.Errorf("连续失败次数达到上限，尝试重新连接 room+lobby")
				globalMutex.Lock()
				err := dialGlobalLocked()
				globalMutex.Unlock()
				if err != nil {
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

// GetGlobalRpcClient 获取 room RpcService 客户端
func GetGlobalRpcClient() pb.RpcServiceClient {
	globalMutex.RLock()
	defer globalMutex.RUnlock()
	return globalRpcClient
}

// GetGlobalLobbyClient 获取 lobby LobbyService 客户端
func GetGlobalLobbyClient() pb.LobbyServiceClient {
	globalMutex.RLock()
	defer globalMutex.RUnlock()
	return globalLobbyClient
}

// GetGlobalPubSubClient 获取 room PubSub 客户端
func GetGlobalPubSubClient() pb.PubSubServiceClient {
	globalMutex.RLock()
	defer globalMutex.RUnlock()
	return globalPubSubClient
}

// GetGlobalLobbyPubSubClient 获取 lobby PubSub 客户端
func GetGlobalLobbyPubSubClient() pb.PubSubServiceClient {
	globalMutex.RLock()
	defer globalMutex.RUnlock()
	return globalLobbyPubSubClient
}

// CloseGlobalClients 关闭全局连接
func CloseGlobalClients() error {
	globalMutex.Lock()
	defer globalMutex.Unlock()

	var firstErr error
	if globalLobbyConn != nil {
		if err := globalLobbyConn.Close(); err != nil {
			firstErr = err
		}
		globalLobbyConn = nil
	}
	if globalConn != nil {
		if err := globalConn.Close(); err != nil && firstErr == nil {
			firstErr = err
		}
		globalConn = nil
	}
	globalRpcClient = nil
	globalLobbyClient = nil
	globalPubSubClient = nil
	globalLobbyPubSubClient = nil
	initialized = false
	return firstErr
}

// Client 统一的客户端接口，组合了 RPC 和 PubSub 客户端
type Client struct {
	roomConn  *grpc.ClientConn
	lobbyConn *grpc.ClientConn
	*PubSubClient
	*RpcClient
}

// NewClient 连接 room 与 lobby（RPC + 双 PubSub）。
func NewClient(roomAddr, lobbyAddr string) (*Client, error) {
	opts := defaultGRPCDialOptions()
	roomConn, err := grpc.NewClient(roomAddr, opts...)
	if err != nil {
		return nil, err
	}
	lobbyConn, err := grpc.NewClient(lobbyAddr, opts...)
	if err != nil {
		_ = roomConn.Close()
		return nil, err
	}
	lobbySvc := pb.NewLobbyServiceClient(lobbyConn)
	return &Client{
		roomConn:     roomConn,
		lobbyConn:    lobbyConn,
		PubSubClient: NewPubSubClientDual(pb.NewPubSubServiceClient(roomConn), pb.NewPubSubServiceClient(lobbyConn)),
		RpcClient:    NewRpcClient(roomConn, lobbyConn, lobbySvc),
	}, nil
}

// Close 关闭客户端连接
func (c *Client) Close() error {
	if c.PubSubClient != nil {
		_ = c.PubSubClient.Close()
	}
	if c.lobbyConn != nil {
		_ = c.lobbyConn.Close()
	}
	if c.roomConn != nil {
		return c.roomConn.Close()
	}
	return nil
}
