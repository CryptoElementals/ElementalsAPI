package scanner

import (
	"context"
	"time"

	"github.com/CryptoElementals/common/internal/evmrpc"
	"github.com/CryptoElementals/common/log"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
)

const finalizedDialTimeout = 5

type FinalizedBlockSource struct {
	ctx         context.Context
	wsRPC       string
	httpRPC     string
	engine      *ScanEngine
	onReconnect func()
}

func NewFinalizedBlockSource(ctx context.Context, wsRPC, httpRPC string, engine *ScanEngine, onReconnect func()) *FinalizedBlockSource {
	return &FinalizedBlockSource{
		ctx:         ctx,
		wsRPC:       wsRPC,
		httpRPC:     httpRPC,
		engine:      engine,
		onReconnect: onReconnect,
	}
}

func (f *FinalizedBlockSource) Run() {
	firstConnect := true
	for {
		select {
		case <-f.ctx.Done():
			log.Info("FinalizedBlockSource stopped")
			return
		default:
		}

		client, err := ethclient.Dial(f.wsRPC)
		if err != nil {
			log.Errorf("FinalizedBlockSource WS dial failed: %v", err)
			time.Sleep(finalizedDialTimeout * time.Second)
			continue
		}

		if err := f.refreshFinalizedHeight(); err != nil {
			log.Errorf("FinalizedBlockSource initial finalized query failed: %v", err)
			client.Close()
			time.Sleep(finalizedDialTimeout * time.Second)
			continue
		}

		headers := make(chan *types.Header)
		sub, err := client.SubscribeNewHead(f.ctx, headers)
		if err != nil {
			log.Errorf("FinalizedBlockSource subscribe failed: %v", err)
			client.Close()
			time.Sleep(finalizedDialTimeout * time.Second)
			continue
		}

		log.Info("FinalizedBlockSource subscribed to newHeads")
		if f.onReconnect != nil && !firstConnect {
			f.onReconnect()
		}
		firstConnect = false

		disconnected := false
		for !disconnected {
			select {
			case <-f.ctx.Done():
				sub.Unsubscribe()
				client.Close()
				return
			case err := <-sub.Err():
				log.Warnf("FinalizedBlockSource subscription error: %v", err)
				sub.Unsubscribe()
				client.Close()
				disconnected = true
			case <-headers:
				if err := f.refreshFinalizedHeight(); err != nil {
					log.Errorf("FinalizedBlockSource finalized query failed: %v", err)
				}
			}
		}
		time.Sleep(finalizedDialTimeout * time.Second)
	}
}

func (f *FinalizedBlockSource) refreshFinalizedHeight() error {
	ctx, cancel := context.WithTimeout(f.ctx, finalizedDialTimeout*time.Second)
	defer cancel()
	height, err := evmrpc.GetFinalizedBlockNumber(ctx, f.httpRPC)
	if err != nil {
		return err
	}
	f.engine.SetScanUpperBound(height)
	log.Debugf("FinalizedBlockSource scanUpperBound=%d", height)
	return nil
}
