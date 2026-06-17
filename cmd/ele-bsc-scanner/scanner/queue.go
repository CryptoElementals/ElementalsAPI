package scanner

import "context"

func requeueBlock(ctx context.Context, blockQueue chan<- uint64, blockNum uint64) {
	select {
	case blockQueue <- blockNum:
	case <-ctx.Done():
	}
}
