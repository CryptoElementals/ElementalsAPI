package botserver

import (
	"context"
	"fmt"

	"net"

	"github.com/CryptoElementals/common/config"
	"github.com/CryptoElementals/common/log"
	"github.com/CryptoElementals/common/rpc/proto"
	"github.com/CryptoElementals/common/rpc/server"
	"google.golang.org/grpc"
)

type BotServer struct {
	cfg        *config.BotConfig
	svr        *grpc.Server
	svc        *Service
	listenPort int
}

func NewBotServer(cfg *config.BotConfig) *BotServer {
	svr := grpc.NewServer()
	svc, err := NewService(context.Background(), cfg.WalletPaths, cfg.ChainCfg.HttpRpc, cfg.RoomServerEndpoint)
	if err != nil {
		log.Fatalw("cannot init bot server", "err", err)
	}
	grpcSvc := server.NewBotRpcServer(svc)

	proto.RegisterBotServiceServer(svr, grpcSvc)
	return &BotServer{
		cfg: cfg,
		svr: svr,
		svc: svc,
	}
}

func (s *BotServer) startListener() error {
	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", s.listenPort))
	if err != nil {
		return err
	}
	go func() {
		if err := s.svr.Serve(lis); err != nil {
			log.Fatalf("server start failed: %v", err)
		}
	}()

	return nil
}

func (s *BotServer) Start() error {
	s.svc.Start()
	return s.startListener()
}

func (s *BotServer) Stop() {
	s.svr.GracefulStop()
	s.svc.Stop()
}
