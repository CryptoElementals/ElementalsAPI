package scanner

import "context"

// BlockHandler applies registry updates and parses events in block order.
type BlockHandler struct {
	registry  *WalletRegistry
	processor *TokenProcessor
	sink      *LogSink
}

func NewBlockHandler(registry *WalletRegistry, processor *TokenProcessor, sink *LogSink) *BlockHandler {
	return &BlockHandler{
		registry:  registry,
		processor: processor,
		sink:      sink,
	}
}

func (h *BlockHandler) HandleBlock(ctx context.Context, data *BlockData) error {
	if data == nil {
		return nil
	}
	for _, lg := range data.Logs {
		if err := h.registry.ApplyReceiptLog(ctx, lg, data.BlockNumber); err != nil {
			return err
		}
	}
	events := h.processor.ParseEvents(data)
	for _, ev := range events {
		h.sink.Emit(ev)
	}
	return nil
}
