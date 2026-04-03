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
}

// NewStreamPublisher creates a publisher that writes proto.Event messages into
// the underlying Stream, one stream per topic.
func NewStreamPublisher(s stream.Stream) *StreamPublisher {
	return &StreamPublisher{stream: s}
}

// Publish serializes the proto.Event and writes it to the stream keyed by req.Topic.
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
	msgID, err := p.stream.Publish(ctx, req.Topic, req.Topic, payload, ts)
	if err != nil {
		return nil, fmt.Errorf("stream publish: %w", err)
	}

	return &proto.PublishResponse{
		MessageId: msgID,
		Success:   true,
	}, nil
}
