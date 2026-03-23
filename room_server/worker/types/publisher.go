package types

import (
	"context"
	"fmt"
	"time"

	"github.com/CryptoElementals/common/rpc/proto"
	"github.com/CryptoElementals/common/stream"
	goproto "google.golang.org/protobuf/proto"
)

// StreamPublisher adapts a Stream backend into the Publisher / EventPublisher
// interface used by game.Service and queue.Service. Each player topic maps to
// its own Redis stream key (<prefix>:<topic>), so consumers can XREAD from a
// single, well-known key per player.
type StreamPublisher struct {
	stream stream.Stream
	prefix string // Redis key prefix, e.g. "player_events"
}

// NewStreamPublisher creates a publisher that writes proto.Event messages into
// the underlying Stream. prefix is prepended to the topic to form the stream
// key (e.g. prefix "player_events" + topic "1_0xabc" → "player_events:1_0xabc").
func NewStreamPublisher(s stream.Stream, prefix string) *StreamPublisher {
	return &StreamPublisher{stream: s, prefix: prefix}
}

func (p *StreamPublisher) streamKey(topic string) string {
	if p.prefix == "" {
		return topic
	}
	return p.prefix + ":" + topic
}

// Publish serializes the proto.Event and writes it to the stream keyed by topic.
// It satisfies both game.Publisher and queue.EventPublisher.
func (p *StreamPublisher) Publish(ctx context.Context, req *proto.PublishRequest) (*proto.PublishResponse, error) {
	if req.Topic == "" {
		return nil, fmt.Errorf("topic is required")
	}
	if req.Event == nil {
		return nil, fmt.Errorf("event is required")
	}

	payload, err := goproto.Marshal(req.Event)
	if err != nil {
		return nil, fmt.Errorf("marshal event: %w", err)
	}

	ts := time.Now().UnixMilli()
	msgID, err := p.stream.Publish(ctx, p.streamKey(req.Topic), req.Topic, payload, ts)
	if err != nil {
		return nil, fmt.Errorf("stream publish: %w", err)
	}

	return &proto.PublishResponse{
		MessageId: msgID,
		Success:   true,
	}, nil
}
