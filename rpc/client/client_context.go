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
	"github.com/CryptoElementals/common/stream"
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

// ClientContext holds package-global room/lobby gRPC clients and the shared event stream.
type ClientContext struct {
	RoomAddr     string
	LobbyAddr    string
	Conn         *grpc.ClientConn
	LobbyConn    *grpc.ClientConn
	RpcClient    pb.RoomServiceClient
	LobbyClient  pb.LobbyServiceClient
	EventStream  stream.Stream
	Mutex        sync.RWMutex
	initComplete bool
}

var grpcGlobals = &ClientContext{}

// dialLocked opens room and lobby connections. Caller must hold c.Mutex (write lock).
func (c *ClientContext) dialLocked() error {
	if c.Conn != nil {
		_ = c.Conn.Close()
		c.Conn = nil
	}
	if c.LobbyConn != nil {
		_ = c.LobbyConn.Close()
		c.LobbyConn = nil
	}
	roomConn, err := grpc.NewClient(c.RoomAddr, defaultGRPCDialOptions()...)
	if err != nil {
		return fmt.Errorf("dial room %s: %w", c.RoomAddr, err)
	}
	lobbyConn, err := grpc.NewClient(c.LobbyAddr, defaultGRPCDialOptions()...)
	if err != nil {
		_ = roomConn.Close()
		return fmt.Errorf("dial lobby %s: %w", c.LobbyAddr, err)
	}
	c.Conn = roomConn
	c.LobbyConn = lobbyConn
	c.RpcClient = pb.NewRoomServiceClient(roomConn)
	c.LobbyClient = pb.NewLobbyServiceClient(lobbyConn)
	if c.EventStream == nil {
		st, err := stream.NewRedisStream()
		if err != nil {
			_ = roomConn.Close()
			_ = lobbyConn.Close()
			c.Conn = nil
			c.LobbyConn = nil
			c.RpcClient = nil
			c.LobbyClient = nil
			return fmt.Errorf("redis stream (init redis before gRPC clients): %w", err)
		}
		c.EventStream = st
	}
	return nil
}

// InitGlobalClients connects to room (Rpc) and lobby (LobbyService). Requires [redis.Init] first for event streams.
func InitGlobalClients(roomAddress, lobbyAddress string) error {
	grpcGlobals.Mutex.Lock()
	defer grpcGlobals.Mutex.Unlock()

	if grpcGlobals.initComplete {
		return nil
	}
	grpcGlobals.RoomAddr = roomAddress
	grpcGlobals.LobbyAddr = lobbyAddress
	if err := grpcGlobals.dialLocked(); err != nil {
		return err
	}
	grpcGlobals.initComplete = true

	go grpcGlobals.startHealthCheck()

	log.Infof("全局gRPC客户端初始化成功，room=%s lobby=%s", roomAddress, lobbyAddress)
	return nil
}

// startHealthCheck 启动健康检查（监控 room 连接；失败时重连 room+lobby）
func (c *ClientContext) startHealthCheck() {
	ticker := time.NewTicker(10 * 60 * time.Second)
	defer ticker.Stop()

	var consecutiveFailures int
	const maxRetries = 3

	for range ticker.C {
		c.Mutex.RLock()
		conn := c.Conn
		c.Mutex.RUnlock()

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
				c.Mutex.Lock()
				err := c.dialLocked()
				c.Mutex.Unlock()
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

// GetGlobalRpcClient 获取 room RoomService 客户端
func GetGlobalRpcClient() pb.RoomServiceClient {
	grpcGlobals.Mutex.RLock()
	defer grpcGlobals.Mutex.RUnlock()
	return grpcGlobals.RpcClient
}

// GetGlobalLobbyClient 获取 lobby LobbyService 客户端
func GetGlobalLobbyClient() pb.LobbyServiceClient {
	grpcGlobals.Mutex.RLock()
	defer grpcGlobals.Mutex.RUnlock()
	return grpcGlobals.LobbyClient
}

// SetGlobalLobbyClientForTest overrides the global lobby client for tests.
func SetGlobalLobbyClientForTest(c pb.LobbyServiceClient) {
	grpcGlobals.Mutex.Lock()
	defer grpcGlobals.Mutex.Unlock()
	grpcGlobals.LobbyClient = c
}

// GetGlobalEventStream returns the shared Redis-backed stream used for game/lobby events (after InitGlobalClients).
func GetGlobalEventStream() stream.Stream {
	grpcGlobals.Mutex.RLock()
	defer grpcGlobals.Mutex.RUnlock()
	return grpcGlobals.EventStream
}

// CloseGlobalClients 关闭全局连接
func CloseGlobalClients() error {
	grpcGlobals.Mutex.Lock()
	defer grpcGlobals.Mutex.Unlock()

	var firstErr error
	if grpcGlobals.LobbyConn != nil {
		if err := grpcGlobals.LobbyConn.Close(); err != nil {
			firstErr = err
		}
		grpcGlobals.LobbyConn = nil
	}
	if grpcGlobals.Conn != nil {
		if err := grpcGlobals.Conn.Close(); err != nil && firstErr == nil {
			firstErr = err
		}
		grpcGlobals.Conn = nil
	}
	grpcGlobals.RpcClient = nil
	grpcGlobals.LobbyClient = nil
	grpcGlobals.EventStream = nil
	grpcGlobals.initComplete = false
	return firstErr
}
