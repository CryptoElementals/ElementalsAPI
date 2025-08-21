package server

import (
	"context"
	"fmt"
	"net"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/keepalive"
	"google.golang.org/grpc/reflection"

	"github.com/CryptoElementals/common/cmd/ele-stat/proto"
	"github.com/CryptoElementals/common/log"
)

// StatServer statistics service struct
type StatServer struct {
	grpcServer *grpc.Server
	listener   net.Listener
	port       uint32
}

// ServerConfig server configuration
type ServerConfig struct {
	Port                  uint32
	ConnectionTimeout     time.Duration
	MaxConnectionIdle     time.Duration
	MaxConnectionAge      time.Duration
	KeepaliveTime         time.Duration
	KeepaliveTimeout      time.Duration
	MaxConcurrentStreams  uint32
	MaxReceiveMessageSize int
	MaxSendMessageSize    int
}

// DefaultServerConfig default server configuration
func DefaultServerConfig(port uint32) *ServerConfig {
	return &ServerConfig{
		Port:                  port,
		ConnectionTimeout:     30 * time.Second,
		MaxConnectionIdle:     5 * time.Minute,
		MaxConnectionAge:      10 * time.Minute,
		KeepaliveTime:         30 * time.Second,
		KeepaliveTimeout:      5 * time.Second,
		MaxConcurrentStreams:  1000,
		MaxReceiveMessageSize: 4 * 1024 * 1024, // 4MB
		MaxSendMessageSize:    4 * 1024 * 1024, // 4MB
	}
}

// NewStatServer create new statistics service
func NewStatServer(config *ServerConfig) (*StatServer, error) {
	// Create gRPC server options
	serverOptions := []grpc.ServerOption{
		grpc.ConnectionTimeout(config.ConnectionTimeout),
		grpc.KeepaliveParams(keepalive.ServerParameters{
			MaxConnectionIdle: config.MaxConnectionIdle,
			MaxConnectionAge:  config.MaxConnectionAge,
			Time:              config.KeepaliveTime,
			Timeout:           config.KeepaliveTimeout,
		}),
		grpc.MaxConcurrentStreams(config.MaxConcurrentStreams),
		grpc.MaxRecvMsgSize(config.MaxReceiveMessageSize),
		grpc.MaxSendMsgSize(config.MaxSendMessageSize),
	}

	// Create gRPC server
	grpcServer := grpc.NewServer(serverOptions...)

	// Register all services
	RegisterAllServices(grpcServer)

	// Create listener
	listener, err := net.Listen("tcp", fmt.Sprintf(":%d", config.Port))
	if err != nil {
		return nil, fmt.Errorf("failed to create listener: %v", err)
	}

	return &StatServer{
		grpcServer: grpcServer,
		listener:   listener,
		port:       config.Port,
	}, nil
}

// Start start the service
func (s *StatServer) Start() error {
	log.Infof("Starting gRPC server on port %d", s.port)

	if err := s.grpcServer.Serve(s.listener); err != nil {
		return fmt.Errorf("service startup failed: %v", err)
	}

	return nil
}

// Stop stop the service
func (s *StatServer) Stop() error {
	log.Info("Stopping gRPC server...")

	// Graceful shutdown, wait for existing connections to complete
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	done := make(chan struct{})
	go func() {
		s.grpcServer.GracefulStop()
		close(done)
	}()

	select {
	case <-done:
		log.Info("gRPC server stopped gracefully")
	case <-ctx.Done():
		log.Warn("gRPC server graceful stop timeout, forcing stop")
		s.grpcServer.Stop()
	}

	return nil
}

// GetPort get service port
func (s *StatServer) GetPort() uint32 {
	return s.port
}

// GetServerInfo get server info
func (s *StatServer) GetServerInfo() map[string]interface{} {
	return map[string]interface{}{
		"port":     s.port,
		"address":  s.listener.Addr().String(),
		"services": "Use gRPC reflection to discover services dynamically",
	}
}

// RegisterAllServices register all gRPC services
func RegisterAllServices(grpcServer *grpc.Server) {
	// Register stat check service
	statService := NewStatService()
	proto.RegisterStatServiceServer(grpcServer, statService)

	// Register other services (can be extended)
	// userService := user.NewUserService()
	// proto.RegisterUserServiceServer(grpcServer, userService)

	// Enable reflection service (for debugging and testing)
	reflection.Register(grpcServer)

	log.Info("All gRPC services registered successfully")
	log.Info("Use gRPC reflection to discover registered services dynamically")
}

// GetRegisteredServices get the list of registered services using gRPC reflection
// This function now attempts to dynamically discover services
func GetRegisteredServices() []string {
	// Try to get services dynamically from a running server
	// If no server is running, return helpful information
	services, err := GetServicesDynamically("localhost:30011")
	if err != nil {
		// Server not running, return helpful information
		return []string{
			"Note: Server not running or not accessible",
			"Start the server first, then use:",
			"  grpcurl -plaintext localhost:30011 list",
			"  Or use reflection client in Go code",
		}
	}

	// Return dynamically discovered services
	return services
}

// GetServicesDynamically get services using gRPC reflection (requires server to be running)
func GetServicesDynamically(serverAddr string) ([]string, error) {
	// This function demonstrates how to get services dynamically
	// It requires the server to be running and accessible
	return GetServicesViaReflection(serverAddr)
}
