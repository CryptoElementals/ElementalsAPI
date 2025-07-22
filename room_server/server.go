package roomserver

import (
	"context"

	"github.com/CryptoElementals/common/cache"
	"github.com/CryptoElementals/common/room_server/worker"
	"github.com/CryptoElementals/common/room_server/worker/chain"
	"github.com/CryptoElementals/common/room_server/worker/game"
	"github.com/CryptoElementals/common/room_server/worker/player"
	"github.com/CryptoElementals/common/room_server/worker/queue"
	"github.com/CryptoElementals/common/rpc/server"
	"github.com/CryptoElementals/common/wallet"
	"github.com/ethereum/go-ethereum/ethclient"
)

type Service struct {
	ctx          context.Context
	cfg          *Config
	mgr          *worker.WorkerManager
	pubsubServer *server.PubSubServer
	chainSvc     *chain.Service
	gameSvc      *game.Service
	playerSvc    *player.Service
	queueSvc     *queue.Service
}

type Config struct {
	ChainID             int64
	ChainRpc            string
	roomManagerContract string
	WalletPath          string
	RoundTimeout        int64
	MaxRounds           int64
	PubSubServerPort    int64
	isDevelop           bool
}

func New(ctx context.Context, cfg *Config) (*Service, error) {
	s := &Service{
		ctx:          ctx,
		cfg:          cfg,
		mgr:          worker.NewWorkerManager(ctx),
		pubsubServer: server.NewPubSubServer(),
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
	chainSvc := chain.NewService(ctx, s.mgr, chainID.Int64(), client, cfg.roomManagerContract, w, cfg.RoundTimeout, cfg.MaxRounds, c)
	s.chainSvc = chainSvc
	gameSvc := game.NewService(ctx, s.mgr)
	s.gameSvc = gameSvc
	playerSvc := player.NewService(ctx, s.pubsubServer, s.mgr, gameSvc, s.queueSvc)
	s.playerSvc = playerSvc
	queueSvc := queue.NewService(ctx, s.mgr, c)
	s.queueSvc = queueSvc
	return s, nil
}

func (s *Service) Start() error {
	err := s.pubsubServer.Run(int(s.cfg.PubSubServerPort))
	if err != nil {
		return err
	}
	err = s.chainSvc.Start()
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
	return nil
}
