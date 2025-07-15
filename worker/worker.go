package worker

import (
	"context"

	"github.com/CryptoElementals/common/worker/types"
)

type State uint16
type WorkerType uint16

type EventSender interface {
	SendEvent(to string, event *types.Event)
}

type EventHandler interface {
	Handle(ctx context.Context, event *types.Event) error
}

type Worker struct {
	ctx      context.Context
	Id       string
	Type     WorkerType
	handler  EventHandler
	msgQueue chan *types.Event
	sender   EventSender
}

func NewWorker(ctx context.Context, id string, t WorkerType) *Worker {
	return &Worker{
		ctx:      ctx,
		Id:       id,
		Type:     t,
		msgQueue: make(chan *types.Event, 100),
	}
}

func (w *Worker) Run() {
	for {
		select {
		case <-w.ctx.Done():
			return
		case event := <-w.msgQueue:
			err := w.handler.Handle(w.ctx, event)
			if err != nil {
				sender := event.Sender
				errEvent := types.NewEvent(w.Id, types.EVENT_TYPE_ERR, &types.ErrorEvent{
					OriginalEvent:    event,
					OriginalReceiver: w.Id,
					Err:              err,
				})
				w.sender.SendEvent(sender, errEvent)
			}
		}
	}
}

func (w *Worker) SetSender(sender EventSender) {
	w.sender = sender
}
