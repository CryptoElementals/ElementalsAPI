package roomserver

import (
	"context"
	"fmt"
	"net"

	"github.com/CryptoElementals/common/config"
	"github.com/CryptoElementals/common/log"
	"github.com/CryptoElementals/common/pubsub"
	"github.com/CryptoElementals/common/room_server/chainclient"
	"github.com/CryptoElementals/common/room_server/worker/game"
	"github.com/CryptoElementals/common/rpc/middleware"
	"github.com/CryptoElementals/common/rpc/proto"
	rpc "github.com/CryptoElementals/common/rpc/server"
	"github.com/CryptoElementals/common/stream"
	"github.com/CryptoElementals/common/timer"
	"google.golang.org/grpc"
)

type Service struct {
	ctx        context.Context
	cfg        *config.RoomServerConfig
	server     *grpc.Server
	chainConn  *chainclient.Client
	gameSvc    *game.Service
	rpcServer  *rpc.Rpc
}

func New(ctx context.Context,
	cfg *config.RoomServerConfig,
	isDevelop ...bool) (*Service, error) {
	_ = isDevelop
	s := &Service{
		ctx: ctx,
		cfg: cfg,
	}
	if cfg.ChainServerAddress == "" {
		return nil, fmt.Errorf("chain-server-address is required")
	}
	chainConn, err := chainclient.Dial(ctx, cfg.ChainServerAddress)
	if err != nil {
		return nil, fmt.Errorf("dial chain server: %w", err)
	}
	s.chainConn = chainConn

	st, err := stream.NewRedisStream()
	if err != nil {
		return nil, fmt.Errorf("redis stream: %w", err)
	}
	roomEventPub := pubsub.NewStreamPublisher(st, pubsub.TopicRoom)
	settlementPVPPub := pubsub.NewStreamPublisher(st, pubsub.TopicRoomSettlementPVP)
	settlementTournamentPub := pubsub.NewStreamPublisher(st, pubsub.TopicRoomSettlementTournament)
	s.gameSvc = game.NewService(ctx, roomEventPub, cfg.GameArgsID, chainConn)
	s.gameSvc.SetGameResultSettler(newSettlementStreamPublisher(ctx, settlementPVPPub, settlementTournamentPub))
	server := grpc.NewServer(grpc.UnaryInterceptor(middleware.UnaryServerInterceptor))
	rpcServer := rpc.NewRpc(s.gameSvc)
	s.rpcServer = rpcServer
	proto.RegisterRoomServiceServer(server, s.rpcServer)
	s.server = server
	return s, nil
}

func (s *Service) Start() error {
	log.Info("starting game service")
	err := s.gameSvc.Start()
	if err != nil {
		return err
	}
	log.Info("game service started")

	timer.StartTimer(timer.ScopeRoom)
	log.Info("starting listener")
	err = s.startListener()
	if err != nil {
		return err
	}
	log.Info("listener started")

	return nil
}

func (s *Service) Stop() {
	log.Info("stopping game service")
	s.gameSvc.Stop()
	log.Info("game service stopped")
	log.Info("stopping grpc server")
	s.server.GracefulStop()
	log.Info("grpc server stopped")
	if s.chainConn != nil {
		_ = s.chainConn.Close()
	}
}

func (s *Service) startListener() error {
	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", s.cfg.ListenPort))
	if err != nil {
		return err
	}
	go func() {
		if err := s.server.Serve(lis); err != nil {
			log.Fatalf("server start failed: %v", err)
		}
	}()

	return nil
}
