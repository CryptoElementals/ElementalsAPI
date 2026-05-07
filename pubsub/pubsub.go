// Package pubsub defines the minimal proto PubSub publish contract shared by room game,
// lobby queue, and stream-backed publishers and subscribers.
package pubsub

import (
	"context"

	"github.com/CryptoElementals/common/rpc/proto"
)

// Publisher is the minimal surface for pushing a proto.Event to a topic.
type Publisher interface {
	Publish(ctx context.Context, evt *proto.Event) (*proto.PublishResponse, error)
	Topic() string
}

// Publish sends evt to pub.Topic(). Nil publisher or nil event is a no-op.
func Publish(ctx context.Context, pub Publisher, evt *proto.Event) error {
	if pub == nil || evt == nil {
		return nil
	}
	topic := pub.Topic()
	if topic == TopicRoom || topic == TopicLobby || topic == TopicTournamentRoster {
		if evt.MessageId == "" {
			evt.MessageId = BuildEventMessageID(evt)
		}
	}
	_, err := pub.Publish(ctx, evt)
	return err
}
