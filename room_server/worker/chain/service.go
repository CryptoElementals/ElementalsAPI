package chain

import (
	"context"

	"github.com/CryptoElementals/common/cache"
	"github.com/CryptoElementals/common/room_server/worker"
	"github.com/CryptoElementals/common/rpc/proto"
	"github.com/CryptoElementals/common/wallet"
	"github.com/ethereum/go-ethereum/accounts/abi/bind/v2"
)

type Service struct {
	ctx   context.Context
	chain *Chain
}

func NewService(ctx context.Context,
	workerManager *worker.WorkerManager, chainID int64, client bind.ContractBackend,
	roomManagerContractAddress string, wallet *wallet.Wallet,
	roundTimeout int64, maxRounds int64, dataCache cache.Cache) *Service {
	chain := NewChain(ctx, workerManager, chainID, client, roomManagerContractAddress, wallet, roundTimeout, maxRounds, dataCache)
	return &Service{ctx: ctx, chain: chain}
}

func (s *Service) SubmitTransactions(txs *proto.TransactionBatch) error {
	done := make(chan struct{})
	errChan := make(chan error, 1)
	evt := &batchTxEvent{
		txs:       txs,
		blockNum:  txs.BlockNumber,
		blockHash: txs.BlockHash,
		done:      done,
		errChan:   errChan,
	}
	s.chain.batchSendTxs(evt)
	<-done
	if err := <-errChan; err != nil {
		return err
	}
	return nil
}

func (s *Service) Start() error {
	return s.chain.Start()
}
