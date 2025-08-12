package botserver

import (
	"context"
	"sync"

	"github.com/CryptoElementals/common/config"
	"github.com/CryptoElementals/common/log"
	"github.com/CryptoElementals/common/room_server/worker/types"
	rpc "github.com/CryptoElementals/common/rpc/client"
	"github.com/CryptoElementals/common/wallet"
	"github.com/ethereum/go-ethereum/ethclient"
)

type Service struct {
	ctx          context.Context
	ccl          context.CancelFunc
	mimicPlayers bool
	bots         []*Bot
	addresses    []*types.PlayerAddress
	chainClient  *ethclient.Client
	rpcClient    *rpc.Client
	wg           sync.WaitGroup
}

func parseWallet(path config.WalletPath) (*playerWallet, error) {
	accountWallet, err := wallet.NewWallet(path.AccountWallet)
	if err != nil {
		return nil, err
	}
	tempWallet, err := wallet.LoadWallet(path.TemporaryWallet)
	if err != nil {
		return nil, err
	}
	return &playerWallet{
		tempWallet:    tempWallet,
		accountWallet: accountWallet,
	}, nil
}

func NewService(
	ctx context.Context,
	walletPaths []config.WalletPath,
	chainEndpoint string,
	roomServerEndpoint string,
	mimicPlayers bool,
) (*Service, error) {
	ctx, ccl := context.WithCancel(ctx)
	chainClient, err := ethclient.Dial(chainEndpoint)
	if err != nil {
		return nil, err
	}
	chainID, err := chainClient.ChainID(ctx)
	if err != nil {
		return nil, err
	}
	rpcClient, err := rpc.NewClient(roomServerEndpoint)
	if err != nil {
		return nil, err
	}
	bots := make([]*Bot, 0, len(walletPaths))
	addresses := make([]*types.PlayerAddress, 0, len(walletPaths))
	for _, path := range walletPaths {
		p, err := parseWallet(path)
		if err != nil {
			return nil, err
		}
		b := NewBot(ctx, p, rpcClient, chainClient, chainID, mimicPlayers)
		bots = append(bots, b)
		addresses = append(addresses, p.address())
	}
	return &Service{
		ctx:          ctx,
		ccl:          ccl,
		mimicPlayers: mimicPlayers,
		chainClient:  chainClient,
		rpcClient:    rpcClient,
		bots:         bots,
		addresses:    addresses,
	}, nil
}

func (s *Service) Start() error {
	if s.mimicPlayers {
		return s.runPlayers()
	} else {
		return s.runBots()
	}
}

func (s *Service) runPlayers() error {
	log.Infow("run players", types.ToJsonLoggable(s.addresses))
	for _, b := range s.bots {
		err := b.client.PubSubClient.Subscribe(b.addr.String(), b.formatBotID(), b.chanEvt, b.chanErr)
		if err != nil {
			return err
		}
		s.wg.Add(1)
		go func() {
			defer s.wg.Done()
			defer b.client.PubSubClient.Unsubscribe(b.addr.String(), b.formatBotID())
			for {
				select {
				case <-s.ctx.Done():
					return
				default:
					err := b.runGameLoop()
					if err != nil {
						log.Errorw("run bot failed", "err", err, "addr", b.addr)
						return
					}
				}
			}

		}()
	}
	return nil
}

func (s *Service) runBots() error {
	log.Infow("run bots", types.ToJsonLoggable(s.addresses))
	for _, b := range s.bots {
		err := b.client.PubSubClient.Subscribe(b.addr.String(), b.formatBotID(), b.chanEvt, b.chanErr)
		if err != nil {
			return err
		}
		s.wg.Add(1)
		go func() {
			defer s.wg.Done()
			for {
				select {
				case <-s.ctx.Done():
					return
				default:
					err := b.runGameLoop()
					if err != nil {
						log.Errorw("start bot failed", "err", err, "addr", b.addr)
						return
					}
				}
			}

		}()
	}
	return nil
}

func (s *Service) Stop() {
	s.ccl()
	s.wg.Wait()
	// unregister all bots anyway
	log.Infow("unregister bots", types.ToJsonLoggable(s.addresses))
	err := s.rpcClient.RpcClient.UnregisterBots(context.Background(), s.addresses)
	if err != nil {
		log.Errorw("cannot unregister bots", "err", err)
	}
}
