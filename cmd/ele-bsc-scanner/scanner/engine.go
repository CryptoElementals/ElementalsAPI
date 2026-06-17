package scanner

import (
	"context"
	"sync"
	"time"

	"github.com/CryptoElementals/common/db"
	"github.com/CryptoElementals/common/log"
	dao "github.com/CryptoElementals/common/models"
)

const (
	defaultBscWorkers      = 20
	engineBlockQueueMax    = 50
	engineBlockQueueCap    = 200
	engineSubmitChanCap    = 100
	engineCheckpointEvery  = 10
	engineRetryDelay       = 3 * time.Second
	engineDistributorWait  = 200 * time.Millisecond
	engineQueueBackoff     = 100 * time.Millisecond
)

type BlockSyncKey struct {
	ChainID uint64
	Type    string
}

type orderedBlockData struct {
	blockNumber uint64
	data        *BlockData
	done        chan error
}

type blockPrefetcher interface {
	PrefetchBlock(ctx context.Context, blockNum uint64) (*BlockData, error)
}

type blockHandler interface {
	HandleBlock(ctx context.Context, data *BlockData) error
}

type ScanEngine struct {
	ctx    context.Context
	cancel context.CancelFunc

	prefetcher blockPrefetcher
	handler    blockHandler
	syncKey    BlockSyncKey
	maxWorkers int

	scanUpperBound         uint64
	scanUpperBoundMutex    sync.RWMutex
	currentScannedHeight   uint64
	currentScannedHeightMu sync.RWMutex
	toSubmitHeight         uint64
	toSubmitHeightMu       sync.RWMutex

	catchupWg sync.WaitGroup
}

func NewScanEngine(parent context.Context, prefetcher blockPrefetcher, handler blockHandler, syncKey BlockSyncKey, maxWorkers int) *ScanEngine {
	if maxWorkers <= 0 {
		maxWorkers = defaultBscWorkers
	}
	ctx, cancel := context.WithCancel(parent)
	return &ScanEngine{
		ctx:        ctx,
		cancel:     cancel,
		prefetcher: prefetcher,
		handler:    handler,
		syncKey:    syncKey,
		maxWorkers: maxWorkers,
	}
}

func (e *ScanEngine) Stop() {
	if e.cancel != nil {
		e.cancel()
	}
}

func (e *ScanEngine) InitFromDB() error {
	sync, err := db.FindBlockSync(e.syncKey.ChainID, e.syncKey.Type)
	if err != nil {
		return err
	}
	if sync != nil {
		e.SetScanUpperBound(sync.BlockHeight)
		e.setScanStartHeight(sync.BlockHeight + 1)
		log.Infof("ScanEngine loaded checkpoint chainID=%d type=%s height=%d next=%d",
			e.syncKey.ChainID, e.syncKey.Type, sync.BlockHeight, sync.BlockHeight+1)
	} else {
		e.setScanStartHeight(defaultBscScanStartBlock)
		log.Infof("ScanEngine no checkpoint, start from default block %d chainID=%d type=%s",
			defaultBscScanStartBlock, e.syncKey.ChainID, e.syncKey.Type)
	}
	return nil
}

func (e *ScanEngine) setScanStartHeight(height uint64) {
	e.currentScannedHeightMu.Lock()
	e.currentScannedHeight = height
	e.currentScannedHeightMu.Unlock()
	e.toSubmitHeightMu.Lock()
	e.toSubmitHeight = height
	e.toSubmitHeightMu.Unlock()
}

func (e *ScanEngine) SetScanUpperBound(height uint64) {
	e.scanUpperBoundMutex.Lock()
	defer e.scanUpperBoundMutex.Unlock()
	if height > e.scanUpperBound {
		e.scanUpperBound = height
	}
}

func (e *ScanEngine) ScanUpperBound() uint64 {
	e.scanUpperBoundMutex.RLock()
	defer e.scanUpperBoundMutex.RUnlock()
	return e.scanUpperBound
}

func (e *ScanEngine) RunCatchUp(catchupCtx context.Context) {
	e.catchupWg.Add(1)
	defer e.catchupWg.Done()
	e.catchUpChain(catchupCtx)
}

func (e *ScanEngine) WaitCatchUpExit() {
	e.catchupWg.Wait()
}

func (e *ScanEngine) AlignScannedHeightToSubmit() {
	e.toSubmitHeightMu.RLock()
	submitH := e.toSubmitHeight
	e.toSubmitHeightMu.RUnlock()

	e.currentScannedHeightMu.Lock()
	prev := e.currentScannedHeight
	if prev != submitH {
		e.currentScannedHeight = submitH
		log.Infof("ScanEngine aligned currentScannedHeight %d -> %d", prev, submitH)
	}
	e.currentScannedHeightMu.Unlock()
}

