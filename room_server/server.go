package roomserver

import (
	"context"
	"fmt"
	"log"
	"net"

	"github.com/CryptoElementals/common/cache"
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
	cfg       *Config
	mgr       *worker.WorkerManager
	server    *grpc.Server
	pubsub    *rpc.PubSub
	rpcServer *rpc.Rpc
	chainSvc  *chain.Service
	gameSvc   *game.Service
	playerSvc *player.Service
	queueSvc  *queue.Service
}

type Config struct {
	ChainID             int64
	ChainRpc            string
	RoomManagerContract string
	WalletPath          string

	RoundTimeout  int64
	MaxRounds     int64
	GameInitialHP int64

	GrpcServerPort int64
	isDevelop      bool
}

func New(ctx context.Context, cfg *Config) (*Service, error) {
	s := &Service{
		ctx:    ctx,
		cfg:    cfg,
		mgr:    worker.NewWorkerManager(ctx),
		pubsub: rpc.NewPubSub(),
	}
	client, err := ethclient.DialContext(ctx, cfg.ChainRpc)
	if err != nil {
		return nil, err
	}
	chainID, err := client.ChainID(ctx)
	if err != nil {
		return nil, err
	}
	w, err := wallet.LoadWallet(cfg.WalletPath)
	if err != nil {
		return nil, err
	}
	var c cache.Cache
	if cfg.isDevelop {
		c = cache.NewMemCache()
	} else {
		c, err = cache.NewRedisCache()
		if err != nil {
			return nil, err
		}
	}

	chainSvc := chain.NewService(ctx, s.mgr, chainID.Int64(), client, cfg.RoomManagerContract, w, c)
	s.chainSvc = chainSvc
	gameSvc := game.NewService(ctx, s.mgr, cfg.GameInitialHP, cfg.RoundTimeout, cfg.MaxRounds)
	s.gameSvc = gameSvc
	playerSvc := player.NewService(ctx, s.pubsub, s.mgr, gameSvc, s.queueSvc)
	s.playerSvc = playerSvc
	queueSvc := queue.NewService(ctx, s.mgr, c)
	s.queueSvc = queueSvc
	server := grpc.NewServer()
	rpcServer := rpc.NewRpc(
		gameSvc,
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

func (s *Service) startListener() error {
	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", s.cfg.GrpcServerPort))
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
