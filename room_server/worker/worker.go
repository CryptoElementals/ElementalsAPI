package worker

import (
	"context"

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
			sender := event.Sender
			err := w.handler.Handle(w.ctx, event)
			if err != nil {
				errEvent := types.NewEvent(w.Id, &types.ErrorEvent{
					OriginalEvent:    event,
					OriginalReceiver: w.Id,
					Err:              err,
				})
				w.SendEvent(sender, errEvent)
			} else if event.NeedAck {
				ackEvent := types.NewEvent(w.Id, &types.AckEvent{
					EventID: event.EventID,
				})
				w.SendEvent(sender, ackEvent)
			}
		}
	}
}

func (w *Worker) SendEvent(to string, event *types.Event) {
	w.sender.SendEvent(to, event)
}
