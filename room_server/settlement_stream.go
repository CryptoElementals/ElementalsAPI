package roomserver

import (
	"context"
	"fmt"

	"github.com/CryptoElementals/common/pubsub"
	"github.com/CryptoElementals/common/room_server/worker/game"
	"github.com/CryptoElementals/common/room_server/worker/types"
	"github.com/CryptoElementals/common/rpc/proto"
)

// settlementStreamPublisher notifies the lobby via the room Pub/Sub stream ([pubsub.TopicRoomSettlementPVP]) when a game ends.
type settlementStreamPublisher struct {
	ctx context.Context
	pub game.Publisher
}

func newSettlementStreamPublisher(ctx context.Context, pub game.Publisher) *settlementStreamPublisher {
	return &settlementStreamPublisher{ctx: ctx, pub: pub}
}

func (p *settlementStreamPublisher) GameResultSettlement(evt *types.GameCompletedEvent) error {
	if p.pub == nil || evt == nil {
		return nil
	}
	out := &proto.Event{
		Type: proto.EventType_TYPE_GAME_COMPLETED,
		Event: &proto.Event_GameCompletedNotice{
			GameCompletedNotice: &proto.GameCompletedNotice{GameId: evt.GameID},
		},
	}
	switch evt.GameInfo.Type {
	case types.GameTypePVP:
		return pubsub.Publish(p.ctx, p.pub, pubsub.TopicRoomSettlementPVP, out)
	case types.GameTypeTournament:
		return pubsub.Publish(p.ctx, p.pub, pubsub.TopicRoomSettlementTournament, out)
	}
	return fmt.Errorf("unknown game type: %d", evt.GameInfo.Type)
}

var _ game.GameResultSettler = (*settlementStreamPublisher)(nil)
