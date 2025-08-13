package roomserver

import (
	"context"
	"fmt"
	"log"
	"net"

	"github.com/CryptoElementals/common/cache"
	"github.com/CryptoElementals/common/config"
	"github.com/CryptoElementals/common/room_server/worker"
	"github.com/CryptoElementals/common/room_server/worker/chain"
	"github.com/CryptoElementals/common/room_server/worker/game"
	"github.com/CryptoElementals/common/room_server/worker/player"
	"github.com/CryptoElementals/common/room_server/worker/queue"
	"github.com/CryptoElementals/common/rpc/proto"
	rpc "github.com/CryptoElementals/common/rpc/server"
	"github.com/CryptoElementals/common/wallet"
	"github.com/ethereum/go-ethereum/ethclient"
	"google.golang.org/grpc"
)

type Service struct {
	ctx       context.Context
	cfg       *config.RoomServerConfig
	mgr       *worker.WorkerManager
	server    *grpc.Server
	pubsub    *rpc.PubSub
	rpcServer *rpc.Rpc
	chainSvc  *chain.Service
	gameSvc   *game.Service
	playerSvc *player.Service
	queueSvc  *queue.Service
}

func New(ctx context.Context,
	cfg *config.RoomServerConfig,
	isDevelop ...bool) (*Service, error) {
	s := &Service{
		ctx:    ctx,
		cfg:    cfg,
		mgr:    worker.NewWorkerManager(ctx),
		pubsub: rpc.NewPubSub(),
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
	var c cache.Cache
	if len(isDevelop) != 0 && isDevelop[0] {
		c = cache.NewMemCache()
	} else {
		c, err = cache.NewRedisCache()
		if err != nil {
			return nil, err
		}
	}

	chainSvc, err := chain.NewService(ctx, s.mgr, chainID.Int64(), client, cfg.ChainCfg.RoomManagerAddress, wallets, c)
	if err != nil {
		return nil, err
	}
	s.chainSvc = chainSvc
	gameSvc := game.NewService(ctx, s.mgr, cfg.GameInitialHP, cfg.RoundTimeout, cfg.MaxRounds, chainSvc)
	s.gameSvc = gameSvc
	queueSvc := queue.NewService(ctx, s.mgr, c, gameSvc, int32(cfg.GameParams.TokenThreshold), cfg.ContinueTimeout, cfg.BotWaitTime)
	s.queueSvc = queueSvc
	playerSvc := player.NewService(ctx, s.pubsub, s.mgr, gameSvc, s.queueSvc)
	s.playerSvc = playerSvc
	gameSvc.SetGameResultSettler(queueSvc)
	s.pubsub.SetPlayerManager(playerSvc)
	server := grpc.NewServer()
	rpcServer := rpc.NewRpc(
		chainSvc,
		playerSvc,
	)
	s.rpcServer = rpcServer
	proto.RegisterPubSubServiceServer(server, s.pubsub)
	proto.RegisterRpcServiceServer(server, s.rpcServer)
	s.server = server
	return s, nil
}

func (s *Service) Start() error {
	err := s.chainSvc.Start()
	if err != nil {
		return err
	}
	err = s.gameSvc.Start()
	if err != nil {
		return err
	}
	err = s.queueSvc.Start()
	if err != nil {
		return err
	}
	err = s.startListener()
	if err != nil {
		return err
	}
	return nil
}

func (s *Service) Stop() {
	s.server.GracefulStop()
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
