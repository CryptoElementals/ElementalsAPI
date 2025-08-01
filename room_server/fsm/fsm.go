package fsm

import (
	"context"

	"github.com/looplab/fsm"
)

type FSMTypeConverter[T any] interface {
	String(T) string
	Parse(string) T
}

type Event[T any] struct {
	Name string
	Src  []T
	Dst  T
}

type EventContext[E any, T FSMTypeConverter[E]] struct {
	// FSM is an reference to the current FSM.
	FSM *FSM[E, T]

	// Event is the event name.
	Event string

	// Src is the state before the transition.
	Src E

	// Dst is the state after the transition.
	Dst E

	// Err is an optional error that can be returned from a callback.
	Err error

	// Args is an optional list of arguments passed to the callback.
	Args []any
}

type FSM[E any, T FSMTypeConverter[E]] struct {
	stateMachine *fsm.FSM
	converter    T
}

type Callback[E any, T FSMTypeConverter[E]] func(context.Context, *EventContext[E, T])

func NewFSM[E any, T FSMTypeConverter[E]](
	initial E,
	events []Event[E],
	callbacks map[string]Callback[E, T],
	converter T,
) *FSM[E, T] {
	stateMachine := &FSM[E, T]{
		converter: converter,
	}
	eventDescs := make([]fsm.EventDesc, 0, len(events))
	for _, event := range events {
		eventDescs = append(eventDescs, stateMachine.eventToLooplabFsmEventDsc(event))
	}
	looplabCallbacks := make(map[string]fsm.Callback)
	for key, callback := range callbacks {
		looplabCallbacks[key] = func(ctx context.Context, e *fsm.Event) {
			evt := stateMachine.EventContextFromLoopLabEvent(e)
			callback(ctx, evt)
		}
	}
	sm := fsm.NewFSM(converter.String(initial), eventDescs, looplabCallbacks)
	stateMachine.stateMachine = sm
	return stateMachine
}

func (m *FSM[E, T]) eventToLooplabFsmEventDsc(e Event[E]) fsm.EventDesc {
	src := make([]string, 0, len(e.Src))
	for _, s := range e.Src {
		src = append(src, m.converter.String(s))
	}
	return fsm.EventDesc{
		Name: e.Name,
		Src:  src,
		Dst:  m.converter.String(e.Dst),
	}
}

func (m *FSM[E, T]) EventContextToLoopLabEvent(ctx *EventContext[E, T]) *fsm.Event {
	return &fsm.Event{
		FSM:   ctx.FSM.stateMachine,
		Event: ctx.Event,
		Src:   m.converter.String(ctx.Src),
		Dst:   m.converter.String(ctx.Dst),
		Err:   ctx.Err,
		Args:  ctx.Args,
	}
}

func (m *FSM[E, T]) EventContextFromLoopLabEvent(e *fsm.Event) *EventContext[E, T] {

	src := m.converter.Parse(e.Src)
	dst := m.converter.Parse(e.Dst)
	return &EventContext[E, T]{
		FSM:   &FSM[E, T]{stateMachine: e.FSM},
		Event: e.Event,
		Src:   src,
		Dst:   dst,
		Err:   e.Err,
		Args:  e.Args,
	}
}

func (m *FSM[E, T]) AvailableTransitions() []E {
	transitions := m.stateMachine.AvailableTransitions()
	var results []E
	for _, transition := range transitions {
		results = append(results, m.converter.Parse(transition))
	}
	return results
}

func (m *FSM[E, T]) Can(event E) bool {
	return m.stateMachine.Can(m.converter.String(event))
}

func (m *FSM[E, T]) Cannot(event E) bool {
	return m.stateMachine.Cannot(m.converter.String(event))
}

func (m *FSM[E, T]) Current() E {
	return m.converter.Parse(m.stateMachine.Current())
}
func (m *FSM[E, T]) DeleteMetadata(key E) {
	m.stateMachine.DeleteMetadata(m.converter.String(key))
}
func (m *FSM[E, T]) Event(ctx context.Context, event E, args ...interface{}) error {
	return m.stateMachine.Event(ctx, m.converter.String(event), args...)
}
func (m *FSM[E, T]) Is(state E) bool {
	return m.stateMachine.Is(m.converter.String(state))
}
func (m *FSM[E, T]) Metadata(key E) (interface{}, bool) {
	return m.stateMachine.Metadata(m.converter.String(key))
}
func (m *FSM[E, T]) SetMetadata(key E, dataValue interface{}) {
	m.stateMachine.SetMetadata(m.converter.String(key), dataValue)
}
func (m *FSM[E, T]) SetState(state E) {
	m.stateMachine.SetState(m.converter.String(state))
}
func (m *FSM[E, T]) Transition() error {
	return m.stateMachine.Transition()
}

func (m *FSM[E, T]) Visualize() string {
	return fsm.Visualize(m.stateMachine)
}
