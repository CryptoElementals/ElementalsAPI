package botserver

import (
	"context"
	"sync"

	"github.com/CryptoElementals/common/log"
	"github.com/CryptoElementals/common/room_server/worker/types"
	rpc "github.com/CryptoElementals/common/rpc/client"
	"github.com/CryptoElementals/common/wallet"
	"github.com/ethereum/go-ethereum/ethclient"
)

type Service struct {
	ctx         context.Context
	ccl         context.CancelFunc
	bots        []*Bot
	addresses   []*types.PlayerAddress
	chainClient *ethclient.Client
	rpcClient   *rpc.Client
	newGameChan chan struct{}
	wg          sync.WaitGroup
}

func parseWallet(walletPath string) (*playerWallet, error) {
	tempWallet, err := wallet.LoadWallet(walletPath)
	if err != nil {
		return nil, err
	}
	accountWallet, err := wallet.NewWallet("")
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
	walletPaths []string,
	chainEndpoint string,
	roomServerEndpoint string) (*Service, error) {
	ctx, ccl := context.WithCancel(ctx)
	chainClient, err := ethclient.DialContext(ctx, chainEndpoint)
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
		b := NewBot(ctx, p, rpcClient, chainClient, chainID)
		bots = append(bots, b)
		addresses = append(addresses, p.address())
	}
	return &Service{
		ctx:         ctx,
		ccl:         ccl,
		chainClient: chainClient,
		rpcClient:   rpcClient,
		bots:        bots,
		addresses:   addresses,
	}, nil
}

func (s *Service) Start() error {
	err := s.rpcClient.RpcClient.RegisterBots(s.ctx, s.addresses)
	if err != nil {
		return err
	}
	for _, b := range s.bots {
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
					err = s.rpcClient.RpcClient.RegisterBot(s.ctx, b.addr)
					if err != nil {
						log.Errorw("register bot failed", "err", err, "addr", b.addr)
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
}
