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

// PendingSummary is the aggregate result of XPENDING <stream> <group> (Redis consumer groups).
type PendingSummary struct {
	Count     int64
	MinID     string
	MaxID     string
	Consumers []PendingConsumer
}

// PendingConsumer is one row in the consumers section of PendingSummary.
type PendingConsumer struct {
	Name  string
	Count int64
}

// AutoClaimResult is the parsed outcome of XAUTOCLAIM (claimed entries + scan cursor).
type AutoClaimResult struct {
	Entries   []Entry
	NextStart string
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

	// GroupCreate creates a consumer group (XGROUP CREATE). startID "0" reads from the beginning, "$" only new entries.
	GroupCreate(ctx context.Context, stream, group, startID string, mkstream bool) error

	// GroupDestroy deletes a consumer group (XGROUP DESTROY). Returns Redis's integer reply.
	GroupDestroy(ctx context.Context, stream, group string) (int, error)

	// GroupDelConsumer removes a consumer from the group (XGROUP DELCONSUMER).
	GroupDelConsumer(ctx context.Context, stream, group, consumer string) (int, error)

	// ReadGroup reads via XREADGROUP. readID ">" delivers new messages; "0" or a specific ID reads the pending entries list for this consumer.
	ReadGroup(ctx context.Context, stream, group, consumer, readID string, count int, blockMs int) ([]Entry, error)

	// Ack acknowledges messages (XACK), removing them from the group's pending list.
	Ack(ctx context.Context, stream, group string, messageIDs ...string) (int, error)

	// Pending returns aggregate pending stats (XPENDING stream group).
	Pending(ctx context.Context, stream, group string) (PendingSummary, error)

	// Claim takes idle pending messages and assigns them to consumer (XCLAIM). Requires at least one messageID.
	Claim(ctx context.Context, stream, group, consumer string, minIdleMs int, messageIDs ...string) ([]Entry, error)

	// AutoClaim scans the group's pending list and claims entries idle longer than minIdleMs (XAUTOCLAIM).
	// start is the scan cursor ("0-0" to begin); use returned NextStart on the next call to continue scanning.
	AutoClaim(ctx context.Context, stream, group, consumer string, minIdleMs int, start string, count int) (AutoClaimResult, error)
}
