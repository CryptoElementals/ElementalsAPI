package scanner

import (
	"context"
	"fmt"
	"strconv"

	"github.com/CryptoElementals/common/internal/evmrpc"
)

// BlockData is prefetched block content for ordered processing.
type BlockData struct {
	BlockNumber uint64
	BlockHash   string
	Timestamp   uint64
	Logs        []evmrpc.ReceiptLog
	Txs         []evmrpc.Tx
}

type BlockPrefetcher struct {
	httpRPC string
}

func NewBlockPrefetcher(httpRPC string) *BlockPrefetcher {
	return &BlockPrefetcher{httpRPC: httpRPC}
}

func (p *BlockPrefetcher) PrefetchBlock(ctx context.Context, blockNum uint64) (*BlockData, error) {
	block, err := evmrpc.GetBlockByNumber(ctx, p.httpRPC, blockNum, true)
	if err != nil {
		return nil, err
	}
	logs, err := evmrpc.GetBlockReceiptLogs(ctx, p.httpRPC, blockNum)
	if err != nil {
		return nil, err
	}
	txs, err := evmrpc.ParseTransactions(block.Transactions)
	if err != nil {
		return nil, err
	}
	ts, _ := strconv.ParseUint(block.Timestamp, 0, 64)
	bn, err := strconv.ParseUint(block.Number, 0, 64)
	if err != nil {
		return nil, fmt.Errorf("parse block number %q: %w", block.Number, err)
	}
	return &BlockData{
		BlockNumber: bn,
		BlockHash:   block.Hash,
		Timestamp:   ts,
		Logs:        logs,
		Txs:         txs,
	}, nil
}
