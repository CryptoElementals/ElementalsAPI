package scanner

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

type stubPrefetcher struct {
	mu    sync.Mutex
	calls []uint64
}

func (s *stubPrefetcher) PrefetchBlock(ctx context.Context, blockNum uint64) (*BlockData, error) {
	s.mu.Lock()
	s.calls = append(s.calls, blockNum)
	s.mu.Unlock()
	return &BlockData{BlockNumber: blockNum}, nil
}

type stubHandler struct {
	mu     sync.Mutex
	blocks []uint64
}

func (s *stubHandler) HandleBlock(ctx context.Context, data *BlockData) error {
	s.mu.Lock()
	s.blocks = append(s.blocks, data.BlockNumber)
	s.mu.Unlock()
	return nil
}

func TestScanEngineDefaultStartHeight(t *testing.T) {
	e := NewScanEngine(context.Background(), nil, nil, BlockSyncKey{ChainID: 97, Type: "finalized"}, 2)
	e.setScanStartHeight(defaultBscScanStartBlock)

	e.currentScannedHeightMu.RLock()
	cur := e.currentScannedHeight
	e.currentScannedHeightMu.RUnlock()
	e.toSubmitHeightMu.RLock()
	submit := e.toSubmitHeight
	e.toSubmitHeightMu.RUnlock()

	require.Equal(t, defaultBscScanStartBlock, cur)
	require.Equal(t, defaultBscScanStartBlock, submit)
}

func TestScanEngineSetScanUpperBoundMonotonic(t *testing.T) {
	e := NewScanEngine(context.Background(), nil, nil, BlockSyncKey{ChainID: 97, Type: "finalized"}, 2)
	e.SetScanUpperBound(100)
	e.SetScanUpperBound(90)
	require.Equal(t, uint64(100), e.ScanUpperBound())
	e.SetScanUpperBound(105)
	require.Equal(t, uint64(105), e.ScanUpperBound())
}

func TestScanEngineProcessesBlocksInOrder(t *testing.T) {
	prefetcher := &stubPrefetcher{}
	handler := &stubHandler{}
	e := NewScanEngine(context.Background(), prefetcher, handler, BlockSyncKey{ChainID: 97, Type: "finalized"}, 2)
	e.SetScanUpperBound(3)
	e.currentScannedHeight = 1
	e.toSubmitHeight = 1

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	done := make(chan struct{})
	go func() {
		e.RunCatchUp(ctx)
		close(done)
	}()

	time.Sleep(300 * time.Millisecond)
	cancel()
	<-done

	handler.mu.Lock()
	defer handler.mu.Unlock()
	require.Equal(t, []uint64{1, 2, 3}, handler.blocks)
}
