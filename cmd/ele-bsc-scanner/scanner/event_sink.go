package scanner

import "context"

// EventSink receives parsed TokenCollector events for a finalized block.
type EventSink interface {
	EmitBlock(ctx context.Context, block *BlockData, events []TokenCollectorEvent) error
	Close() error
}
