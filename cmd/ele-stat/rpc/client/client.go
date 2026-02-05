package client

import (
	"context"
	"fmt"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/keepalive"

	"github.com/CryptoElementals/common/cmd/ele-stat/proto"
	"github.com/CryptoElementals/common/log"
)

// HealthCheckResponse health check response
type HealthCheckResponse struct {
	Status    string `json:"status"`
	Uptime    string `json:"uptime"`
	Timestamp int64  `json:"timestamp"`
	Message   string `json:"message"`
}

// StatClient gRPC client struct
type StatClient struct {
	conn       *grpc.ClientConn
	config     *ClientConfig
	grpcClient proto.StatServiceClient // Cached gRPC client
}

// ClientConfig client configuration
type ClientConfig struct {
	ServerAddr            string
	ConnectionTimeout     time.Duration
	KeepaliveTime         time.Duration
	KeepaliveTimeout      time.Duration
	MaxReceiveMessageSize int
	MaxSendMessageSize    int
}

// DefaultClientConfig default client configuration
func DefaultClientConfig(serverAddr string) *ClientConfig {
	return &ClientConfig{
		ServerAddr:            serverAddr,
		ConnectionTimeout:     10 * time.Second,
		KeepaliveTime:         30 * time.Second,
		KeepaliveTimeout:      5 * time.Second,
		MaxReceiveMessageSize: 4 * 1024 * 1024, // 4MB
		MaxSendMessageSize:    4 * 1024 * 1024, // 4MB
	}
}

// NewStatClient creates a new statistics client
func NewStatClient(config *ClientConfig) (*StatClient, error) {
	// Create gRPC connection options
	dialOptions := []grpc.DialOption{
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithBlock(),
		grpc.WithTimeout(config.ConnectionTimeout),
		grpc.WithKeepaliveParams(keepalive.ClientParameters{
			Time:                config.KeepaliveTime,
			Timeout:             config.KeepaliveTimeout,
			PermitWithoutStream: true,
		}),
		grpc.WithDefaultCallOptions(
			grpc.MaxCallRecvMsgSize(config.MaxReceiveMessageSize),
			grpc.MaxCallSendMsgSize(config.MaxSendMessageSize),
		),
	}

	// Establish connection
	conn, err := grpc.Dial(config.ServerAddr, dialOptions...)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to server %s: %v", config.ServerAddr, err)
	}

	log.Infof("Successfully connected to gRPC server at %s", config.ServerAddr)

	return &StatClient{
		conn:       conn,
		config:     config,
		grpcClient: proto.NewStatServiceClient(conn),
	}, nil
}

// Close close client connection
func (c *StatClient) Close() error {
	if c.conn != nil {
		log.Info("Closing gRPC client connection...")
		return c.conn.Close()
	}
	return nil
}

// GetConnection get underlying connection
func (c *StatClient) GetConnection() *grpc.ClientConn {
	return c.conn
}

// GetConfig get client configuration
func (c *StatClient) GetConfig() *ClientConfig {
	return c.config
}

// IsConnected check connection status
func (c *StatClient) IsConnected() bool {
	if c.conn == nil {
		return false
	}
	state := c.conn.GetState()
	return state.String() == "READY"
}

// WaitForConnection wait for connection to be ready
func (c *StatClient) WaitForConnection(ctx context.Context) error {
	if c.conn == nil {
		return fmt.Errorf("client not initialized")
	}

	// Wait for connection state change
	c.conn.WaitForStateChange(ctx, c.conn.GetState())
	return nil
}

// GetConnectionState get connection state
func (c *StatClient) GetConnectionState() string {
	if c.conn == nil {
		return "NOT_INITIALIZED"
	}
	return c.conn.GetState().String()
}

// UpdatePlayerStats 请求统计服务增量更新玩家统计（user_stat + card_stat）
func (c *StatClient) UpdatePlayerStats(ctx context.Context, playerIDs []int64) (*proto.UpdatePlayerStatsResponse, error) {
	if c.grpcClient == nil {
		return nil, fmt.Errorf("client not initialized")
	}
	return c.grpcClient.UpdatePlayerStats(ctx, &proto.UpdatePlayerStatsRequest{PlayerIds: playerIDs})
}

// HealthCheck perform health check directly
func (c *StatClient) HealthCheck(clientID string) (*HealthCheckResponse, error) {
	// Check if client is connected
	if !c.IsConnected() {
		return nil, fmt.Errorf("client not connected to server")
	}

	// Set timeout context
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	req := &proto.HealthCheckRequest{
		ClientId: clientID,
	}
	resp, err := c.grpcClient.HealthCheck(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("health check failed: %v", err)
	}

	// Convert to local response type
	return &HealthCheckResponse{
		Status:    resp.Status,
		Uptime:    resp.Uptime,
		Timestamp: resp.Timestamp,
		Message:   resp.Message,
	}, nil
}

// HealthCheckWithRetry perform health check with retry mechanism
func (c *StatClient) HealthCheckWithRetry(clientID string, maxRetries int) (*HealthCheckResponse, error) {
	var lastErr error

	for attempt := 1; attempt <= maxRetries; attempt++ {
		log.Infof("Health check attempt %d/%d for client: %s", attempt, maxRetries, clientID)

		resp, err := c.HealthCheck(clientID)
		if err == nil {
			log.Infof("Health check successful on attempt %d", attempt)
			return resp, nil
		}

		lastErr = err
		log.Warnf("Health check attempt %d failed: %v", attempt, err)

		if attempt < maxRetries {
			// Wait before retry (exponential backoff)
			waitTime := time.Duration(attempt) * time.Second
			log.Infof("Waiting %v before retry...", waitTime)
			time.Sleep(waitTime)
		}
	}

	return nil, fmt.Errorf("health check failed after %d attempts: %v", maxRetries, lastErr)
}
