package types

type NewGameEvent struct {
	MsgSender string
	Players   []PlayerAddress
}

// EventType implements Event.
func (e *NewGameEvent) EventType() uint32 {
	return EVENT_TYPE_NEW_GAME
}

// Sender implements Event.
func (e *NewGameEvent) Sender() string {
	return e.MsgSender
}
