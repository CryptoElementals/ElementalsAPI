package worker

import (
	"context"
	"time"

	"github.com/CryptoElementals/common/log"
)

type poolTicker struct {
	ctx    context.Context
	chain  *Chain
	period time.Duration
	done   chan struct{}
}

func newPoolTicker(ctx context.Context, chain *Chain, period time.Duration) *poolTicker {
	return &poolTicker{
		ctx:    ctx,
		chain:  chain,
		period: period,
		done:   make(chan struct{}),
	}
}

func (t *poolTicker) start() {
	go t.run()
}

func (t *poolTicker) stop() {
	select {
	case <-t.done:
	default:
		close(t.done)
	}
}

func (t *poolTicker) run() {
	ticker := time.NewTicker(t.period)
	defer ticker.Stop()
	for {
		select {
		case <-t.ctx.Done():
			return
		case <-t.done:
			return
		case <-ticker.C:
			if t.ctx.Err() != nil {
				return
			}
			log.Debugw("chain tx pool ticker tick")
			t.chain.runAllPoolTicks()
		}
	}
}
