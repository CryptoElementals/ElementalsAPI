package chain

import (
	"context"

	"github.com/CryptoElementals/common/log"
	"github.com/CryptoElementals/common/room_server/worker"
	"github.com/CryptoElementals/common/room_server/worker/types"
	"github.com/CryptoElementals/common/rpc/proto"
	"github.com/CryptoElementals/common/wallet"
	"github.com/ethereum/go-ethereum/ethclient"
)

type Service struct {
	ctx   context.Context
	chain *Chain
}

func NewService(ctx context.Context,
	workerManager *worker.WorkerManager,
	chainID int64,
	client *ethclient.Client,
	roomV3ContractAddress string,
	wallets []*wallet.Wallet,
	isDevelop ...bool) (*Service, error) {
	chain, err := NewChain(ctx, workerManager, chainID, client, roomV3ContractAddress, wallets, isDevelop...)
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

// SubmitTasks submits a batch of pre-encoded contract tasks to the underlying chain.
func (s *Service) SubmitTasks(tasks []types.RoomContractTask) error {
	return s.chain.SubmitTasks(tasks)
}

func (s *Service) Start() error {
	return s.chain.Start()
}
