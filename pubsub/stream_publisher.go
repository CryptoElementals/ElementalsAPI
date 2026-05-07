package pubsub

import (
	"context"
	"fmt"
	"time"

	"github.com/CryptoElementals/common/rpc/proto"
	"github.com/CryptoElementals/common/stream"
	goproto "google.golang.org/protobuf/proto"
)

// StreamPublisher adapts a Stream backend into [Publisher]. The Redis stream key is
// exactly req.Topic (same string used by [StreamSubscriber]).
type StreamPublisher struct {
	stream stream.Stream
	topic  string
}

// NewStreamPublisher creates a publisher that writes proto.Event messages into
// the underlying Stream, one stream per topic.
func NewStreamPublisher(s stream.Stream, topic string) *StreamPublisher {
	return &StreamPublisher{stream: s, topic: topic}
}

func (p *StreamPublisher) Topic() string {
	return p.topic
}

// Publish serializes the proto.Event and writes it to the stream keyed by Topic().
func (p *StreamPublisher) Publish(ctx context.Context, evt *proto.Event) (*proto.PublishResponse, error) {
	if p.topic == "" {
		return nil, fmt.Errorf("topic is required")
	}
	if evt == nil {
		return nil, fmt.Errorf("event is required")
	}

	payload, err := goproto.Marshal(evt)
	if err != nil {
		return nil, fmt.Errorf("marshal event: %w", err)
	}

	ts := time.Now().UnixMilli()
	msgID, err := p.stream.Publish(ctx, p.topic, p.topic, payload, ts)
	if err != nil {
		return nil, fmt.Errorf("stream publish: %w", err)
	}

	return &proto.PublishResponse{
		MessageId: msgID,
		Success:   true,
	}, nil
}
