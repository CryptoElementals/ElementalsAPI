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
	roomManagerContractAddress string, // This parameter is kept for backward compatibility but not used
	roomV2ContractAddress string,
	wallets []*wallet.Wallet,
	dataCache cache.Cache, isDevelop ...bool) (*Service, error) {
	chain, err := NewChain(ctx, workerManager, chainID, client, roomV2ContractAddress, wallets, dataCache, isDevelop...)
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

func (s *Service) SetTurnReady(evt *types.RequireSetupNewTurnEvent) error {
	return s.chain.SetTurnReady(evt)
}

// SubmitPlayerCommitmentsBatch submits multiple player commitments in a batch
func (s *Service) SubmitPlayerCommitmentsBatch(events []*types.SubmitPlayerCommitment) error {
	return s.chain.submitPlayerCommitmentsBatch(events)
}

// SubmitPlayerCardsBatch submits multiple player cards in a batch
func (s *Service) SubmitPlayerCardsBatch(events []*types.SubmitPlayerCard) error {
	return s.chain.submitPlayerCardsBatch(events)
}

func (s *Service) Start() error {
	return s.chain.Start()
}
