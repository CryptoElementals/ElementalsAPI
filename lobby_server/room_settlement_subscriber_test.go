package lobbyserver

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/CryptoElementals/common/pubsub"
	"github.com/CryptoElementals/common/rpc/proto"
	"github.com/CryptoElementals/common/stream"
	goproto "google.golang.org/protobuf/proto"
	"github.com/stretchr/testify/require"
)

// streamStub implements [stream.Stream] for unit tests (unused methods return zero values).
type streamStub struct {
	mu           sync.Mutex
	groupCreateN int
	groupErr     error
}

func (s *streamStub) Publish(ctx context.Context, streamName string, topic string, payload []byte, ts int64) (string, error) {
	return "", nil
}
func (s *streamStub) Read(ctx context.Context, streamName string, startID string, blockMs int) ([]stream.Entry, error) {
	return nil, nil
}
func (s *streamStub) Trim(ctx context.Context, streamName string, maxAge time.Duration) (int, error) {
	return 0, nil
}
func (s *streamStub) Len(ctx context.Context, streamName string) (int, error) {
	return 0, nil
}
func (s *streamStub) Range(ctx context.Context, streamName string, startID, endID string) ([]stream.Entry, error) {
	return nil, nil
}
func (s *streamStub) GroupCreate(ctx context.Context, streamName, group, startID string, mkstream bool) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.groupCreateN++
	if s.groupErr != nil {
		return s.groupErr
	}
	return nil
}
func (s *streamStub) GroupDestroy(ctx context.Context, streamName, group string) (int, error) {
	return 0, nil
}
func (s *streamStub) GroupDelConsumer(ctx context.Context, streamName, group, consumer string) (int, error) {
	return 0, nil
}
func (s *streamStub) ReadGroup(ctx context.Context, streamName, group, consumer, readID string, count int, blockMs int) ([]stream.Entry, error) {
	return nil, nil
}
func (s *streamStub) Ack(ctx context.Context, streamName, group string, messageIDs ...string) (int, error) {
	return 0, nil
}
func (s *streamStub) Pending(ctx context.Context, streamName, group string) (stream.PendingSummary, error) {
	return stream.PendingSummary{}, nil
}
func (s *streamStub) Claim(ctx context.Context, streamName, group, consumer string, minIdleMs int, messageIDs ...string) ([]stream.Entry, error) {
	return nil, nil
}
func (s *streamStub) AutoClaim(ctx context.Context, streamName, group, consumer string, minIdleMs int, start string, count int) (stream.AutoClaimResult, error) {
	return stream.AutoClaimResult{}, nil
}

func TestEnsureSettlementConsumerGroup_BusyGroupIgnored(t *testing.T) {
	ctx := context.Background()
	st := &streamStub{
		groupErr: errors.New("BUSYGROUP Consumer Group name already exists"),
	}
	err := ensureSettlementConsumerGroup(ctx, st, pubsub.TopicRoomSettlementPVP, settlementGroupPVP)
	require.NoError(t, err)
	require.Equal(t, 1, st.groupCreateN)
}

func TestEnsureSettlementConsumerGroup_OtherErrorPropagates(t *testing.T) {
	ctx := context.Background()
	want := errors.New("no connection")
	st := &streamStub{groupErr: want}
	err := ensureSettlementConsumerGroup(ctx, st, "s", "g")
	require.ErrorIs(t, err, want)
}

func TestProcessSettlementEntry_AcksIrrelevant(t *testing.T) {
	calls := 0
	handle := func(int64) error {
		calls++
		return nil
	}
	ack, err := processSettlementEntry(stream.Entry{Payload: []byte("not-proto")}, handle)
	require.NoError(t, err)
	require.True(t, ack)
	require.Zero(t, calls)

	ev := &proto.Event{Type: proto.EventType_TYPE_ROUND_READY}
	b, err := goproto.Marshal(ev)
	require.NoError(t, err)
	ack, err = processSettlementEntry(stream.Entry{Payload: b}, handle)
	require.NoError(t, err)
	require.True(t, ack)
	require.Zero(t, calls)
}

func TestProcessSettlementEntry_HandlerErrorNoAck(t *testing.T) {
	ev := &proto.Event{
		Type: proto.EventType_TYPE_GAME_COMPLETED,
		Event: &proto.Event_GameCompletedNotice{
			GameCompletedNotice: &proto.GameCompletedNotice{GameId: 42},
		},
	}
	b, err := goproto.Marshal(ev)
	require.NoError(t, err)
	want := errors.New("handler failed")
	ack, err := processSettlementEntry(stream.Entry{Payload: b}, func(int64) error { return want })
	require.ErrorIs(t, err, want)
	require.False(t, ack)
}

// fakeConsumerGroupStream models Redis XREADGROUP: each new message (">") is delivered to at most one
// consumer; a second consumer does not see the same message until it is pending-reclaimed (not simulated here).
type fakeConsumerGroupStream struct {
	mu           sync.Mutex
	newMessages  []stream.Entry
	pel          map[string][]stream.Entry // consumer -> pending (undelivered to ack)
	acked        map[string]struct{}
	groupEnsured bool
}

func newFakeConsumerGroupStream(payloads ...[]byte) *fakeConsumerGroupStream {
	entries := make([]stream.Entry, len(payloads))
	for i, p := range payloads {
		entries[i] = stream.Entry{
			ID:        fmt.Sprintf("0-%d", i),
			Topic:     pubsub.TopicRoomSettlementPVP,
			Payload:   p,
			Timestamp: time.Now().UnixMilli(),
		}
	}
	return &fakeConsumerGroupStream{
		newMessages:  entries,
		pel:          make(map[string][]stream.Entry),
		acked:        make(map[string]struct{}),
		groupEnsured: false,
	}
}

