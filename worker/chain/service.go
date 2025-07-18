package chain

import (
	"context"

	"github.com/CryptoElementals/common/rpc/proto"
	"github.com/CryptoElementals/common/worker"
	"github.com/ethereum/go-ethereum/ethclient"
)

type Service struct {
	ctx   context.Context
	chain *Chain
}

func NewService(ctx context.Context, workerManager *worker.WorkerManager, client *ethclient.Client,
	roomManagerContractAddress string, roundTimeout int64, maxRounds int64) *Service {
	chain := NewChain(ctx, workerManager, client, roomManagerContractAddress, roundTimeout, maxRounds)
	chain.createSelf()
	return &Service{ctx: ctx, chain: chain}
}

func (s *Service) ReceiveTransactions(blockNum uint64, blockHash []byte, txs *proto.TransactionBatch) error {
	done := make(chan struct{})
	errChan := make(chan error, 1)
	evt := &batchTxEvent{
		txs:       txs,
		blockNum:  blockNum,
		blockHash: blockHash,
		done:      done,
	}
	s.chain.batchSendTxs(evt)
	<-done
	if err := <-errChan; err != nil {
		return err
	}
	return nil
}
