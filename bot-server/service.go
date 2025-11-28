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

func parseWallet(path config.WalletInfo) (*playerWallet, error) {
	tempWallet, err := wallet.LoadWallet(path.TemporaryWallet)
	if err != nil {
		return nil, err
	}

	return &playerWallet{
		playerId:   path.PlayerId,
		tempWallet: tempWallet,
	}, nil
}

func NewService(
	ctx context.Context,
	walletInfos []config.WalletInfo,
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
	bots := make([]*Bot, 0, len(walletInfos))
	addresses := make([]*types.PlayerAddress, 0, len(walletInfos))
	for _, walletInfo := range walletInfos {
		p, err := parseWallet(walletInfo)
		if err != nil {
			return nil, err
		}
		b := NewBot(ctx, p, rpcClient, chainClient, chainID)
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
	s.runBots()
	return nil
}

func (s *Service) runBots() {
	log.Infow("run bots", types.ToJsonLoggable(s.addresses))
	for _, b := range s.bots {
		s.wg.Add(1)
		go func() {
			defer s.wg.Done()
			err := b.run()
			if err != nil {
				log.Errorw("cannot run bot", "err", err, "addr", b.addr.String())
			}
		}()
	}
}

func (s *Service) Stop() {
	s.ccl()
	s.wg.Wait()
}
