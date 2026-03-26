package worker

import (
	"context"
	"sync"

	"github.com/CryptoElementals/common/room_server/worker/types"
)

// WorkerManager owns all Workers; it is the EventSender/WorkerCloser passed into each Worker.
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

// SpawnWorker starts a goroutine that drains events for id until the worker context is canceled.
func (w *WorkerManager) SpawnWorker(ctx context.Context, id string, t WorkerType, handler EventHandler) {
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
	if event.AckChan != nil {
		close(event.AckChan)
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
