package roomserver

import (
	"context"

	"github.com/CryptoElementals/common/rpc/server"
	"github.com/CryptoElementals/common/wallet"
	"github.com/CryptoElementals/common/worker"
	"github.com/CryptoElementals/common/worker/chain"
	"github.com/CryptoElementals/common/worker/game"
	"github.com/CryptoElementals/common/worker/player"
	"github.com/CryptoElementals/common/worker/queue"
	"github.com/ethereum/go-ethereum/ethclient"
)

type Service struct {
	ctx          context.Context
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
}

func New(ctx context.Context, cfg *Config) (*Service, error) {
	s := &Service{
		ctx:          ctx,
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
	chainSvc := chain.NewService(ctx, s.mgr, chainID.Int64(), client, cfg.roomManagerContract, w, cfg.RoundTimeout, cfg.MaxRounds)
	s.chainSvc = chainSvc
	gameSvc := game.NewService(ctx, s.mgr)
	s.gameSvc = gameSvc
	return s, nil
}
