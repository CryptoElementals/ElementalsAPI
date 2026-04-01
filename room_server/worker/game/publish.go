package game

import (
	"github.com/CryptoElementals/common/log"
	"github.com/CryptoElementals/common/room_server/worker/protopub"
	"github.com/CryptoElementals/common/room_server/worker/types"
	"github.com/CryptoElementals/common/rpc/proto"
	"google.golang.org/protobuf/types/known/emptypb"
)

func (g *Game) publishProtoToAllPlayers(evt *proto.Event) {
	if evt == nil {
		return
	}
	for _, p := range g.currentRound.gamePlayers {
		addr := p.PlayerAddress()
		g.publishProto((&addr).String(), evt)
	}
}

// publishPartialReadyToOtherPlayers publishes TYPE_PART_CONFIRMED to all players except the one who confirmed.
func (g *Game) publishPartialReadyToOtherPlayers(readyAddress types.PlayerAddress) {
	out := &proto.Event{
		Type: proto.EventType_TYPE_PART_CONFIRMED,
		Event: &proto.Event_X{
			X: &emptypb.Empty{},
		},
	}
	for _, p := range g.currentRound.gamePlayers {
		addr := p.PlayerAddress()
		if addr == readyAddress {
			continue
		}
		g.publishProto((&addr).String(), out)
	}
}

func (g *Game) publishProto(topic string, evt *proto.Event) {
	if err := protopub.Publish(g.ctx, g.publisher, topic, evt); err != nil {
		log.Errorw("publish to player failed", "topic", topic, "eventType", evt.Type.String(), "err", err)
	}
}

func (r *GameManager) syncGamePhasePublish(address types.PlayerAddress, gamePhase *proto.GamePhase) error {
	return protopub.Publish(r.ctx, r.publisher, (&address).String(), &proto.Event{
		Type: proto.EventType_TYPE_GAME_PHASE_SYNC,
		Event: &proto.Event_GamePhase{
			GamePhase: gamePhase,
		},
	})
}
