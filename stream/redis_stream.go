package stream

import (
	"context"
	"encoding/base64"
	"fmt"
	"strconv"
	"time"

	"github.com/CryptoElementals/common/redis"
	redigo "github.com/gomodule/redigo/redis"
)

var _ Stream = (*RedisStream)(nil)

// RedisStream implements Stream against a fixed [redis.RedisOperator] (default or named pool from [redis.Init]).
type RedisStream struct {
	op *redis.RedisOperator
}

// NewRedisStream returns a Redis-backed Stream on the default pool. Requires redis.Init to be called first.
func NewRedisStream() (Stream, error) {
	op, err := redis.DefaultOperator()
	if err != nil {
		return nil, err
	}
	if err := op.Ping(); err != nil {
		return nil, err
	}
	return &RedisStream{op: op}, nil
}

// NewRedisStreamForPool returns a Stream backed by the named Redis pool. Empty poolName is equivalent to [NewRedisStream].
func NewRedisStreamForPool(poolName string) (Stream, error) {
	if poolName == "" {
		return NewRedisStream()
	}
	op, err := redis.Pool(poolName)
	if err != nil {
		return nil, err
	}
	if err := op.Ping(); err != nil {
		return nil, err
	}
	return &RedisStream{op: op}, nil
}

func (r *RedisStream) Publish(ctx context.Context, stream string, topic string, payload []byte, ts int64) (string, error) {
	payloadB64 := base64.StdEncoding.EncodeToString(payload)
	reply, err := r.op.XAdd(stream, "*", map[string]interface{}{
		"topic":   topic,
		"payload": payloadB64,
		"ts":      ts,
	})
	if err != nil {
		return "", fmt.Errorf("XADD failed: %w", err)
	}
	return reply, nil
}

func (r *RedisStream) Read(ctx context.Context, streamName string, startID string, blockMs int) ([]Entry, error) {
	if blockMs < 0 {
		blockMs = 0
	}

	reply, err := r.op.XRead(streamName, startID, 100, blockMs)
	if err != nil {
		return nil, err
	}
	if reply == nil {
		return nil, nil
	}

	return parseXReadReply(reply, streamName)
}

func (r *RedisStream) Len(ctx context.Context, stream string) (int, error) {
	return r.op.XLen(stream)
}

func (r *RedisStream) Range(ctx context.Context, stream string, startID, endID string) ([]Entry, error) {
	reply, err := r.op.XRange(stream, startID, endID, 0)
	if err != nil {
		return nil, err
	}
	return parseRangeReply(reply)
}

func (r *RedisStream) GroupCreate(ctx context.Context, stream, group, startID string, mkstream bool) error {
	return r.op.XGroupCreate(stream, group, startID, mkstream)
}

func (r *RedisStream) GroupDestroy(ctx context.Context, stream, group string) (int, error) {
	return r.op.XGroupDestroy(stream, group)
}

func (r *RedisStream) GroupDelConsumer(ctx context.Context, stream, group, consumer string) (int, error) {
	return r.op.XGroupDelConsumer(stream, group, consumer)
}

func (r *RedisStream) ReadGroup(ctx context.Context, streamName, group, consumer, readID string, count int, blockMs int) ([]Entry, error) {
	if blockMs < 0 {
		blockMs = 0
	}
	if count <= 0 {
		count = 100
	}

	reply, err := r.op.XReadGroup(group, consumer, streamName, readID, count, blockMs)
	if err != nil {
		return nil, err
	}
	if reply == nil {
		return nil, nil
	}
	return parseXReadReply(reply, streamName)
}

func (r *RedisStream) Ack(ctx context.Context, stream, group string, messageIDs ...string) (int, error) {
	if len(messageIDs) == 0 {
		return 0, nil
	}
	return r.op.XAck(stream, group, messageIDs...)
}

func (r *RedisStream) Pending(ctx context.Context, stream, group string) (PendingSummary, error) {
	reply, err := r.op.XPending(stream, group)
	if err != nil {
		return PendingSummary{}, err
	}
	if reply == nil {
		return PendingSummary{}, nil
	}
	return parsePendingSummary(reply)
}

func (r *RedisStream) Claim(ctx context.Context, stream, group, consumer string, minIdleMs int, messageIDs ...string) ([]Entry, error) {
	if len(messageIDs) == 0 {
		return nil, fmt.Errorf("claim: at least one message id is required")
	}
	reply, err := r.op.XClaim(stream, group, consumer, minIdleMs, messageIDs...)
	if err != nil {
		return nil, err
	}
	if reply == nil {
		return nil, nil
	}
	return parseRangeReply(reply)
}

func (r *RedisStream) AutoClaim(ctx context.Context, streamName, group, consumer string, minIdleMs int, start string, count int) (AutoClaimResult, error) {
	if start == "" {
		start = "0-0"
	}
	if count <= 0 {
		count = 100
	}
	reply, err := r.op.XAutoClaim(streamName, group, consumer, minIdleMs, start, count)
	if err != nil {
		return AutoClaimResult{}, err
	}
	entries, next, err := parseAutoClaimReply(reply)
	if err != nil {
		return AutoClaimResult{}, err
	}
	return AutoClaimResult{Entries: entries, NextStart: next}, nil
}

