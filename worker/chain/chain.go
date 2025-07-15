package chain

import (
	"context"

	"github.com/CryptoElementals/common/worker"
	"github.com/CryptoElementals/common/worker/types"
)

type Chain struct {
	ctx           context.Context
	workerManager *worker.WorkerManager
}

func (c *Chain) Handle(ctx context.Context, event *types.Event) error {
	return nil
}

func (c *Chain) createRoomContract(players []types.PlayerAddress) error {
	return nil
}

func (c *Chain) setRoomReady(roomContract string) error {
	return nil
}

func (c *Chain) createSelf() {
	c.workerManager.RegisterWorkerFactory(types.WORKER_TYPE_CHAIN, func(id string, t worker.WorkerType) *worker.Worker {
		return worker.NewWorker(c.ctx, id, types.WORKER_TYPE_CHAIN)
	})
	c.workerManager.SpwanWorker(types.CHAIN_MANAGER_ID, types.WORKER_TYPE_CHAIN, c)
}
