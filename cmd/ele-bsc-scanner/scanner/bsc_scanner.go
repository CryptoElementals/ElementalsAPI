package scanner

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/CryptoElementals/common/config"
	"github.com/CryptoElementals/common/internal/evmrpc"
	"github.com/CryptoElementals/common/log"
)

type BscScanner struct {
	ctx           context.Context
	cancel        context.CancelFunc
	chainID       uint64
	engine        *ScanEngine
	source        *FinalizedBlockSource
	registry      *WalletRegistry
	eventSink     EventSink
	catchupCancel context.CancelFunc
	catchupWg     sync.WaitGroup
}

func NewBscScanner(parent context.Context) (*BscScanner, error) {
	cfg := config.BscScannerGConf
	if cfg.ChainCfg.WalletManagerAddress == "" {
		return nil, fmt.Errorf("wallet-manager-address is required")
	}

	chainID, err := resolveChainID(parent, cfg.ChainCfg.HttpRpc)
	if err != nil {
		return nil, err
	}

	registry, err := NewWalletRegistry(chainID, cfg.ChainCfg.HttpRpc, cfg.ChainCfg.WalletManagerAddress)
	if err != nil {
		return nil, err
	}
	if err := registry.LoadFromDB(); err != nil {
		return nil, err
	}
	refreshCtx, cancelRefresh := context.WithTimeout(parent, 30*time.Second)
	defer cancelRefresh()
	if err := registry.FullRefresh(refreshCtx); err != nil {
		return nil, err
	}

	processor, err := NewTokenProcessor(chainID, registry)
	if err != nil {
		return nil, err
	}
	sink, err := NewEventSink(parent)
	if err != nil {
		return nil, err
	}
	handler := NewBlockHandler(registry, processor, sink)
	prefetcher := NewBlockPrefetcher(cfg.ChainCfg.HttpRpc)

	workers := cfg.WorkerCount
	engine := NewScanEngine(parent, prefetcher, handler, BlockSyncKey{
		ChainID: chainID,
		Type:    blockSyncTypeFinalized,
	}, workers)

	ctx, cancel := context.WithCancel(parent)
	bs := &BscScanner{
		ctx:       ctx,
		cancel:    cancel,
		chainID:   chainID,
		engine:    engine,
		registry:  registry,
		eventSink: sink,
	}
	bs.source = NewFinalizedBlockSource(ctx, cfg.ChainCfg.WsRpc, cfg.ChainCfg.HttpRpc, engine, bs.onWSReconnect)
	return bs, nil
}

func resolveChainID(parent context.Context, httpRPC string) (uint64, error) {
	for {
		select {
		case <-parent.Done():
			return 0, parent.Err()
		default:
		}
		ctx, cancel := context.WithTimeout(parent, finalizedDialTimeout*time.Second)
		chainID, err := evmrpc.GetChainID(ctx, httpRPC)
		cancel()
		if err != nil {
			log.Errorf("GetChainID from %s failed: %v, retrying...", httpRPC, err)
			time.Sleep(finalizedDialTimeout * time.Second)
			continue
		}
		log.Infof("Resolved chainID=%d from %s", chainID, httpRPC)
		return chainID, nil
	}
}

func (b *BscScanner) Stop() {
	if b.cancel != nil {
		b.cancel()
	}
	if b.catchupCancel != nil {
		b.catchupCancel()
	}
	b.engine.Stop()
	if b.eventSink != nil {
		if err := b.eventSink.Close(); err != nil {
			log.Errorf("close event sink: %v", err)
		}
	}
}

func (b *BscScanner) Run() {
	if err := b.engine.InitFromDB(); err != nil {
		log.Errorf("BscScanner InitFromDB failed: %v", err)
		return
	}
	if err := b.bootstrapFinalized(); err != nil {
		log.Errorf("BscScanner bootstrap finalized failed: %v", err)
		return
	}

	interval := config.BscScannerGConf.WalletRegistryRefreshInterval
	go b.registry.RunPeriodicRefresh(b.ctx, interval)

	b.startCatchUp()
	go b.source.Run()
}

func (b *BscScanner) bootstrapFinalized() error {
	ctx, cancel := context.WithTimeout(b.ctx, finalizedDialTimeout*time.Second)
	defer cancel()
	height, err := evmrpc.GetFinalizedBlockNumber(ctx, config.BscScannerGConf.ChainCfg.HttpRpc)
	if err != nil {
		return err
	}
	b.engine.SetScanUpperBound(height)
	log.Infof("BscScanner bootstrap finalized height=%d chainID=%d", height, b.chainID)
	return nil
}

func (b *BscScanner) startCatchUp() {
	if b.catchupCancel != nil {
		b.catchupCancel()
		b.engine.WaitCatchUpExit()
		b.engine.AlignScannedHeightToSubmit()
	}
	catchupCtx, cancel := context.WithCancel(b.ctx)
	b.catchupCancel = cancel
	b.catchupWg.Add(1)
	go func() {
		defer b.catchupWg.Done()
		b.engine.RunCatchUp(catchupCtx)
	}()
}

func (b *BscScanner) onWSReconnect() {
	log.Info("BscScanner WS reconnected, restarting catch-up")
	b.startCatchUp()
}
