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
	if p.pub == nil || evt == nil || evt.GameID == 0 {
		return nil
	}
	out := &proto.Event{
		Type: proto.EventType_TYPE_GAME_COMPLETED,
		Event: &proto.Event_GameCompletedNotice{
			GameCompletedNotice: &proto.GameCompletedNotice{GameId: evt.GameID},
		},
	}
	gt := evt.GameType
	if gt == proto.GameType_GAME_TYPE_UNKNOWN {
		gt = proto.GameType_PVP
	}
	switch gt {
	case proto.GameType_TOURNAMENT:
		return pubsub.Publish(p.ctx, p.pub, pubsub.TopicRoomSettlementTournament, out)
	case proto.GameType_PVP:
		return pubsub.Publish(p.ctx, p.pub, pubsub.TopicRoomSettlementPVP, out)
	default:
		return fmt.Errorf("unknown game type: %d", evt.GameType)
	}
}

var _ game.GameResultSettler = (*settlementStreamPublisher)(nil)
