package utils

import (
	"sync"
	"sync/atomic"
	"time"
)

// A minimal Snowflake-like ID generator without external deps:
// 41 bits timestamp (ms since custom epoch)
// 10 bits node (fixed 1)
// 12 bits sequence per millisecond

const (
	customEpochMs int64 = 1762819200000 // 2025-11-11
	nodeIDBits          = 10
	sequenceBits        = 12

	maxSequence = (1 << sequenceBits) - 1
	nodeID      = 1
)

var (
	lastTimestamp int64
	sequence      uint32
	genMu         sync.Mutex
)

func currentMs() int64 {
	return time.Now().UnixNano() / int64(time.Millisecond)
}

// GenerateSnowflakeID returns a unique uint64 ID.
func GenerateSnowflakeID() uint64 {
	genMu.Lock()
	defer genMu.Unlock()

	ts := currentMs()
	if ts == atomic.LoadInt64(&lastTimestamp) {
		sequence = (sequence + 1) & maxSequence
		if sequence == 0 {
			// spin until next millisecond
			for ts <= atomic.LoadInt64(&lastTimestamp) {
				ts = currentMs()
			}
		}
	} else {
		sequence = 0
	}
	atomic.StoreInt64(&lastTimestamp, ts)

	timestampPart := uint64(ts - customEpochMs)
	nodePart := uint64(nodeID)
	seqPart := uint64(sequence)

	return (timestampPart << (nodeIDBits + sequenceBits)) |
		(nodePart << sequenceBits) |
		seqPart
}
