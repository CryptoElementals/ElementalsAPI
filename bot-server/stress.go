package botserver

import (
	"context"

	"github.com/CryptoElementals/common/db"
	gameclient "github.com/CryptoElementals/common/game_client"
	"github.com/CryptoElementals/common/log"
	"github.com/ethereum/go-ethereum/ethclient"
)

type StressContextConfig struct {
	Endpoint      string         `json:"endpoint"`
	ChainEndpoint string         `json:"chain_endpoint"`
	PlayerWallets []PlayerWallet `json:"player_wallets"`
}

type StressContext struct {
	ctx         context.Context
	cfg         *StressContextConfig
	chainClient *ethclient.Client
}

func NewStressContext(ctx context.Context, cfg *StressContextConfig) *StressContext {
	return &StressContext{
		ctx: ctx,
		cfg: cfg,
	}
}

func (s *StressContext) Run() error {
	log.Infow("Starting stress test")
	chainClient, err := ethclient.DialContext(s.ctx, s.cfg.ChainEndpoint)
	if err != nil {
		log.Errorw("Failed to connect to chain client", "err", err)
		return err
	}
	s.chainClient = chainClient
	for i := range s.cfg.PlayerWallets {
		if err := s.runSingleGameContext(&s.cfg.PlayerWallets[i]); err != nil {
			log.Errorw("Failed to run single game context", "err", err)
			return err
		}
	}
	return nil
}

func (s *StressContext) runSingleGameContext(playerWallet *PlayerWallet) error {
	err := db.CreateOrCheckBot(playerWallet.accountWallet.GetAddrHex(), true)
	if err != nil {
		log.Errorw("Failed to create or check bot in database", "err", err)
		return err
	}
	log.Infow("Running game context for bot", "address", playerWallet.accountWallet.GetAddrHex())
	httpClient := gameclient.NewHttpClient(
		s.ctx,
		s.cfg.Endpoint,
		playerWallet.accountWallet, // Assuming wallet is not needed for stress test
	)
	gameContext, err := gameclient.NewGameContext(
		s.ctx,
		&gameclient.GameContextConfig{
			Wallet:          playerWallet.accountWallet,
			TemporaryWallet: playerWallet.tempWallet,
			HttpClient:      httpClient,
			ChainClient:     s.chainClient,
		},
	)
	if err != nil {
		log.Errorw("Failed to create game context", "err", err)
		return err
	}
	return gameContext.Run()
}
