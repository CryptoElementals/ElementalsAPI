package chain

import (
	"context"

	"github.com/CryptoElementals/common/rpc/proto"
	"github.com/CryptoElementals/common/wallet"
	"github.com/CryptoElementals/common/worker"
	"github.com/ethereum/go-ethereum/accounts/abi/bind/v2"
)

type Service struct {
	ctx   context.Context
	chain *Chain
}

func NewService(ctx context.Context,
	workerManager *worker.WorkerManager, chainID int64, client bind.ContractBackend,
	roomManagerContractAddress string, wallet *wallet.Wallet,
	roundTimeout int64, maxRounds int64) *Service {
	chain := NewChain(ctx, workerManager, chainID, client, roomManagerContractAddress, wallet, roundTimeout, maxRounds)
	chain.createSelf()
	return &Service{ctx: ctx, chain: chain}
}

func (s *Service) ReceiveTransactions(blockNum uint64, txs *proto.TransactionBatch) error {
	done := make(chan struct{})
	errChan := make(chan error, 1)
	evt := &batchTxEvent{
		txs:       txs,
		blockNum:  blockNum,
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