func (f *fakeConsumerGroupStream) Publish(ctx context.Context, streamName string, topic string, payload []byte, ts int64) (string, error) {
	return "", nil
}
func (f *fakeConsumerGroupStream) Read(ctx context.Context, streamName string, startID string, blockMs int) ([]stream.Entry, error) {
	return nil, nil
}
func (f *fakeConsumerGroupStream) Trim(ctx context.Context, streamName string, maxAge time.Duration) (int, error) {
	return 0, nil
}
func (f *fakeConsumerGroupStream) Len(ctx context.Context, streamName string) (int, error) {
	return 0, nil
}
func (f *fakeConsumerGroupStream) Range(ctx context.Context, streamName string, startID, endID string) ([]stream.Entry, error) {
	return nil, nil
}
func (f *fakeConsumerGroupStream) GroupCreate(ctx context.Context, streamName, group, startID string, mkstream bool) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.groupEnsured {
		return errors.New("BUSYGROUP Consumer Group name already exists")
	}
	f.groupEnsured = true
	return nil
}
func (f *fakeConsumerGroupStream) GroupDestroy(ctx context.Context, streamName, group string) (int, error) {
	return 0, nil
}
func (f *fakeConsumerGroupStream) GroupDelConsumer(ctx context.Context, streamName, group, consumer string) (int, error) {
	return 0, nil
}
func (f *fakeConsumerGroupStream) ReadGroup(ctx context.Context, streamName, group, consumer, readID string, count int, blockMs int) ([]stream.Entry, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if readID == "0" {
		out := f.pel[consumer]
		f.pel[consumer] = nil
		return out, nil
	}
	if readID != ">" {
		return nil, nil
	}
	if len(f.newMessages) == 0 {
		return nil, nil
	}
	e := f.newMessages[0]
	f.newMessages = f.newMessages[1:]
	f.pel[consumer] = append(f.pel[consumer], e)
	return []stream.Entry{e}, nil
}
func (f *fakeConsumerGroupStream) Ack(ctx context.Context, streamName, group string, messageIDs ...string) (int, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	n := 0
	for _, id := range messageIDs {
		if _, ok := f.acked[id]; ok {
			continue
		}
		f.acked[id] = struct{}{}
		n++
	}
	return n, nil
}
func (f *fakeConsumerGroupStream) Pending(ctx context.Context, streamName, group string) (stream.PendingSummary, error) {
	return stream.PendingSummary{}, nil
}
func (f *fakeConsumerGroupStream) Claim(ctx context.Context, streamName, group, consumer string, minIdleMs int, messageIDs ...string) ([]stream.Entry, error) {
	return nil, nil
}
func (f *fakeConsumerGroupStream) AutoClaim(ctx context.Context, streamName, group, consumer string, minIdleMs int, start string, count int) (stream.AutoClaimResult, error) {
	return stream.AutoClaimResult{}, nil
}

// TestConsumerGroupFake_OneMessageOneConsumer models two lobby replicas: only the first XREADGROUP gets the work item.
func TestConsumerGroupFake_OneMessageOneConsumer(t *testing.T) {
	ctx := context.Background()
	key := pubsub.TopicRoomSettlementPVP
	p := mustMarshalGameCompleted(t, 9001)
	fake := newFakeConsumerGroupStream(p)
	require.NoError(t, ensureSettlementConsumerGroup(ctx, fake, key, settlementGroupPVP))

	c1, c2 := "pod-1", "pod-2"
	got1, err := fake.ReadGroup(ctx, key, settlementGroupPVP, c1, ">", 10, 0)
	require.NoError(t, err)
	got2, err := fake.ReadGroup(ctx, key, settlementGroupPVP, c2, ">", 10, 0)
	require.NoError(t, err)
	require.Len(t, got1, 1)
	require.Empty(t, got2)
}

// TestConsumerGroupFake_TwoMessagesSplit models Redis assigning two new entries across two consumers.
func TestConsumerGroupFake_TwoMessagesSplit(t *testing.T) {
	ctx := context.Background()
	key := pubsub.TopicRoomSettlementPVP
	fake := newFakeConsumerGroupStream(
		mustMarshalGameCompleted(t, 1),
		mustMarshalGameCompleted(t, 2),
	)
	require.NoError(t, fake.GroupCreate(ctx, key, settlementGroupPVP, "0", true))

	a, err := fake.ReadGroup(ctx, key, settlementGroupPVP, "c1", ">", 10, 0)
	require.NoError(t, err)
	b, err := fake.ReadGroup(ctx, key, settlementGroupPVP, "c2", ">", 10, 0)
	require.NoError(t, err)
	require.Len(t, a, 1)
	require.Len(t, b, 1)
}

func mustMarshalGameCompleted(t *testing.T, gameID int64) []byte {
	t.Helper()
	ev := &proto.Event{
		Type: proto.EventType_TYPE_GAME_COMPLETED,
		Event: &proto.Event_GameCompletedNotice{
			GameCompletedNotice: &proto.GameCompletedNotice{GameId: gameID},
		},
	}
	b, err := goproto.Marshal(ev)
	require.NoError(t, err)
	return b
}
