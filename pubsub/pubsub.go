// Package pubsub defines the minimal proto PubSub publish contract shared by room game,
// lobby queue, and stream-backed publishers and subscribers.
package pubsub

import (
	"context"

	"github.com/CryptoElementals/common/rpc/proto"
)

// Publisher is the minimal surface for pushing a proto.Event to a topic.
type Publisher interface {
	Publish(ctx context.Context, req *proto.PublishRequest) (*proto.PublishResponse, error)
}

// Publish sends evt to topic. Nil publisher or nil event is a no-op.
func Publish(ctx context.Context, pub Publisher, topic string, evt *proto.Event) error {
	if pub == nil || evt == nil {
		return nil
	}
	if topic == TopicRoom || topic == TopicLobby {
		if evt.MessageId == "" {
			evt.MessageId = BuildEventMessageID(evt)
		}
	}
	_, err := pub.Publish(ctx, &proto.PublishRequest{Topic: topic, Event: evt})
	return err
}
