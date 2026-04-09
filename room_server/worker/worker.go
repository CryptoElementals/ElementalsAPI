// Package worker implements the room server's in-process worker pool: one goroutine per logical id
// (player, chain manager, game manager) with events routed through WorkerManager.
package worker

import (
	"context"
	"reflect"

	"github.com/CryptoElementals/common/log"
	"github.com/CryptoElementals/common/room_server/worker/types"
)

type State uint16
type WorkerType uint16

type EventSender interface {
	SendEvent(to string, event *types.Event)
}

type WorkerCloser interface {
	CloseWorker(string)
}

type EventHandler interface {
	Handle(ctx context.Context, event *types.Event) error
}

type Worker struct {
	ctx      context.Context
	ccl      context.CancelFunc
	Id       string
	Type     WorkerType
	handler  EventHandler
	msgQueue chan *types.Event
	sender   EventSender
	closer   WorkerCloser
}

func signalAck(event *types.Event, handleErr error, workerID string) {
	if event == nil || event.AckChan == nil {
		return
	}
	if handleErr != nil {
		log.Debugw("worker handle event failed", "worker id", workerID, "event", types.ToJsonLoggable(event), "err", handleErr)
		event.AckChan <- handleErr
	}
	close(event.AckChan)
}

func NewWorker(ctx context.Context, id string, t WorkerType, workerCloser WorkerCloser, sender EventSender, handler EventHandler) *Worker {
	ctx, ccl := context.WithCancel(ctx)
	return &Worker{
		ctx:      ctx,
		ccl:      ccl,
		Id:       id,
		Type:     t,
		closer:   workerCloser,
		handler:  handler,
		sender:   sender,
		msgQueue: make(chan *types.Event, 100),
	}
}

func (w *Worker) Run() {
	for {
		select {
		case <-w.ctx.Done():
			w.closer.CloseWorker(w.Id)
			return
		case event := <-w.msgQueue:
			log.Debugw("worker received event", "worker id", w.Id, "eventType", reflect.TypeOf(event.Data))
			err := w.handler.Handle(w.ctx, event)
			signalAck(event, err, w.Id)
		}
	}
}

func (w *Worker) SendEvent(to string, event *types.Event) {
	w.sender.SendEvent(to, event)
}