func (e *ScanEngine) catchUpChain(ctx context.Context) {
	submitChan := make(chan *orderedBlockData, engineSubmitChanCap)
	blockQueue := make(chan uint64, engineBlockQueueCap)

	var wg sync.WaitGroup

	log.Infof("ScanEngine catchUp started: currentScannedHeight=%d toSubmitHeight=%d scanUpperBound=%d",
		e.currentScannedHeight, e.toSubmitHeight, e.ScanUpperBound())

	wg.Add(1)
	go func() {
		defer wg.Done()
		for {
			select {
			case <-ctx.Done():
				return
			default:
				e.currentScannedHeightMu.RLock()
				cur := e.currentScannedHeight
				e.currentScannedHeightMu.RUnlock()
				upper := e.ScanUpperBound()
				if cur > upper {
					time.Sleep(engineDistributorWait)
					continue
				}
				if len(blockQueue) <= engineBlockQueueMax {
					e.addBlockToQueue(ctx, blockQueue)
				} else {
					time.Sleep(engineQueueBackoff)
				}
			}
		}
	}()

	for i := 0; i < e.maxWorkers; i++ {
		workerID := i
		wg.Add(1)
		go func() {
			defer wg.Done()
			for {
				select {
				case <-ctx.Done():
					return
				case blockNumber, ok := <-blockQueue:
					if !ok {
						return
					}
					data, err := e.prefetcher.PrefetchBlock(ctx, blockNumber)
					if err != nil {
						log.Warnf("worker %d prefetch block %d err: %v", workerID, blockNumber, err)
						go requeueBlock(ctx, blockQueue, blockNumber)
						time.Sleep(engineRetryDelay)
						continue
					}
					ordered := &orderedBlockData{
						blockNumber: blockNumber,
						data:        data,
						done:        make(chan error, 1),
					}
					select {
					case submitChan <- ordered:
					case <-ctx.Done():
						return
					}
					select {
					case err := <-ordered.done:
						if err != nil {
							go requeueBlock(ctx, blockQueue, blockNumber)
							time.Sleep(engineRetryDelay)
							continue
						}
					case <-ctx.Done():
						return
					}
				}
			}
		}()
	}

	wg.Add(1)
	go func() {
		defer wg.Done()
		e.orderedSubmitWorker(ctx, submitChan)
	}()

	<-ctx.Done()
	wg.Wait()
	log.Info("ScanEngine catchUp exited")
}

func (e *ScanEngine) addBlockToQueue(ctx context.Context, blockQueue chan<- uint64) {
	e.currentScannedHeightMu.Lock()
	defer e.currentScannedHeightMu.Unlock()
	current := e.currentScannedHeight
	select {
	case <-ctx.Done():
	case blockQueue <- current:
		e.currentScannedHeight = current + 1
	}
}

func (e *ScanEngine) orderedSubmitWorker(ctx context.Context, submitChan <-chan *orderedBlockData) {
	pending := make(map[uint64]*orderedBlockData)
	tick := time.NewTicker(time.Second)
	defer tick.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case batch, ok := <-submitChan:
			if !ok {
				return
			}
			pending[batch.blockNumber] = batch
		case <-tick.C:
		}

		for {
			e.toSubmitHeightMu.RLock()
			cur := e.toSubmitHeight
			e.toSubmitHeightMu.RUnlock()

			next, exists := pending[cur]
			if !exists {
				break
			}

			err := e.handler.HandleBlock(ctx, next.data)
			notified := notifyWorkerDone(next, cur, err)
			if err != nil {
				if notified {
					delete(pending, cur)
				}
				break
			}
			if !notified {
				break
			}
			delete(pending, cur)

			if cur%engineCheckpointEvery == 0 {
				if saveErr := db.SaveBlockSync(dao.BlockSync{
					ChainID:     e.syncKey.ChainID,
					Type:        e.syncKey.Type,
					BlockHeight: cur,
				}); saveErr != nil {
					log.Errorf("save block_sync %d err: %v", cur, saveErr)
				} else {
					log.Infof("checkpoint block %d chainID=%d type=%s", cur, e.syncKey.ChainID, e.syncKey.Type)
				}
			}

			e.toSubmitHeightMu.Lock()
			e.toSubmitHeight++
			e.toSubmitHeightMu.Unlock()
		}
	}
}

func notifyWorkerDone(batch *orderedBlockData, blockNum uint64, submitErr error) bool {
	select {
	case batch.done <- submitErr:
		return true
	default:
		log.Warnf("failed to notify worker for block %d (submitErr=%v)", blockNum, submitErr)
		return false
	}
}
