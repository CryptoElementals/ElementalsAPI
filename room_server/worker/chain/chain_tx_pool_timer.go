package chain

import (
	"encoding/json"
	"fmt"

	"github.com/CryptoElementals/common/timer"
)

const chainTxPoolTickEventType = "chain_tx_pool_tick"

// chainTxPoolTickEvent is enqueued on timer.ScopeRoom to flush all chain tx pools on an interval.
// One run loads all pending rows, partitions by chain_id, and submits per chain.
type chainTxPoolTickEvent struct{}

func (e *chainTxPoolTickEvent) EventType() string { return chainTxPoolTickEventType }

func (e *chainTxPoolTickEvent) Marshal() []byte {
	b, _ := json.Marshal(e)
	return b
}

func (e *chainTxPoolTickEvent) Unmarshal(data []byte) error {
	return json.Unmarshal(data, e)
}

func (e *chainTxPoolTickEvent) String() string { return "chain_tx_pool_tick{}" }

// registerTxPoolTimerHandler is called from Start; handler closes over the Chain.
func (h *Chain) registerTxPoolTimerHandler() error {
	return timer.RegisterHandler(timer.ScopeRoom, &chainTxPoolTickEvent{}, func(evt timer.TimerEvent) error {
		_, ok := evt.(*chainTxPoolTickEvent)
		if !ok {
			return fmt.Errorf("chain tx pool tick: bad event type %T", evt)
		}
		return h.onTxPoolTimerTick()
	})
}

// registerChainTxPoolPeriodic registers a shared Asynq cron that periodically enqueues the tick
// to the room timer queue.
func (h *Chain) registerChainTxPoolPeriodic(evt *chainTxPoolTickEvent) error {
	return timer.RegisterRoomChainTxPoolRecurring(h.poolTickerDur, evt)
}

func (h *Chain) onTxPoolTimerTick() error {
	if h.ctx.Err() != nil {
		return nil
	}
	h.runAllPoolTicks()
	return nil
}
