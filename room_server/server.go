package roomserver

import (
	"context"
	"fmt"
	"net"
	"time"

	"github.com/CryptoElementals/common/config"
	"github.com/CryptoElementals/common/db"
	"github.com/CryptoElementals/common/log"
	"github.com/CryptoElementals/common/pubsub"
	"github.com/CryptoElementals/common/room_server/worker"
	"github.com/CryptoElementals/common/room_server/worker/chain"
	"github.com/CryptoElementals/common/room_server/worker/game"
	"github.com/CryptoElementals/common/room_server/worker/types"
	"github.com/CryptoElementals/common/rpc/proto"
	rpc "github.com/CryptoElementals/common/rpc/server"
	"github.com/CryptoElementals/common/stream"
	"github.com/CryptoElementals/common/wallet"
	"github.com/ethereum/go-ethereum/ethclient"
	"google.golang.org/grpc"
)

type Service struct {
	ctx       context.Context
	cfg       *config.RoomServerConfig
	mgr       *worker.WorkerManager
	server   *grpc.Server
	chainSvc *chain.Chain
	gameSvc   *game.Service
	rpcServer *rpc.Rpc
}

func New(ctx context.Context,
	cfg *config.RoomServerConfig,
	isDevelop ...bool) (*Service, error) {
	_ = isDevelop
	s := &Service{
		ctx: ctx,
		cfg: cfg,
		mgr: worker.NewWorkerManager(ctx),
	}
	client, err := ethclient.DialContext(ctx, cfg.ChainCfg.HttpRpc)
	if err != nil {
		return nil, err
	}
	chainID, err := client.ChainID(ctx)
	if err != nil {
		return nil, err
	}
	wallets := make([]*wallet.Wallet, 0, len(cfg.WalletPaths))
	for _, path := range cfg.WalletPaths {
		w, err := wallet.LoadWallet(path)
		if err != nil {
			return nil, err
		}
		wallets = append(wallets, w)
	}
	chainSvc, err := chain.NewChain(ctx, s.mgr, chainID.Int64(), client, cfg.ChainCfg.RoomV3ContractAddress, wallets)
	if err != nil {
		return nil, err
	}
	s.chainSvc = chainSvc
	argsTemplate, err := db.LoadRoomServerGameArgs(cfg.GameArgsID)
	if err != nil {
		log.Fatalf("game_args template required (game-args-id=%d): %v", cfg.GameArgsID, err)
	}
	st, err := stream.NewRedisStream()
	if err != nil {
		return nil, fmt.Errorf("redis stream: %w", err)
	}
	eventPub := pubsub.NewStreamPublisher(st)
	s.gameSvc = game.NewService(ctx, s.mgr, eventPub, argsTemplate, chainSvc, cfg.PoolBatchSize, cfg.PoolProcessingInterval)
	s.gameSvc.SetGameResultSettler(newSettlementStreamPublisher(ctx, eventPub))
	server := grpc.NewServer(grpc.UnaryInterceptor(UnaryServerInterceptor))
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

func UnaryServerInterceptor(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
	start := time.Now()

	resp, err := handler(ctx, req)

	duration := time.Since(start)
	if err != nil {
		log.Errorw("rpc called", "method", info.FullMethod, "req", types.ToJsonLoggable(req), "err", err, "duration", duration.Seconds())
	} else {
		log.Debugw("rpc called", "method", info.FullMethod, "req", types.ToJsonLoggable(req), "resp", types.ToJsonLoggable(resp), "duration", duration.Seconds())
	}

	return resp, err
}
