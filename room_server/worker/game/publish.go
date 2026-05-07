package game

import (
	"github.com/CryptoElementals/common/log"
	"github.com/CryptoElementals/common/pubsub"
	"github.com/CryptoElementals/common/room_server/worker/types"
	"github.com/CryptoElementals/common/rpc/proto"
	"google.golang.org/protobuf/types/known/emptypb"
)

func (g *Game) publishProtoToAllPlayers(evt *proto.Event) {
	if evt == nil {
		return
	}
	receivers := make([]*proto.PlayerAddress, 0, len(g.currentRound.gamePlayers))
	for _, p := range g.currentRound.gamePlayers {
		addr := p.PlayerAddress()
		receivers = append(receivers, addr.ToProto())
	}
	evt.Receivers = receivers
	g.publishProto(evt)
}

// publishPartialReadyToOtherPlayers publishes TYPE_PART_CONFIRMED to all players except the one who confirmed.
func (g *Game) publishPartialReadyToOtherPlayers(readyAddress types.PlayerAddress) {
	receivers := make([]*proto.PlayerAddress, 0, len(g.currentRound.gamePlayers))
	for _, p := range g.currentRound.gamePlayers {
		addr := p.PlayerAddress()
		if addr == readyAddress {
			continue
		}
		receivers = append(receivers, addr.ToProto())
	}
	out := &proto.Event{
		Type:      proto.EventType_TYPE_PART_CONFIRMED,
		Receivers: receivers,
		Event: &proto.Event_X{
			X: &emptypb.Empty{},
		},
	}
	g.publishProto(out)
}

func (g *Game) publishProto(evt *proto.Event) {
	if err := pubsub.Publish(g.ctx, g.publisher, evt); err != nil {
		log.Errorw("publish failed", "topic", pubsub.TopicRoom, "eventType", evt.Type.String(), "err", err)
	}
}

func (r *GameManager) syncGamePhasePublish(address types.PlayerAddress, gamePhase *proto.GamePhase) error {
	return pubsub.Publish(r.ctx, r.publisher, &proto.Event{
		Type:      proto.EventType_TYPE_GAME_PHASE_SYNC,
		Receivers: []*proto.PlayerAddress{address.ToProto()},
		Event: &proto.Event_GamePhase{
			GamePhase: gamePhase,
		},
	})
}
