package worker

import (
	"context"
	"errors"

	"github.com/CryptoElementals/common/worker/types"
)

type workerFactroy func(id string, t WorkerType) *Worker

type WorkerManager struct {
	ctx           context.Context
	workers       map[string]*Worker
	workerFactory map[WorkerType]workerFactroy
}

func NewWorkerManager(ctx context.Context) *WorkerManager {
	return &WorkerManager{
		ctx:           ctx,
		workers:       make(map[string]*Worker),
		workerFactory: make(map[WorkerType]workerFactroy),
	}
}

func (w *WorkerManager) SpwanWorker(id string, t WorkerType, handler EventHandler) error {
	if factory := w.workerFactory[t]; factory != nil {
		worker := factory(id, t)
		worker.handler = handler
		w.workers[id] = worker
		worker.Run()
		return nil
	}
	return errors.New("worker type not registered")
}

func (w *WorkerManager) RegisterWorkerFactory(t WorkerType, factory workerFactroy) {
	w.workerFactory[t] = factory
}

func (w *WorkerManager) GetWorker(id string) *Worker {
	return w.workers[id]
}

func (w *WorkerManager) SendEvent(id string, event *types.Event) {
	if worker := w.GetWorker(id); worker != nil {
		worker.msgQueue <- event
	}
}

func (w *WorkerManager) SendEventToAll(event *types.Event) {
	for _, worker := range w.workers {
		worker.msgQueue <- event
	}
}
