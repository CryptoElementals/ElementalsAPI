package types

import "fmt"

const (
	WORKER_TYPE_GAME         = 1
	WORKER_TYPE_PLAYER       = 2
	WORKER_TYPE_GAME_MANAGER = 3
	WORKER_TYPE_CHAIN        = 4
	WORKER_TYPE_QUEUE        = 5
)

const GameTypePVP = 1

// well known worker id
const (
	GAME_MANAGER_ID  = "game_manager"
	QUEUE_MANAGER_ID = "queue_manager"
)

type PlayerAddress struct {
	WalletAddress    string
	TemporaryAddress string
}

func (a *PlayerAddress) String() string {
	return fmt.Sprintf("%s_%s", a.WalletAddress, a.TemporaryAddress)
}

type Event struct {
	EventType uint32
	Sender    string
	Data      any
}

const (
	EVENT_TYPE_ERR = iota
	EVENT_TYPE_NEW_GAME

	EVENT_TYPE_JOIN_QUEUE
	EVENT_TYPE_EXIT_QUEUE
)

type ErrorEvent struct {
	OriginalEvent    *Event
	OriginalReceiver string
	Err              error
}

func NewEvent(sender string, eventType uint32, evt any) *Event {
	return &Event{
		EventType: EVENT_TYPE_ERR,
		Sender:    sender,
		Data:      evt,
	}
}
