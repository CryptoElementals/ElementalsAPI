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

// ClientContext holds room/lobby gRPC clients and the shared event stream for one logical bundle (e.g. one shard).
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

func NewClientContext(roomAddress, lobbyAddress string) *ClientContext {
	return &ClientContext{
		RoomAddr:  roomAddress,
		LobbyAddr: lobbyAddress,
	}
}

// GlobalContextKey is the registry key used by InitGlobalClients and GetGlobal* helpers.
const GlobalContextKey = "GLOBAL"

var (
	clientContextsMu sync.RWMutex
	clientContexts   = make(map[string]*ClientContext)
)

// RegisterClientContexts pre-creates empty client context slots for the given keys (no dialing).
// Calling with no arguments is a no-op.
func RegisterClientContexts(keys ...string) {
	clientContextsMu.Lock()
	defer clientContextsMu.Unlock()
	for _, k := range keys {
		if k == "" {
			continue
		}
		if clientContexts[k] == nil {
			clientContexts[k] = &ClientContext{}
		}
	}
}

func getOrCreateClientContext(key string) *ClientContext {
	clientContextsMu.Lock()
	defer clientContextsMu.Unlock()
	if c := clientContexts[key]; c != nil {
		return c
	}
	c := &ClientContext{}
	clientContexts[key] = c
	return c
}

func getClientContext(key string) (*ClientContext, bool) {
	clientContextsMu.RLock()
	defer clientContextsMu.RUnlock()
	c, ok := clientContexts[key]
	return c, ok && c != nil
}

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

// Init dials room and lobby, creates the Redis event stream if needed, and starts the health check loop.
// Caller must hold c.Mutex (write lock).
func (c *ClientContext) Init() error {
	if c.initComplete {
		return nil
	}
	if err := c.dialLocked(); err != nil {
		return err
	}
	c.initComplete = true
	go c.startHealthCheck()
	log.Infof("全局gRPC客户端初始化成功，room=%s lobby=%s", c.RoomAddr, c.LobbyAddr)
	return nil
}

// InitClientContext connects to room (Rpc) and lobby (LobbyService) for the named context. Requires [redis.Init] first for event streams.
func InitClientContext(key, roomAddress, lobbyAddress string) error {
	if key == "" {
		return fmt.Errorf("rpc/client: empty client context key")
	}
	c := getOrCreateClientContext(key)
	c.Mutex.Lock()
	defer c.Mutex.Unlock()
	if c.initComplete {
		return nil
	}
	cfg := NewClientContext(roomAddress, lobbyAddress)
	c.RoomAddr = cfg.RoomAddr
	c.LobbyAddr = cfg.LobbyAddr
	return c.Init()
}

// InitGlobalClients connects to room (Rpc) and lobby (LobbyService) for [GlobalContextKey].
func InitGlobalClients(roomAddress, lobbyAddress string) error {
	return InitClientContext(GlobalContextKey, roomAddress, lobbyAddress)
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

// GetRoomServiceClient returns the room RoomService client for key, or nil if missing or not initialized.
func GetRoomServiceClient(key string) pb.RoomServiceClient {
	c, ok := getClientContext(key)
	if !ok {
		return nil
	}
	c.Mutex.RLock()
	defer c.Mutex.RUnlock()
	return c.RpcClient
}

// GetGlobalRpcClient 获取 room RoomService 客户端
func GetGlobalRpcClient() pb.RoomServiceClient {
	return GetRoomServiceClient(GlobalContextKey)
}

// GetLobbyServiceClient returns the lobby LobbyService client for key, or nil if missing or not initialized.
func GetLobbyServiceClient(key string) pb.LobbyServiceClient {
	c, ok := getClientContext(key)
	if !ok {
		return nil
	}
	c.Mutex.RLock()
	defer c.Mutex.RUnlock()
	return c.LobbyClient
}

// GetGlobalLobbyClient 获取 lobby LobbyService 客户端
func GetGlobalLobbyClient() pb.LobbyServiceClient {
	return GetLobbyServiceClient(GlobalContextKey)
}

// SetLobbyClientForTest overrides the lobby client for tests for the given context key.
func SetLobbyClientForTest(key string, cl pb.LobbyServiceClient) {
	if key == "" {
		key = GlobalContextKey
	}
	c := getOrCreateClientContext(key)
	c.Mutex.Lock()
	defer c.Mutex.Unlock()
	c.LobbyClient = cl
}

// SetGlobalLobbyClientForTest overrides the global lobby client for tests.
func SetGlobalLobbyClientForTest(c pb.LobbyServiceClient) {
	SetLobbyClientForTest(GlobalContextKey, c)
}

// GetEventStream returns the Redis-backed stream for key, or nil if missing or not initialized.
func GetEventStream(key string) stream.Stream {
	c, ok := getClientContext(key)
	if !ok {
		return nil
	}
	c.Mutex.RLock()
	defer c.Mutex.RUnlock()
	return c.EventStream
}

// GetGlobalEventStream returns the shared Redis-backed stream used for game/lobby events (after InitGlobalClients).
func GetGlobalEventStream() stream.Stream {
	return GetEventStream(GlobalContextKey)
}

// CloseClientContext closes connections for the named context. The slot remains in the registry for reuse.
func CloseClientContext(key string) error {
	c, ok := getClientContext(key)
	if !ok {
		return nil
	}
	c.Mutex.Lock()
	defer c.Mutex.Unlock()

	var firstErr error
	if c.LobbyConn != nil {
		if err := c.LobbyConn.Close(); err != nil {
			firstErr = err
		}
		c.LobbyConn = nil
	}
	if c.Conn != nil {
		if err := c.Conn.Close(); err != nil && firstErr == nil {
			firstErr = err
		}
		c.Conn = nil
	}
	c.RpcClient = nil
	c.LobbyClient = nil
	c.EventStream = nil
	c.initComplete = false
	return firstErr
}

// CloseGlobalClients 关闭全局连接
func CloseGlobalClients() error {
	return CloseClientContext(GlobalContextKey)
}
