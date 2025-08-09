package botserver

import (
	"context"
	"sync"

	rpc "github.com/CryptoElementals/common/rpc/client"
	"github.com/CryptoElementals/common/wallet"
	"github.com/ethereum/go-ethereum/ethclient"
)

type Service struct {
	ctx         context.Context
	ccl         context.CancelFunc
	bots        []*Bot
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
	newGameChan := make(chan struct{})
	bots := make([]*Bot, 0, len(walletPaths))
	for _, path := range walletPaths {
		p, err := parseWallet(path)
		if err != nil {
			return nil, err
		}
		b := NewBot(ctx, p, rpcClient, chainClient, chainID, newGameChan)
		bots = append(bots, b)
	}
	return &Service{
		ctx:         ctx,
		ccl:         ccl,
		bots:        bots,
		chainClient: chainClient,
		rpcClient:   rpcClient,
		newGameChan: newGameChan,
	}, nil
}

func (s *Service) Start() {
	s.wg.Add(len(s.bots))
	for _, b := range s.bots {
		go func() {
			defer s.wg.Done()
			b.run()
		}()
	}
}

func (s *Service) Stop() {
	s.ccl()
	s.wg.Wait()
}

func (s *Service) DispatchNewBot() {
	s.newGameChan <- struct{}{}
}
