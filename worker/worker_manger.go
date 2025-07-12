package worker

import (
	"context"
	"errors"
)

type WorkerManager struct {
	ctx           context.Context
	workers       map[string]*Worker
	workerFactory map[WorkerType]func(ctx context.Context, id string, t WorkerType) *Worker
}

func NewWorkerManager(ctx context.Context) *WorkerManager {
	return &WorkerManager{
		ctx:     ctx,
		workers: make(map[string]*Worker),
	}
}

func (w *WorkerManager) SpwanWorker(id string, t WorkerType, recoveryEvent *Event) error {
	if factory := w.workerFactory[t]; factory != nil {
		worker := factory(w.ctx, id, t)
		w.workers[id] = worker
		worker.msgQueue <- recoveryEvent
		worker.Run()
		return nil
	}
	return errors.New("worker type not registered")
}

func (w *WorkerManager) RegisterWorkerFactory(t WorkerType, factory func(ctx context.Context, id string, t WorkerType) *Worker) {
	w.workerFactory[t] = factory
}

func (w *WorkerManager) GetWorker(id string) *Worker {
	return w.workers[id]
}

func (w *WorkerManager) SendEvent(id string, event *Event) {
	if worker := w.GetWorker(id); worker != nil {
		worker.msgQueue <- event
	}
}

func (w *WorkerManager) SendEventToAll(event *Event) {
	for _, worker := range w.workers {
		worker.msgQueue <- event
	}
}
