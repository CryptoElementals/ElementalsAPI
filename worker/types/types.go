package types

import (
	"fmt"
	"strings"

	dao "github.com/CryptoElementals/common/models"
	"github.com/CryptoElementals/common/rpc/proto"
)

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
	CHAIN_MANAGER_ID = "chain_manager"
)

type PlayerAddress struct {
	WalletAddress    string
	TemporaryAddress string
}

func (a *PlayerAddress) String() string {
	return fmt.Sprintf("%s_%s", a.WalletAddress, a.TemporaryAddress)
}

func (a *PlayerAddress) Parse(str string) error {
	parts := strings.Split(str, "_")
	if len(parts) != 2 {
		return fmt.Errorf("invalid player address")
	}
	a.WalletAddress = parts[0]
	a.TemporaryAddress = parts[1]
	return nil
}

func (a *PlayerAddress) ToDao() dao.GamePlayer {
	return dao.GamePlayer{
		WalletAddress:    a.WalletAddress,
		TemporaryAddress: a.TemporaryAddress,
	}
}

func (a *PlayerAddress) ToProto() *proto.PlayerAddress {
	return &proto.PlayerAddress{
		WalletAddress:    a.WalletAddress,
		TemporaryAddress: a.TemporaryAddress,
	}
}

func (a *PlayerAddress) FromDao(player dao.GamePlayer) {
	a.WalletAddress = player.WalletAddress
	a.TemporaryAddress = player.TemporaryAddress
}

type Event struct {
	EventType uint32
	Sender    string
	Data      any
}

const (
	EVENT_TYPE_ERR = iota
	// queue related event
	EVENT_TYPE_NEW_GAME
	EVENT_TYPE_JOIN_QUEUE
	EVENT_TYPE_EXIT_QUEUE

	// game related eventws
	EVENT_TYPE_GAME_CREATED
	EVENT_TYPE_GAME_READY
	EVENT_TYPE_ROUND_READY
	EVENT_TYPE_COMMITMENTS_ON_CHAIN
	EVENT_TYPE_CARDS_ON_CHAIN
	EVENT_TYPE_ROUND_COMPLETED
	EVENT_TYPE_GAME_COMPLETED
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
