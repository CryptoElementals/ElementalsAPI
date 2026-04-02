package queue

import (
	"context"

	"github.com/CryptoElementals/common/db"
	"github.com/CryptoElementals/common/log"
	dao "github.com/CryptoElementals/common/models"
	"github.com/CryptoElementals/common/pubsub"
	"github.com/CryptoElementals/common/room_server/worker/types"
	"github.com/CryptoElementals/common/rpc/proto"
	"google.golang.org/protobuf/types/known/emptypb"
)

func notifyContinueCanceled(ctx context.Context, pub EventPublisher, player types.PlayerAddress) {
	topic := (&player).String()
	evt := &proto.Event{
		Type: proto.EventType_TYPE_CONTINUE_CANCELED,
		Event: &proto.Event_X{
			X: &emptypb.Empty{},
		},
	}
	if err := pubsub.Publish(ctx, pub, topic, evt); err != nil {
		log.Errorw("notifyContinueCanceled publish failed", "player", topic, "err", err)
	}
}

// availableTokens returns recorded balance minus locked rows (same gate as joining the matchmaking queue).
func availableTokens(ut *dao.UserToken) int {
	var locked int32
	for _, row := range ut.LockedTokens {
		locked += row.TokenAmount
	}
	return int(ut.TokenAmount) - int(locked)
}

// anyHumanPlayerBelowQueueThreshold reports whether any non-bot player cannot afford the queue lock after settlement.
func (q *Queue) anyHumanPlayerBelowQueueThreshold(game *dao.Game, bots Set[types.PlayerAddress]) bool {
	for _, p := range game.Players {
		if p == nil {
			continue
		}
		addr := types.NewPlayerAddress(p.PlayerId, p.TemporaryAddress)
		if bots.Contains(*addr) {
			continue
		}
		ut, err := db.GetPlayerToken(context.Background(), p.PlayerId)
		if err != nil {
			log.Errorw("failed to get player token after settlement", "player_id", p.PlayerId, "err", err)
			continue
		}
		avail := availableTokens(ut)
		if avail < int(q.minTokenToJoinQueue) {
			log.Infow("player doesn't have enough tokens after settlement",
				"player_id", p.PlayerId,
				"available_tokens", avail,
				"required_tokens", q.minTokenToJoinQueue)
			return true
		}
	}
	return false
}

func publishContinueCanceledForHumans(ctx context.Context, pub EventPublisher, game *dao.Game, bots Set[types.PlayerAddress]) {
	for _, p := range game.Players {
		if p == nil {
			continue
		}
		addr := types.NewPlayerAddress(p.PlayerId, p.TemporaryAddress)
		if bots.Contains(*addr) {
			continue
		}
		notifyContinueCanceled(ctx, pub, *addr)
	}
}
