package client

import (
	"context"
	"testing"

	"github.com/CryptoElementals/common/pubsub"
	"github.com/CryptoElementals/common/rpc/proto"
	"github.com/stretchr/testify/require"
)

func TestEventBusDispatchTokenToPlayerID(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	bus := &eventBus{
		ctx:                  ctx,
		cancel:               cancel,
		subscribers:          map[string]map[string]*subscriberState{},
		allEventsSubscribers: map[string]*subscriberState{},
	}

	player1 := &proto.PlayerAddress{Id: 1, TemporaryAddress: "0xaa"}
	player2 := &proto.PlayerAddress{Id: 2, TemporaryAddress: "0xbb"}

	ch1, _ := registerTestSubscriber(bus, SubscriberID{Address: player1, ClientID: "c1"})
	ch2, _ := registerTestSubscriber(bus, SubscriberID{Address: player2, ClientID: "c2"})
	ch1b, _ := registerTestSubscriber(bus, SubscriberID{
		Address:  &proto.PlayerAddress{Id: 1, TemporaryAddress: "0xcc"},
		ClientID: "c1b",
	})

	msg := tokenUpdatedMessage(1, 1000)
	bus.dispatch(msg)

	require.Len(t, drain(ch1), 1)
	require.Len(t, drain(ch1b), 1)
	require.Len(t, drain(ch2), 0)
}

func TestEventBusDispatchTokenToAllEventsSubscriber(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	bus := &eventBus{
		ctx:                  ctx,
		cancel:               cancel,
		subscribers:          map[string]map[string]*subscriberState{},
		allEventsSubscribers: map[string]*subscriberState{},
	}

	player1 := &proto.PlayerAddress{Id: 1, TemporaryAddress: "0xaa"}
	chPlayer, _ := registerTestSubscriber(bus, SubscriberID{Address: player1, ClientID: "c1"})
	chAll, _ := bus.RegisterAllEventsSubscriber("all")

	msg1 := tokenUpdatedMessage(1, 1000)
	msg2 := tokenUpdatedMessage(2, 2000)
	bus.dispatch(msg1)
	bus.dispatch(msg2)

	require.Len(t, drain(chAll), 2)
	require.Len(t, drain(chPlayer), 1)
}

func tokenUpdatedMessage(playerID int64, tokens int32) *proto.Message {
	return &proto.Message{
		Topic: pubsub.TopicToken,
		Event: &proto.Event{
			Type: proto.EventType_TYPE_TOKEN_UPDATED,
			Event: &proto.Event_TokenUpdated{
				TokenUpdated: &proto.TokenUpdated{
					PlayerId: playerID,
					Tokens:   tokens,
				},
			},
		},
	}
}

func registerTestSubscriber(bus *eventBus, id SubscriberID) (chan *proto.Message, chan error) {
	state := &subscriberState{
		id:    id,
		msgCh: make(chan *proto.Message, 4),
		errCh: make(chan error, 1),
	}
	key := subscriberKey(id)
	subKey := subscriberInstanceKey(id)
	if bus.subscribers[key] == nil {
		bus.subscribers[key] = map[string]*subscriberState{}
	}
	bus.subscribers[key][subKey] = state
	return state.msgCh, state.errCh
}

func drain(ch chan *proto.Message) []*proto.Message {
	var out []*proto.Message
	for {
		select {
		case m := <-ch:
			out = append(out, m)
		default:
			return out
		}
	}
}
