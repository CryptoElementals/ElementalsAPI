package chainserver

import (
	"context"
	"fmt"
	"net"

	"github.com/CryptoElementals/common/chain_server/worker"
	"github.com/CryptoElementals/common/config"
	"github.com/CryptoElementals/common/log"
	"github.com/CryptoElementals/common/rpc/middleware"
	"github.com/CryptoElementals/common/rpc/proto"
	"google.golang.org/grpc"
)

// Service is the chain server process: tx pool drain and on-chain submissions.
type Service struct {
	ctx          context.Context
	cfg          *config.ChainServerConfig
	grpcServer   *grpc.Server
	chainWorker  *worker.Chain
	grpcHandlers *GRPCServices
}

// New constructs a chain server. Call Start after DB is initialized.
func New(ctx context.Context, cfg *config.ChainServerConfig, isDevelop ...bool) (*Service, error) {
	s := &Service{ctx: ctx, cfg: cfg}

	chainWorker, err := worker.NewChain(ctx, cfg, isDevelop...)
	if err != nil {
		return nil, err
	}
	s.chainWorker = chainWorker
	s.grpcHandlers = NewGRPCServices(chainWorker)

	server := grpc.NewServer(grpc.UnaryInterceptor(middleware.UnaryServerInterceptor))
	proto.RegisterChainServiceServer(server, s.grpcHandlers)
	s.grpcServer = server
	return s, nil
}

// Start runs the pool ticker and gRPC listener.
func (s *Service) Start() error {
	log.Info("starting chain worker")
	if err := s.chainWorker.Start(); err != nil {
		return err
	}
	log.Info("chain worker started")
	return s.startListener()
}

// Stop shuts down the chain worker and gRPC server.
func (s *Service) Stop() {
	log.Info("stopping chain worker")
	s.chainWorker.Stop()
	log.Info("chain worker stopped")
	log.Info("stopping grpc server")
	s.grpcServer.GracefulStop()
	log.Info("grpc server stopped")
}

func (s *Service) startListener() error {
	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", s.cfg.ListenPort))
	if err != nil {
		return err
	}
	go func() {
		if err := s.grpcServer.Serve(lis); err != nil {
			log.Fatalf("chain server grpc serve failed: %v", err)
		}
	}()
	log.Infow("chain server listening", "port", s.cfg.ListenPort)
	return nil
}
