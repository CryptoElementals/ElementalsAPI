package lobbyserver

import (
	"context"
	"time"

	"github.com/CryptoElementals/common/log"
	"github.com/CryptoElementals/common/pubsub"
	"github.com/CryptoElementals/common/rpc/proto"
)

// runRoomSettlementSubscriber reads [pubsub.TopicRoomSettlement] from Redis (same cluster as room server).
func (s *Service) runRoomSettlementSubscriber() {
	go func() {
		for {
			if s.ctx.Err() != nil {
				return
			}
			err := s.readRoomSettlementStream(s.ctx)
			if s.ctx.Err() != nil {
				return
			}
			if err != nil {
				log.Warnw("lobby: room settlement stream ended", "err", err)
			}
			select {
			case <-s.ctx.Done():
				return
			case <-time.After(2 * time.Second):
			}
		}
	}()
}

func (s *Service) readRoomSettlementStream(ctx context.Context) error {
	if s.eventStream == nil {
		return nil
	}
	sub := pubsub.NewStreamSubscriber(s.eventStream)
	msgCh, stop, err := sub.Subscribe(ctx, pubsub.TopicRoomSettlement, pubsub.SubscribeOptions{})
	if err != nil {
		return err
	}
	defer stop()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case msg, ok := <-msgCh:
			if !ok {
				return nil
			}
			ev := msg.GetEvent()
			if ev == nil || ev.GetType() != proto.EventType_TYPE_GAME_COMPLETED {
				continue
			}
			notice := ev.GetGameCompletedNotice()
			if notice == nil {
				continue
			}
			gameID := notice.GetGameId()
			if err := s.grpcHandlers.HandleGameCompletedFromRoom(gameID); err != nil {
				log.Errorw("lobby: game completed settlement failed", "game_id", gameID, "err", err)
			}
		}
	}
}
