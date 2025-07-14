package worker

import (
	"context"
	"math"
)

type State uint16
type WorkerType uint16

const (
	EXIT State = math.MaxUint16
)

type EventSender interface {
	SendEvent(id string, event *Event)
}

type Handler interface {
	Handle(ctx context.Context, sender EventSender, event *Event) (State, error)
}

type Event struct {
	Data interface{}
}

type Worker struct {
	ctx          context.Context
	Id           string
	Type         WorkerType
	CurrentState State
	Handlers     map[State]Handler
	msgQueue     chan *Event
	sender       EventSender
}

func (w *Worker) Run() {
	for {
		select {
		case <-w.ctx.Done():
			return
		case event := <-w.msgQueue:
			if w.Handlers[w.CurrentState] != nil {
				if nextState, err := w.Handlers[w.CurrentState].Handle(w.ctx, w.sender, event); err != nil {
					return
				} else {
					w.CurrentState = nextState
				}
			}
		}
	}
}

func (w *Worker) SetSender(sender EventSender) {
	w.sender = sender
}

func (w *Worker) RegisterHandler(state State, handler Handler) {
	w.Handlers[state] = handler
}
