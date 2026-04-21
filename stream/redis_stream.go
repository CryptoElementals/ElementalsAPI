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

type RedisStream struct{}

// NewRedisStream returns a Redis-backed Stream. Requires redis.Init to be called first.
func NewRedisStream() (Stream, error) {
	if err := redis.Ping(); err != nil {
		return nil, err
	}
	return &RedisStream{}, nil
}

func (r *RedisStream) Publish(ctx context.Context, stream string, topic string, payload []byte, ts int64) (string, error) {
	payloadB64 := base64.StdEncoding.EncodeToString(payload)
	reply, err := redis.XAdd(stream, "*", map[string]interface{}{
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

	reply, err := redis.XRead(streamName, startID, 100, blockMs)
	if err != nil {
		return nil, err
	}
	if reply == nil {
		return nil, nil
	}

	return parseXReadReply(reply, streamName)
}

func (r *RedisStream) Len(ctx context.Context, stream string) (int, error) {
	return redis.XLen(stream)
}

func (r *RedisStream) Range(ctx context.Context, stream string, startID, endID string) ([]Entry, error) {
	reply, err := redis.XRange(stream, startID, endID, 0)
	if err != nil {
		return nil, err
	}
	return parseRangeReply(reply)
}

func (r *RedisStream) GroupCreate(ctx context.Context, stream, group, startID string, mkstream bool) error {
	return redis.XGroupCreate(stream, group, startID, mkstream)
}

func (r *RedisStream) GroupDestroy(ctx context.Context, stream, group string) (int, error) {
	return redis.XGroupDestroy(stream, group)
}

func (r *RedisStream) GroupDelConsumer(ctx context.Context, stream, group, consumer string) (int, error) {
	return redis.XGroupDelConsumer(stream, group, consumer)
}

func (r *RedisStream) ReadGroup(ctx context.Context, streamName, group, consumer, readID string, count int, blockMs int) ([]Entry, error) {
	if blockMs < 0 {
		blockMs = 0
	}
	if count <= 0 {
		count = 100
	}

	reply, err := redis.XReadGroup(group, consumer, streamName, readID, count, blockMs)
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
	return redis.XAck(stream, group, messageIDs...)
}

func (r *RedisStream) Pending(ctx context.Context, stream, group string) (PendingSummary, error) {
	reply, err := redis.XPending(stream, group)
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
	reply, err := redis.XClaim(stream, group, consumer, minIdleMs, messageIDs...)
	if err != nil {
		return nil, err
	}
	if reply == nil {
		return nil, nil
	}
	return parseRangeReply(reply)
}

func (r *RedisStream) Trim(ctx context.Context, stream string, maxAge time.Duration) (int, error) {
	cutoff := time.Now().Add(-maxAge)
	minID := fmt.Sprintf("%d-0", cutoff.UnixMilli())

	n, err := redis.XTrimMinID(stream, minID)
	if err == nil {
		return n, nil
	}

	reply, err := redis.XRange(stream, "-", minID, 0)
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
		n, e := redis.XDel(stream, msgID)
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
