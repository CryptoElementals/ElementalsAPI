package pubsub

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/CryptoElementals/common/rpc/proto"
	"github.com/CryptoElementals/common/stream"
	goproto "google.golang.org/protobuf/proto"
)

// SubscribeOptions configures [StreamSubscriber.Subscribe].
type SubscribeOptions struct {
	// StartID is the Redis stream ID for XREAD. "$" = only entries added after subscribe; "0-0" = from the start.
	// Empty string defaults to "$".
	StartID string
	// BlockMS is XREAD BLOCK duration in milliseconds. Zero defaults to 1000.
	BlockMS int
}

// StreamSubscriber reads proto.Events from the same stream keys [StreamPublisher] writes to (stream name == topic).
type StreamSubscriber struct {
	stream stream.Stream
}

// NewStreamSubscriber creates a subscriber backed by s.
func NewStreamSubscriber(s stream.Stream) *StreamSubscriber {
	return &StreamSubscriber{stream: s}
}

// Subscribe starts a background read loop on stream key topic. It unmarshals each entry's payload as *proto.Event
// and sends a *proto.Message (MessageId = Redis ID, Topic and Timestamp from the entry).
//
// Call the returned cancel function to stop the loop and close the channel (waits until the reader exits).
func (s *StreamSubscriber) Subscribe(ctx context.Context, topic string, opts SubscribeOptions) (<-chan *proto.Message, context.CancelFunc, error) {
	if topic == "" {
		return nil, nil, fmt.Errorf("topic is required")
	}
	if s.stream == nil {
		return nil, nil, fmt.Errorf("stream is nil")
	}

	startID := opts.StartID
	if startID == "" {
		startID = "$"
	}
	block := opts.BlockMS
	if block <= 0 {
		block = 1000
	}

	ctx, cancel := context.WithCancel(ctx)
	out := make(chan *proto.Message, 32)
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		defer close(out)
		lastID := startID
		for {
			select {
			case <-ctx.Done():
				return
			default:
			}

			entries, err := s.stream.Read(ctx, topic, lastID, block)
			if err != nil {
				select {
				case <-ctx.Done():
					return
				case <-time.After(500 * time.Millisecond):
				}
				continue
			}

			for _, e := range entries {
				lastID = e.ID
				if len(e.Payload) == 0 {
					continue
				}
				var ev proto.Event
				if err := goproto.Unmarshal(e.Payload, &ev); err != nil {
					continue
				}
				msg := &proto.Message{
					MessageId:   e.ID,
					Topic:       e.Topic,
					Timestamp:   e.Timestamp,
					Event:       goproto.Clone(&ev).(*proto.Event),
					Metadata:    nil,
					PublisherId: "",
				}
				select {
				case out <- msg:
				case <-ctx.Done():
					return
				}
			}
		}
	}()

	stop := func() {
		cancel()
		wg.Wait()
	}
	return out, stop, nil
}
