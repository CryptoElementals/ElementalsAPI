package scanner

import (
	"context"
	"encoding/json"

	"github.com/CryptoElementals/common/log"
)

type LogSink struct{}

func NewLogSink() *LogSink {
	return &LogSink{}
}

func (s *LogSink) EmitBlock(ctx context.Context, block *BlockData, events []TokenCollectorEvent) error {
	_ = ctx
	_ = block
	for _, ev := range events {
		s.Emit(ev)
	}
	return nil
}

func (s *LogSink) Close() error {
	return nil
}

func (s *LogSink) Emit(ev TokenCollectorEvent) {
	data, err := json.Marshal(ev)
	if err != nil {
		log.Errorf("LogSink marshal event: %v", err)
		return
	}
	log.Infof("bsc-scanner event: %s", string(data))
}
