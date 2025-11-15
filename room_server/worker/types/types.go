package types

import (
	"encoding/json"
	"fmt"
	"reflect"
	"strings"
	"time"

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

func NewPlayerAddress(walletAddress, temporaryAddress string) *PlayerAddress {
	return &PlayerAddress{
		WalletAddress:    strings.ToLower(walletAddress),
		TemporaryAddress: strings.ToLower(temporaryAddress),
	}
}

func (a *PlayerAddress) String() string {
	return fmt.Sprintf("%s_%s", a.WalletAddress, a.TemporaryAddress)
}

func (a *PlayerAddress) Parse(str string) error {
	parts := strings.Split(str, "_")
	if len(parts) != 2 {
		return fmt.Errorf("invalid player address")
	}
	a.WalletAddress = strings.ToLower(parts[0])
	a.TemporaryAddress = strings.ToLower(parts[1])
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

func (a *PlayerAddress) ToProtoNoWallet() *proto.PlayerAddress {
	return &proto.PlayerAddress{
		TemporaryAddress: strings.ToLower(a.TemporaryAddress),
	}
}

func (a *PlayerAddress) FromDao(player dao.GamePlayerInfo) {
	a.WalletAddress = strings.ToLower(player.WalletAddress)
	a.TemporaryAddress = strings.ToLower(player.TemporaryAddress)
}

func (a *PlayerAddress) FromProto(player *proto.PlayerAddress) {
	a.WalletAddress = strings.ToLower(player.WalletAddress)
	a.TemporaryAddress = strings.ToLower(player.TemporaryAddress)
}

type Event struct {
	Sender  string
	EventID string
	AckChan chan any `json:",omitempty"` // Response channel that returns error or any value
	Data    any
}

func NewEvent(sender string, evt any, needAck ...bool) *Event {
	var ack chan any
	if len(needAck) > 0 && needAck[0] {
		ack = make(chan any, 1) // Buffered channel to avoid blocking
	}
	eid := uuid.NewString()
	return &Event{
		Sender:  sender,
		EventID: eid,
		AckChan: ack,
		Data:    evt,
	}
}

// Await waits for the response from AckChan and returns the value or error
func (e *Event) Await() (any, error) {
	if e.AckChan == nil {
		return nil, nil
	}
	response := <-e.AckChan
	if err, ok := response.(error); ok {
		return nil, err
	}
	return response, nil
}

func AssertInterface[T any](evt *Event) (T, error) {
	data, ok := evt.Data.(T)
	if !ok {
		t := *new(T)
		return *new(T), fmt.Errorf("event data type not match: %s, received: %s", reflect.TypeOf(t), reflect.TypeOf(evt.Data))
	}
	return data, nil
}

type BatchDone struct {
	ID string
}

type EventBatch struct {
	evt []*Event
}

func (b *EventBatch) Add(evt *Event) {
	b.evt = append(b.evt, evt)
}

func (b *EventBatch) Wait() {
	for _, e := range b.evt {
		_, _ = e.Await() // Ignore response and error for batch operations
	}
}

func ToJsonLoggable(obj any) string {
	res, _ := json.Marshal(obj)
	return string(res)
}

type GameContinueInfo struct {
	GameID          uint
	EndTime         time.Time
	ContinueTimeout int64
	Players         []PlayerAddress
}
