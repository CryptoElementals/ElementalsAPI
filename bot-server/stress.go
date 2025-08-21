package botserver

import (
	"context"
	"math/big"

	"github.com/CryptoElementals/common/config"
	"github.com/CryptoElementals/common/db"
	gameclient "github.com/CryptoElementals/common/game_client"
	"github.com/CryptoElementals/common/log"
	"github.com/CryptoElementals/common/wallet"
	"github.com/ethereum/go-ethereum/ethclient"
)

type StressService struct {
	ctx           context.Context
	ccl           context.CancelFunc
	chainClient   *ethclient.Client
	chainID       *big.Int
	Endpoint      string
	ChainEndpoint string
	PlayerWallets []PlayerWallet
}

func NewStressService(ctx context.Context, cfg *config.BotConfig) *StressService {
	ctx, ccl := context.WithCancel(ctx)
	playerWallets := make([]PlayerWallet, len(cfg.WalletPaths))
	for i, path := range cfg.WalletPaths {
		accountWallet, err := wallet.LoadWallet(path.AccountWallet)
		if err != nil {
			log.Errorw("Failed to load account wallet", "err", err, "path", path.AccountWallet)
			continue
		}
		tempWallet, err := wallet.LoadWallet(path.TemporaryWallet)
		if err != nil {
			log.Errorw("Failed to load temporary wallet", "err", err, "path", path.TemporaryWallet)
			continue
		}
		playerWallets[i] = PlayerWallet{
			accountWallet: accountWallet,
			tempWallet:    tempWallet,
		}
	}
	return &StressService{
		ctx:           ctx,
		ccl:           ccl,
		PlayerWallets: playerWallets,
		Endpoint:      cfg.RoomServerEndpoint,
		ChainEndpoint: cfg.ChainCfg.HttpRpc,
	}
}

func (s *StressService) Start() error {
	log.Infow("Starting stress test")
	chainClient, err := ethclient.DialContext(s.ctx, s.ChainEndpoint)
	if err != nil {
		log.Errorw("Failed to connect to chain client", "err", err)
		return err
	}
	chainId, err := s.chainClient.ChainID(s.ctx)
	if err != nil {
		return err
	}
	s.chainID = chainId
	s.chainClient = chainClient
	for i := range s.PlayerWallets {
		if err := s.runSingleGameContext(&s.PlayerWallets[i]); err != nil {
			log.Errorw("Failed to run single game context", "err", err)
			return err
		}
	}
	return nil
}

func (s *StressService) Stop() error {
	s.ccl()
	return nil
}

func (s *StressService) runSingleGameContext(playerWallet *PlayerWallet) error {
	err := db.CreateOrCheckBot(playerWallet.accountWallet.GetAddrHex(), true)
	if err != nil {
		log.Errorw("Failed to create or check bot in database", "err", err)
		return err
	}
	log.Infow("Running game context for bot", "address", playerWallet.accountWallet.GetAddrHex())
	httpClient := gameclient.NewHttpClient(
		s.ctx,
		s.Endpoint,
		playerWallet.accountWallet, // Assuming wallet is not needed for stress test
	)
	err = httpClient.Start()
	if err != nil {
		log.Errorw("Failed to start HTTP client", "err", err)
		return err
	}
	gc := gameclient.WrapHttpClient(httpClient)
	bot := NewBot(s.ctx, playerWallet, gc, s.chainClient, s.chainID)
	err = bot.runStress()
	if err != nil {
		log.Errorw("Failed to run stress test for bot", "err", err, "address", playerWallet.accountWallet.GetAddrHex())
		return err
	}
	return nil
}
