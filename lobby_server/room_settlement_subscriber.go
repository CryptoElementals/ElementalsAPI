package lobbyserver

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/CryptoElementals/common/log"
	"github.com/CryptoElementals/common/pubsub"
	"github.com/CryptoElementals/common/rpc/proto"
	"github.com/CryptoElementals/common/stream"
	goproto "google.golang.org/protobuf/proto"
)

// Redis consumer group names for multi-lobby settlement (one group per stream key).
const (
	settlementGroupPVP        = "lobby_room_settlement_pvp"
	settlementGroupTournament = "lobby_room_settlement_tournament"
)

const (
	settlementReadCount = 32
	settlementBlockMS   = 1000
	handlerErrBackoff   = time.Second
	readErrBackoff      = 500 * time.Millisecond
	// settlementAutoClaimMinIdleMS is the minimum time a message may sit in another consumer's PEL before XAUTOCLAIM steals it (dead consumer / stuck peer).
	settlementAutoClaimMinIdleMS = 2_000
)

// runRoomSettlementSubscriber reads settlement streams via Redis consumer groups so each
// message is handled by at most one lobby replica (XREADGROUP + XACK).
func (s *Service) runRoomSettlementSubscriber() {
	consumer := settlementConsumerName()
	go s.runSettlementStreamLoop(pubsub.TopicRoomSettlementPVP, settlementGroupPVP, consumer, s.grpcHandlers.HandleGameCompletedFromRoom, 2*time.Second)

	go s.runSettlementStreamLoop(pubsub.TopicRoomSettlementTournament, settlementGroupTournament, consumer, s.grpcHandlers.HandleGameCompletedFromTournamentStream, time.Second)
}

func settlementConsumerName() string {
	host, err := os.Hostname()
	if err != nil || host == "" {
		host = "unknown"
	}
	return fmt.Sprintf("%s-%d", host, os.Getpid())
}

func ensureSettlementConsumerGroup(ctx context.Context, st stream.Stream, streamKey, group string) error {
	err := st.GroupCreate(ctx, streamKey, group, "0", true)
	if err == nil {
		return nil
	}
	if strings.Contains(strings.ToLower(err.Error()), "busygroup") {
		return nil
	}
	return fmt.Errorf("xgroup create %s %s: %w", streamKey, group, err)
}

func (s *Service) runSettlementStreamLoop(streamKey, group, consumer string, handle func(int64) error, restartDelay time.Duration) {
	for {
		if s.ctx.Err() != nil {
			return
		}
		if s.eventStream == nil {
			return
		}
		err := ensureSettlementConsumerGroup(s.ctx, s.eventStream, streamKey, group)
		if err != nil {
			log.Errorw("lobby: settlement consumer group create failed", "stream", streamKey, "err", err)
			select {
			case <-s.ctx.Done():
				return
			case <-time.After(restartDelay):
			}
			continue
		}
		err = s.readSettlementGroupLoop(s.ctx, streamKey, group, consumer, handle)
		if s.ctx.Err() != nil {
			return
		}
		if err != nil {
			log.Warnw("lobby: settlement group read loop ended", "stream", streamKey, "err", err)
		}
		select {
		case <-s.ctx.Done():
			return
		case <-time.After(restartDelay):
		}
	}
}

// readSettlementGroupLoop reads pending for this consumer ("0"), then idle pending from others (XAUTOCLAIM), then new (">") until ctx done.
func (s *Service) readSettlementGroupLoop(ctx context.Context, streamKey, group, consumer string, handle func(int64) error) error {
	claimCursor := "0-0"
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		entries, err := s.eventStream.ReadGroup(ctx, streamKey, group, consumer, "0", settlementReadCount, 0)
		if err != nil {
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(readErrBackoff):
			}
			continue
		}
		if len(entries) == 0 {
			ac, acErr := s.eventStream.AutoClaim(ctx, streamKey, group, consumer, settlementAutoClaimMinIdleMS, claimCursor, settlementReadCount)
			if acErr != nil {
				log.Warnw("lobby: settlement xautoclaim failed", "stream", streamKey, "err", acErr)
			} else {
				entries = ac.Entries
				if ac.NextStart != "" {
					claimCursor = ac.NextStart
				}
				if len(entries) > 0 {
					log.Infow("lobby: settlement reclaimed idle pending", "stream", streamKey, "count", len(entries))
				}
			}
		}
		if len(entries) == 0 {
			entries, err = s.eventStream.ReadGroup(ctx, streamKey, group, consumer, ">", settlementReadCount, settlementBlockMS)
			if err != nil {
				select {
				case <-ctx.Done():
					return ctx.Err()
				case <-time.After(readErrBackoff):
				}
				continue
			}
		}

		for _, e := range entries {
			ack, errProc := processSettlementEntry(e, handle)
			if !ack {
				log.Errorw("lobby: settlement handler failed", "stream", streamKey, "msg_id", e.ID, "err", errProc)
				select {
				case <-ctx.Done():
					return ctx.Err()
				case <-time.After(handlerErrBackoff):
				}
				break
			}
			if _, err := s.eventStream.Ack(ctx, streamKey, group, e.ID); err != nil {
				log.Errorw("lobby: settlement xack failed", "stream", streamKey, "msg_id", e.ID, "err", err)
			}
		}
	}
}

// processSettlementEntry runs handle for TYPE_GAME_COMPLETED. Returns (ack, err): ack false if handler failed (do not ack);
// ack true if the entry should be acknowledged (success, irrelevant, or unrecoverable bad payload).
func processSettlementEntry(e stream.Entry, handle func(int64) error) (ack bool, err error) {
	if len(e.Payload) == 0 {
		return true, nil
	}
	var ev proto.Event
	if err := goproto.Unmarshal(e.Payload, &ev); err != nil {
		log.Warnw("lobby: settlement unmarshal failed", "msg_id", e.ID, "err", err)
		return true, nil
	}
	if ev.GetType() != proto.EventType_TYPE_GAME_COMPLETED {
		return true, nil
	}
	notice := ev.GetGameCompletedNotice()
	if notice == nil {
		return true, nil
	}
	gameID := notice.GetGameId()
	if gameID == 0 {
		return true, nil
	}
	if err := handle(gameID); err != nil {
		return false, err
	}
	return true, nil
}
