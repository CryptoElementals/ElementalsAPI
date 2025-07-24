package types

import (
	"fmt"
	"reflect"
	"strings"

	dao "github.com/CryptoElementals/common/models"
	"github.com/CryptoElementals/common/rpc/proto"
	"github.com/google/uuid"
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
	GAME_MANAGER_ID   = "game_manager"
	QUEUE_MANAGER_ID  = "queue_manager"
	CHAIN_MANAGER_ID  = "chain_manager"
	WORKER_MANAGER_ID = "worker_manager"
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

func (a *PlayerAddress) ToDao() *dao.GamePlayerInfo {
	return &dao.GamePlayerInfo{
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

func (a *PlayerAddress) FromDao(player dao.GamePlayerInfo) {
	a.WalletAddress = player.WalletAddress
	a.TemporaryAddress = player.TemporaryAddress
}

func (a *PlayerAddress) FromProto(player *proto.PlayerAddress) {
	a.WalletAddress = player.WalletAddress
	a.TemporaryAddress = player.TemporaryAddress
}

type Event struct {
	Sender  string
	EventID string
	NeedAck bool
	Data    any
}

type ErrorEvent struct {
	OriginalEvent    *Event
	OriginalReceiver string
	Err              error
}

func NewEvent(sender string, evt any, needAck ...bool) *Event {
	ack := false
	if len(needAck) != 0 && needAck[0] {
		ack = true
	}

	eid := uuid.NewString()
	return &Event{
		Sender:  sender,
		EventID: eid,
		NeedAck: ack,
		Data:    evt,
	}
}

func AssertInterface[T any](evt *Event) (T, error) {
	data, ok := evt.Data.(T)
	if !ok {
		t := *new(T)
		return *new(T), fmt.Errorf("event data type not match: %s, received: %s", reflect.TypeOf(t), reflect.TypeOf(evt.Data))
	}
	return data, nil
}

type AckEvent struct {
	EventID string
}
