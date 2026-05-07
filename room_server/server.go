package roomserver

import (
	"context"
	"fmt"
	"net"

	"github.com/CryptoElementals/common/config"
	"github.com/CryptoElementals/common/log"
	"github.com/CryptoElementals/common/pubsub"
	"github.com/CryptoElementals/common/room_server/worker/chain"
	"github.com/CryptoElementals/common/room_server/worker/game"
	"github.com/CryptoElementals/common/rpc/middleware"
	"github.com/CryptoElementals/common/rpc/proto"
	rpc "github.com/CryptoElementals/common/rpc/server"
	"github.com/CryptoElementals/common/stream"
	"github.com/CryptoElementals/common/timer"
	"github.com/CryptoElementals/common/wallet"
	"google.golang.org/grpc"
)

type Service struct {
	ctx       context.Context
	cfg       *config.RoomServerConfig
	server    *grpc.Server
	chainSvc  *chain.Chain
	gameSvc   *game.Service
	rpcServer *rpc.Rpc
}

func New(ctx context.Context,
	cfg *config.RoomServerConfig,
	isDevelop ...bool) (*Service, error) {
	s := &Service{
		ctx: ctx,
		cfg: cfg,
	}
	wallets := make([]*wallet.Wallet, 0, len(cfg.WalletPaths))
	for _, path := range cfg.WalletPaths {
		w, err := wallet.LoadWallet(path)
		if err != nil {
			return nil, err
		}
		wallets = append(wallets, w)
	}
	chainSvc, err := chain.NewChain(ctx, cfg, wallets, isDevelop...)
	if err != nil {
		return nil, err
	}
	s.chainSvc = chainSvc
	st, err := stream.NewRedisStream()
	if err != nil {
		return nil, fmt.Errorf("redis stream: %w", err)
	}
	roomEventPub := pubsub.NewStreamPublisher(st, pubsub.TopicRoom)
	settlementPVPPub := pubsub.NewStreamPublisher(st, pubsub.TopicRoomSettlementPVP)
	settlementTournamentPub := pubsub.NewStreamPublisher(st, pubsub.TopicRoomSettlementTournament)
	s.gameSvc = game.NewService(ctx, roomEventPub, cfg.GameArgsID, chainSvc)
	s.gameSvc.SetGameResultSettler(newSettlementStreamPublisher(ctx, settlementPVPPub, settlementTournamentPub))
	server := grpc.NewServer(grpc.UnaryInterceptor(middleware.UnaryServerInterceptor))
	// game.Service implements chain/player/game handlers.
	rpcServer := rpc.NewRpc(s.gameSvc)
	s.rpcServer = rpcServer
	proto.RegisterRoomServiceServer(server, s.rpcServer)
	s.server = server
	return s, nil
}

func (s *Service) Start() error {
	log.Info("starting chain service")
	err := s.chainSvc.Start()
	if err != nil {
		return err
	}
	log.Info("chain service started")
	log.Info("starting game service")
	err = s.gameSvc.Start()
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
