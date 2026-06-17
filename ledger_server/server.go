package ledgerserver

import (
	"context"
	"fmt"
	"net"

	"github.com/CryptoElementals/common/config"
	"github.com/CryptoElementals/common/log"
	"github.com/CryptoElementals/common/rpc/middleware"
	"github.com/CryptoElementals/common/rpc/proto"
	"google.golang.org/grpc"
)

// ServiceProcess is the ledger server process.
type ServiceProcess struct {
	ctx          context.Context
	cfg          *config.LedgerServerConfig
	grpcServer   *grpc.Server
	grpcHandlers *GRPCServices
}

// New constructs a ledger server. Call Start after DB is initialized.
func New(ctx context.Context, cfg *config.LedgerServerConfig) (*ServiceProcess, error) {
	svc := NewService()
	handlers := NewGRPCServices(svc)
	server := grpc.NewServer(grpc.UnaryInterceptor(middleware.UnaryServerInterceptor))
	proto.RegisterLedgerServiceServer(server, handlers)
	return &ServiceProcess{
		ctx:          ctx,
		cfg:          cfg,
		grpcServer:   server,
		grpcHandlers: handlers,
	}, nil
}

// Start runs the gRPC listener.
func (s *ServiceProcess) Start() error {
	return s.startListener()
}

// Stop shuts down the gRPC server.
func (s *ServiceProcess) Stop() {
	log.Info("stopping ledger server grpc")
	s.grpcServer.GracefulStop()
	log.Info("ledger server grpc stopped")
}

func (s *ServiceProcess) startListener() error {
	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", s.cfg.ListenPort))
	if err != nil {
		return err
	}
	go func() {
		if err := s.grpcServer.Serve(lis); err != nil {
			log.Fatalf("ledger server grpc serve failed: %v", err)
		}
	}()
	log.Infow("ledger server listening", "port", s.cfg.ListenPort)
	return nil
}
