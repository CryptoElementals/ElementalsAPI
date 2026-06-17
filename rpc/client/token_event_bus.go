package client

import (
	"github.com/CryptoElementals/common/pubsub"
)

// NewTokenEventBus subscribes only to token_events for SubscribeTokenUpdates SSE.
func NewTokenEventBus(subscriber *pubsub.StreamSubscriber) EventBus {
	return NewEventBus(subscriber, pubsub.TopicToken)
}
