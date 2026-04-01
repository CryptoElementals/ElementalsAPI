package roomclient

import (
	"context"

	"github.com/CryptoElementals/common/room_server/worker/types"
	"github.com/CryptoElementals/common/rpc/proto"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// GameCreator implements queue.GameCreator and turnament.GameCreator via RoomWorkerService.
type GameCreator struct {
	Client proto.RoomWorkerServiceClient
}

func (c *GameCreator) CreatePvpGameAfterQueueConfirm(players []types.PlayerAddress, gameType uint, completedMatchID int64) (uint, error) {
	if c.Client == nil {
		return 0, status.Error(codes.Unavailable, "room worker client not configured")
	}
	req := &proto.CreatePvpGameAfterQueueConfirmRequest{
		GameType:         uint32(gameType),
		CompletedMatchId: completedMatchID,
	}
	for i := range players {
		p := players[i]
		req.Players = append(req.Players, (&p).ToProto())
	}
	resp, err := c.Client.CreatePvpGameAfterQueueConfirm(context.Background(), req)
	if err != nil {
		return 0, err
	}
	return uint(resp.GameId), nil
}

func (c *GameCreator) HandleGameMatchedEvent(evt *types.GameMatchedEvent) (uint, error) {
	if c.Client == nil {
		return 0, status.Error(codes.Unavailable, "room worker client not configured")
	}
	req := &proto.HandleGameMatchedEventRequest{
		ConfirmationTimeout: evt.ConfirmationTimeout,
		GameType:            uint32(evt.GameType),
	}
	for i := range evt.Players {
		p := evt.Players[i]
		req.Players = append(req.Players, (&p).ToProto())
	}
	resp, err := c.Client.HandleGameMatchedEvent(context.Background(), req)
	if err != nil {
		return 0, err
	}
	return uint(resp.GameId), nil
}