func (r *RedisStream) Trim(ctx context.Context, stream string, maxAge time.Duration) (int, error) {
	cutoff := time.Now().Add(-maxAge)
	minID := fmt.Sprintf("%d-0", cutoff.UnixMilli())

	n, err := r.op.XTrimMinID(stream, minID)
	if err == nil {
		return n, nil
	}

	reply, err := r.op.XRange(stream, "-", minID, 0)
	if err != nil {
		return 0, err
	}

	deleted := 0
	for _, item := range reply {
		entry, err := redigo.Values(item, nil)
		if err != nil || len(entry) < 1 {
			continue
		}
		msgID, _ := redigo.String(entry[0], nil)
		if msgID >= minID {
			continue
		}
		n, e := r.op.XDel(stream, msgID)
		if e == nil {
			deleted += n
		}
	}
	return deleted, nil
}

func parsePendingSummary(reply []interface{}) (PendingSummary, error) {
	out := PendingSummary{}
	if len(reply) < 4 {
		return out, nil
	}
	out.Count, _ = redigo.Int64(reply[0], nil)
	out.MinID, _ = redigo.String(reply[1], nil)
	out.MaxID, _ = redigo.String(reply[2], nil)
	consumers, err := redigo.Values(reply[3], nil)
	if err != nil {
		return out, nil
	}
	for _, c := range consumers {
		pair, err := redigo.Values(c, nil)
		if err != nil || len(pair) < 2 {
			continue
		}
		name, _ := redigo.String(pair[0], nil)
		cnt, _ := redigo.Int64(pair[1], nil)
		out.Consumers = append(out.Consumers, PendingConsumer{Name: name, Count: cnt})
	}
	return out, nil
}

func parseXReadReply(reply interface{}, streamName string) ([]Entry, error) {
	top, err := redigo.Values(reply, nil)
	if err != nil {
		return nil, err
	}

	var result []Entry
	for _, item := range top {
		streamBlock, err := redigo.Values(item, nil)
		if err != nil || len(streamBlock) != 2 {
			continue
		}

		name, _ := redigo.String(streamBlock[0], nil)
		if name != streamName {
			continue
		}

		msgs, err := redigo.Values(streamBlock[1], nil)
		if err != nil {
			continue
		}

		for _, m := range msgs {
			ent, err := parseStreamEntry(m)
			if err != nil {
				continue
			}
			result = append(result, ent)
		}
	}
	return result, nil
}

func parseRangeReply(reply []interface{}) ([]Entry, error) {
	var result []Entry
	for _, item := range reply {
		ent, err := parseStreamEntry(item)
		if err != nil {
			continue
		}
		result = append(result, ent)
	}
	return result, nil
}

// parseAutoClaimReply decodes XAUTOCLAIM reply: [next-start, entries]. Redis 7+ may append more fields; entries stay at index 1.
func parseAutoClaimReply(reply interface{}) (entries []Entry, nextStart string, err error) {
	if reply == nil {
		return nil, "0-0", nil
	}
	vals, err := redigo.Values(reply, nil)
	if err != nil {
		return nil, "", fmt.Errorf("xautoclaim reply: %w", err)
	}
	if len(vals) < 2 {
		return nil, "", fmt.Errorf("xautoclaim reply: expected >=2 elements, got %d", len(vals))
	}
	nextStart, err = redigo.String(vals[0], nil)
	if err != nil {
		return nil, "", fmt.Errorf("xautoclaim next id: %w", err)
	}
	if vals[1] == nil {
		return nil, nextStart, nil
	}
	entryVals, err := redigo.Values(vals[1], nil)
	if err != nil {
		return nil, nextStart, fmt.Errorf("xautoclaim entries: %w", err)
	}
	if len(entryVals) == 0 {
		return nil, nextStart, nil
	}
	entries, err = parseRangeReply(entryVals)
	if err != nil {
		return nil, nextStart, err
	}
	return entries, nextStart, nil
}

func parseStreamEntry(item interface{}) (Entry, error) {
	msg, err := redigo.Values(item, nil)
	if err != nil || len(msg) != 2 {
		return Entry{}, fmt.Errorf("invalid entry")
	}

	msgID, _ := redigo.String(msg[0], nil)
	fields, err := redigo.StringMap(msg[1], nil)
	if err != nil {
		return Entry{}, err
	}

	topic := fields["topic"]
	payloadB64 := fields["payload"]
	ts := int64(0)
	if t, ok := fields["ts"]; ok && t != "" {
		ts, _ = strconv.ParseInt(t, 10, 64)
	}

	payload := []byte{}
	if payloadB64 != "" {
		payload, _ = base64.StdEncoding.DecodeString(payloadB64)
	}

	return Entry{
		ID:        msgID,
		Topic:     topic,
		Payload:   payload,
		Timestamp: ts,
	}, nil
}
