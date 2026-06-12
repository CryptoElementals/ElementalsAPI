package scanner

import (
	"encoding/json"

	"github.com/CryptoElementals/common/log"
)

type LogSink struct{}

func NewLogSink() *LogSink {
	return &LogSink{}
}

func (s *LogSink) Emit(ev TokenCollectorEvent) {
	data, err := json.Marshal(ev)
	if err != nil {
		log.Errorf("LogSink marshal event: %v", err)
		return
	}
	log.Infof("bsc-scanner event: %s", string(data))
}
