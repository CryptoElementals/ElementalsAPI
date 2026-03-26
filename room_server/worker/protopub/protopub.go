// Package protopub centralizes gRPC PubSub-style publishes used by game and queue workers.
package protopub

import (
	"context"

	"github.com/CryptoElementals/common/rpc/proto"
)

// Publisher is the minimal surface for pushing a proto.Event to a topic (player id, worker id, etc.).
type Publisher interface {
	Publish(ctx context.Context, req *proto.PublishRequest) (*proto.PublishResponse, error)
}

// Publish sends evt to topic. Nil publisher or nil event is a no-op.
func Publish(ctx context.Context, pub Publisher, topic string, evt *proto.Event) error {
	if pub == nil || evt == nil {
		return nil
	}
	_, err := pub.Publish(ctx, &proto.PublishRequest{Topic: topic, Event: evt})
	return err
}
