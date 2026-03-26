package stream

import (
	"context"
	"time"
)

// Entry represents a message in the stream.
type Entry struct {
	ID        string
	Topic     string
	Payload   []byte
	Timestamp int64
}

// Stream abstracts a message stream backend (Redis, Kafka, etc.).
// Implementations can be swapped without changing callers.
type Stream interface {
	// Publish adds a message to the stream. Returns the message ID.
	Publish(ctx context.Context, stream string, topic string, payload []byte, ts int64) (string, error)

	// Read reads messages from the stream. startID="$" means only new messages.
	// blockMs: block up to this many ms waiting for data; 0 = non-blocking.
	// Returns entries (may be empty on timeout) and nil error.
	Read(ctx context.Context, stream string, startID string, blockMs int) ([]Entry, error)

	// Trim removes entries older than maxAge. Returns the number of entries deleted.
	Trim(ctx context.Context, stream string, maxAge time.Duration) (int, error)

	// Len returns the number of entries in the stream.
	Len(ctx context.Context, stream string) (int, error)

	// Range returns entries with ID in [startID, endID]. Use "-" and "+" for min/max.
	Range(ctx context.Context, stream string, startID, endID string) ([]Entry, error)
}
