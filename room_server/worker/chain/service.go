package chain

import (
	"context"

	"github.com/CryptoElementals/common/cache"
	"github.com/CryptoElementals/common/log"
	"github.com/CryptoElementals/common/room_server/worker"
	"github.com/CryptoElementals/common/room_server/worker/types"
	"github.com/CryptoElementals/common/rpc/proto"
	"github.com/CryptoElementals/common/wallet"
	"github.com/ethereum/go-ethereum/accounts/abi/bind/v2"
)

type Service struct {
	ctx   context.Context
	chain *Chain
}

func NewService(ctx context.Context,
	workerManager *worker.WorkerManager,
	chainID int64,
	client bind.ContractBackend,
	roomManagerContractAddress string,
	wallets []*wallet.Wallet,
	dataCache cache.Cache, isDevelop ...bool) (*Service, error) {
	chain, err := NewChain(ctx, workerManager, chainID, client, roomManagerContractAddress, wallets, dataCache, isDevelop...)
	if err != nil {
		return nil, err
	}
	return &Service{ctx: ctx, chain: chain}, nil
}

func (s *Service) SubmitTransactions(txs *proto.TransactionBatch) error {
	log.Info("receive tx batch, block number: ", txs.BlockNumber)
	evt := &batchTxEvent{
		txs:       txs,
		blockNum:  txs.BlockNumber,
		blockHash: txs.BlockHash,
	}
	s.chain.batchSendTxs(evt)
	log.Info("SubmitTransactions done")
	return nil
}

func (s *Service) CreateRoomContract(evt *types.RequireContractCreationEvent) error {
	return s.chain.CreateRoomContract(evt)
}

func (s *Service) SetRoundReady(evt *types.RequireSetupNewRoundEvent) error {
	return s.chain.SetRoundReady(evt)
}

func (s *Service) SetTurnReady(evt *types.RequireSetupNewTurnEvent) error {
	return s.chain.SetTurnReady(evt)
}

func (s *Service) Start() error {
	return s.chain.Start()
}
