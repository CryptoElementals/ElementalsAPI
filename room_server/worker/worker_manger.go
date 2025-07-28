package worker

import (
	"context"
	"sync"

	"github.com/CryptoElementals/common/room_server/worker/types"
)

type WorkerManager struct {
	ctx     context.Context
	lock    sync.RWMutex
	workers map[string]*Worker
}

func NewWorkerManager(ctx context.Context) *WorkerManager {
	return &WorkerManager{
		ctx:     ctx,
		workers: make(map[string]*Worker),
	}
}

func (w *WorkerManager) SpwanWorker(ctx context.Context, id string, t WorkerType, handler EventHandler) {
	w.lock.Lock()
	defer w.lock.Unlock()
	worker := NewWorker(ctx, id, t, w, w, handler)
	w.workers[id] = worker
	go worker.Run()
}

func (w *WorkerManager) SendEvent(to string, event *types.Event) {
	w.lock.RLock()
	defer w.lock.RUnlock()
	if worker := w.workers[to]; worker != nil {
		worker.msgQueue <- event
		return
	}
	// we might not find the worker, in this case, we should send an ack event
	// for not blocking the sender
	if event.NeedAck {
		w.SendEvent(event.Sender, &types.Event{
			Sender:  types.WORKER_MANAGER_ID,
			EventID: event.EventID,
			NeedAck: false,
			Data:    &types.AckEvent{EventID: event.EventID},
		})
	}
}

func (w *WorkerManager) CloseWorker(id string) {
	w.lock.Lock()
	defer w.lock.Unlock()
	if worker := w.workers[id]; worker != nil {
		worker.ccl()
		delete(w.workers, id)
	}
}
