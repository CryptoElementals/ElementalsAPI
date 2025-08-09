package botserver

import (
	"context"
	"sync"

	rpc "github.com/CryptoElementals/common/rpc/client"
	"github.com/CryptoElementals/common/rpc/proto"
	"github.com/CryptoElementals/common/wallet"
	"github.com/ethereum/go-ethereum/ethclient"
)

type Service struct {
	ctx         context.Context
	ccl         context.CancelFunc
	bots        chan *Bot
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
	bots := make(chan *Bot, len(walletPaths))
	for _, path := range walletPaths {
		p, err := parseWallet(path)
		if err != nil {
			return nil, err
		}
		b := NewBot(ctx, p, rpcClient, chainClient, chainID, newGameChan)
		bots <- b
	}
	return &Service{
		ctx:         ctx,
		ccl:         ccl,
		chainClient: chainClient,
		rpcClient:   rpcClient,
		newGameChan: newGameChan,
	}, nil
}

func (s *Service) Start() {
}

func (s *Service) Stop() {
	s.ccl()
	s.wg.Wait()
}

func (s *Service) AddBot() *proto.PlayerAddress {
	b := <-s.bots
	s.wg.Add(1)
	go func() {
		defer s.wg.Done()
		b.runGameLoop()
		s.bots <- b
	}()
	return b.addr.ToProto()
}
