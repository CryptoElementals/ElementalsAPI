package lobbyserver

import (
	"context"
	"io"
	"time"

	"github.com/CryptoElementals/common/log"
	"github.com/CryptoElementals/common/pubsub"
	"github.com/CryptoElementals/common/rpc/proto"
)

const roomSettlementSubscriberID = "lobby_room_settlement"

// runRoomSettlementSubscriber subscribes to the room host Pub/Sub topic [pubsub.TopicRoomSettlement] and runs settlement when a game completes.
func (s *Service) runRoomSettlementSubscriber() {
	pc := proto.NewPubSubServiceClient(s.roomConn)
	go func() {
		for {
			err := s.readRoomSettlementStream(s.ctx, pc)
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

func (s *Service) readRoomSettlementStream(ctx context.Context, pc proto.PubSubServiceClient) error {
	stream, err := pc.Subscribe(ctx, &proto.SubscribeRequest{
		Topic:        pubsub.TopicRoomSettlement,
		SubscriberId: roomSettlementSubscriberID,
	})
	if err != nil {
		return err
	}
	for {
		msg, err := stream.Recv()
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return err
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
