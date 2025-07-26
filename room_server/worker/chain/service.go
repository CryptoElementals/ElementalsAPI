package chain

import (
	"context"
	"sync"

	"github.com/CryptoElementals/common/cache"
	"github.com/CryptoElementals/common/log"
	"github.com/CryptoElementals/common/room_server/worker"
	"github.com/CryptoElementals/common/rpc/proto"
	"github.com/CryptoElementals/common/wallet"
	"github.com/ethereum/go-ethereum/accounts/abi/bind/v2"
)

type Service struct {
	ctx         context.Context
	chain       *Chain
	batchTxLock sync.RWMutex
}

func NewService(ctx context.Context,
	workerManager *worker.WorkerManager,
	chainID int64,
	client bind.ContractBackend,
	roomManagerContractAddress string,
	wallet *wallet.Wallet,
	dataCache cache.Cache, isDevelop ...bool) *Service {
	chain := NewChain(ctx, workerManager, chainID, client, roomManagerContractAddress, wallet, dataCache, isDevelop...)
	return &Service{ctx: ctx, chain: chain}
}

func (s *Service) SubmitTransactions(txs *proto.TransactionBatch) error {
	log.Info("receive tx batch, block number: ", txs.BlockNumber)
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
		log.Error("SubmitTransactions failed", err)
		return err
	}
	log.Info("SubmitTransactions success")
	return nil
}

func (s *Service) Start() error {
	return s.chain.Start()
}
