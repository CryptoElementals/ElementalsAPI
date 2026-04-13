package event_v2

import (
	"context"
	"fmt"
	"sync"

	"github.com/CryptoElementals/common/pubsub"
	"github.com/CryptoElementals/common/rpc/proto"
)

type SubscriberID struct {
	Address  *proto.PlayerAddress
	ClientID string
}

type EventBus interface {
	RegisterSubscriber(subscriberID SubscriberID) (chan *proto.Message, chan error)
	UnregisterSubscriber(subscriberID SubscriberID)
}

type eventBus struct {
	subscriber *pubsub.StreamSubscriber
	topics     []string

	ctx    context.Context
	cancel context.CancelFunc

	mu          sync.RWMutex
	subscribers map[string]map[string]*subscriberState
}

type subscriberState struct {
	id       SubscriberID
	msgCh    chan *proto.Message
	errCh    chan error
	doneOnce sync.Once
	mu       sync.RWMutex
	closed   bool
}

func NewEventBus(subscriber *pubsub.StreamSubscriber, topics ...string) EventBus {
	if subscriber == nil {
		panic("stream subscriber is nil")
	}
	if len(topics) == 0 {
		panic("no topics configured")
	}

	ctx, cancel := context.WithCancel(context.Background())
	b := &eventBus{
		subscriber:  subscriber,
		topics:      topics,
		ctx:         ctx,
		cancel:      cancel,
		subscribers: map[string]map[string]*subscriberState{},
	}
	b.start()
	return b
}

func (b *eventBus) RegisterSubscriber(subscriberID SubscriberID) (chan *proto.Message, chan error) {
	state := &subscriberState{
		id:    subscriberID,
		msgCh: make(chan *proto.Message, 32),
		errCh: make(chan error, 1),
	}
	addressMapKey := subscriberKey(subscriberID)
	subKey := subscriberInstanceKey(subscriberID)

	b.mu.Lock()
	if b.subscribers[addressMapKey] == nil {
		b.subscribers[addressMapKey] = map[string]*subscriberState{}
	}
	b.subscribers[addressMapKey][subKey] = state
	b.mu.Unlock()

	return state.msgCh, state.errCh
}

func (b *eventBus) UnregisterSubscriber(subscriberID SubscriberID) {
	addressMapKey := subscriberKey(subscriberID)
	subKey := subscriberInstanceKey(subscriberID)

	b.mu.Lock()
	group, ok := b.subscribers[addressMapKey]
	if !ok {
		b.mu.Unlock()
		return
	}
	state, exists := group[subKey]
	if !exists {
		b.mu.Unlock()
		return
	}
	b.removeSubscriberLocked(addressMapKey, subKey)
	b.mu.Unlock()

	b.closeSubscriber(state, nil)
}

func (b *eventBus) start() {
	for _, topic := range b.topics {
		t := topic
		msgCh, stop, err := b.subscriber.Subscribe(b.ctx, t, pubsub.SubscribeOptions{})
		if err != nil {
			b.finishAll(fmt.Errorf("subscribe topic %s failed: %w", t, err))
		}

		go func() {
			defer stop()
			for {
				select {
				case <-b.ctx.Done():
					return
				case msg, ok := <-msgCh:
					if !ok {
						if b.ctx.Err() == nil {
							b.finishAll(fmt.Errorf("subscriber channel closed unexpectedly for topic %s", t))
						}
						return
					}
					b.dispatch(msg)
				}
			}
		}()
	}
}

func (b *eventBus) dispatch(msg *proto.Message) {
	ev := msg.GetEvent()
	if msg.GetTopic() == pubsub.TopicTournamentRoster {
		b.dispatchBroadcast(msg)
		return
	}
	receivers := ev.GetReceivers()

	if len(receivers) == 0 {
		// Ignore events with empty receivers.
		return
	}

	// Direct targeting: lookup subscribers by receiver key instead of full scan.
	for _, r := range receivers {
		if r == nil {
			continue
		}
		key := addressKey(r)

		b.mu.RLock()
		group, ok := b.subscribers[key]
		var targets []*subscriberState
		if ok {
			targets = make([]*subscriberState, 0, len(group))
			for _, sub := range group {
				targets = append(targets, sub)
			}
		}
		b.mu.RUnlock()
		if !ok || len(targets) == 0 {
			continue
		}

		for _, sub := range targets {
			b.sendToSubscriber(sub, msg)
		}
	}
}

func (b *eventBus) dispatchBroadcast(msg *proto.Message) {
	b.mu.RLock()
	targets := make([]*subscriberState, 0)
	for _, group := range b.subscribers {
		for _, sub := range group {
			targets = append(targets, sub)
		}
	}
	b.mu.RUnlock()
	for _, sub := range targets {
		b.sendToSubscriber(sub, msg)
	}
}

func (b *eventBus) finishAll(err error) {
	b.cancel()

	b.mu.RLock()
	targets := make([]*subscriberState, 0)
	for _, group := range b.subscribers {
		for _, sub := range group {
			targets = append(targets, sub)
		}
	}
	b.mu.RUnlock()

	for _, sub := range targets {
		b.closeSubscriber(sub, err)
	}
}

func (b *eventBus) closeSubscriber(sub *subscriberState, err error) {
	sub.doneOnce.Do(func() {
		sub.mu.Lock()
		sub.closed = true
		if err != nil {
			select {
			case sub.errCh <- err:
			default:
			}
		}
		close(sub.msgCh)
		close(sub.errCh)
		sub.mu.Unlock()
	})
}

func (b *eventBus) sendToSubscriber(sub *subscriberState, msg *proto.Message) bool {
	sub.mu.RLock()
	defer sub.mu.RUnlock()
	if sub.closed {
		return true
	}

	select {
	case sub.msgCh <- msg:
		return true
	case <-b.ctx.Done():
		return false
	}
}

func subscriberKey(id SubscriberID) string {
	if id.Address == nil {
		return "anonymous"
	}
	return addressKey(id.Address)
}

func addressKey(addr *proto.PlayerAddress) string {
	if addr == nil {
		return "anonymous"
	}
	return fmt.Sprintf("%d:%s", addr.GetId(), addr.GetTemporaryAddress())
}

func subscriberInstanceKey(id SubscriberID) string {
	if id.ClientID != "" {
		return id.ClientID
	}
	if id.Address != nil {
		return addressKey(id.Address)
	}
	return "anonymous"
}

func (b *eventBus) removeSubscriberLocked(addressMapKey, subKey string) {
	group, ok := b.subscribers[addressMapKey]
	if !ok || len(group) == 0 {
		return
	}
	delete(group, subKey)
	if len(group) == 0 {
		delete(b.subscribers, addressMapKey)
	}
}
