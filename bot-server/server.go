package botserver

import (
	"context"

	"github.com/CryptoElementals/common/config"
	"github.com/CryptoElementals/common/log"
)

type BotServer struct {
	cfg *config.BotConfig
	svc *Service
}

func NewBotServer(cfg *config.BotConfig) *BotServer {
	svc, err := NewService(context.Background(), cfg.WalletPaths, cfg.ChainCfg.HttpRpc, cfg.RoomServerEndpoint, cfg.MimicPlayers)
	if err != nil {
		log.Fatalw("cannot init bot server", "err", err)
	}
	return &BotServer{
		cfg: cfg,
		svc: svc,
	}
}

func (s *BotServer) Start() error {
	return s.svc.runBots()
}

func (s *BotServer) Stop() {
	s.svc.Stop()
}
