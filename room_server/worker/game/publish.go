package game

import (
	"github.com/CryptoElementals/common/log"
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
	if evt == nil {
		return
	}
	if _, err := g.publisher.Publish(g.ctx, &proto.PublishRequest{Topic: topic, Event: evt}); err != nil {
		log.Errorw("publish to player failed", "topic", topic, "eventType", evt.Type.String(), "err", err)
	}
}

// notifyPlayerGameCreated publishes TYPE_MATCHED when a game is matched (skipped for continue games).
func (r *GameManager) notifyPlayerGameCreated(player types.PlayerAddress, evt *types.GameCreatedEvent) {
	if evt.IsContinueGame {
		return
	}
	players := make([]*proto.PlayerAddress, 0, len(evt.Players))
	for _, pl := range evt.Players {
		players = append(players, pl.ToProto())
	}
	if _, err := r.publisher.Publish(r.ctx, &proto.PublishRequest{
		Topic: (&player).String(),
		Event: &proto.Event{
			Type: proto.EventType_TYPE_MATCHED,
			Event: &proto.Event_GameMatched{
				GameMatched: &proto.GameMatched{
					GameId:              uint32(evt.GameID),
					Players:             players,
					ConfirmationTimeout: evt.ConfirmationTimeout,
				},
			},
		},
	}); err != nil {
		log.Errorw("publish game matched failed", "topic", (&player).String(), "err", err)
	}
}

func (r *GameManager) syncGamePhasePublish(address types.PlayerAddress, gamePhase *proto.GamePhase) error {
	_, err := r.publisher.Publish(r.ctx, &proto.PublishRequest{
		Topic: (&address).String(),
		Event: &proto.Event{
			Type: proto.EventType_TYPE_GAME_PHASE_SYNC,
			Event: &proto.Event_GamePhase{
				GamePhase: gamePhase,
			},
		},
	})
	return err
}
