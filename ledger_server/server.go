package ledgerserver

import (
	"context"
	"fmt"
	"net"

	"github.com/CryptoElementals/common/config"
	"github.com/CryptoElementals/common/ledger_server/chainclient"
	"github.com/CryptoElementals/common/log"
	"github.com/CryptoElementals/common/pubsub"
	"github.com/CryptoElementals/common/rpc/middleware"
	"github.com/CryptoElementals/common/rpc/proto"
	"github.com/CryptoElementals/common/stream"
	"google.golang.org/grpc"
)

// ServiceProcess is the ledger server process.
type ServiceProcess struct {
	ctx          context.Context
	cfg          *config.LedgerServerConfig
	grpcServer   *grpc.Server
	grpcHandlers *GRPCServices
	eventStream  stream.Stream
	chainClient  *chainclient.Client
}

// New constructs a ledger server. Call Start after DB is initialized.
func New(ctx context.Context, cfg *config.LedgerServerConfig) (*ServiceProcess, error) {
	st, err := stream.NewRedisStream()
	if err != nil {
		return nil, fmt.Errorf("redis stream: %w", err)
	}
	tokenPublisher := pubsub.NewStreamPublisher(st, pubsub.TopicToken)

	var chainCli *chainclient.Client
	if cfg != nil && cfg.UChainCfg.ChainServerAddress != "" {
		chainCli, err = chainclient.Dial(ctx, cfg.UChainCfg.ChainServerAddress)
		if err != nil {
			return nil, fmt.Errorf("dial chain server: %w", err)
		}
	}

	chainID := int64(0)
	if cfg != nil {
		chainID = cfg.UChainCfg.ChainID
	}
	svc := NewService(tokenPublisher, chainCli, chainID)
	handlers := NewGRPCServices(svc)
	server := grpc.NewServer(grpc.UnaryInterceptor(middleware.UnaryServerInterceptor))
	proto.RegisterLedgerServiceServer(server, handlers)
	return &ServiceProcess{
		ctx:          ctx,
		cfg:          cfg,
		grpcServer:   server,
		grpcHandlers: handlers,
		eventStream:  st,
		chainClient:  chainCli,
	}, nil
}

// Start runs the gRPC listener.
func (s *ServiceProcess) Start() error {
	return s.startListener()
}

// Stop shuts down the gRPC server.
func (s *ServiceProcess) Stop() {
	log.Info("stopping ledger server grpc")
	if s.chainClient != nil {
		_ = s.chainClient.Close()
	}
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
