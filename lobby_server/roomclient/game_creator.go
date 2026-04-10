package roomclient

import (
	"context"

	"github.com/CryptoElementals/common/room_server/worker/types"
	"github.com/CryptoElementals/common/rpc/proto"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// GameCreator implements queue.GameCreator and tournament.GameCreator via RoomService.
type GameCreator struct {
	Client proto.RoomServiceClient
}

func (c *GameCreator) CreateGameAndRun(players []types.PlayerAddress, gameType uint, completedMatchID int64) (int64, error) {
	if c.Client == nil {
		return 0, status.Error(codes.Unavailable, "room worker client not configured")
	}
	req := &proto.CreateGameAndRunRequest{
		GameType:         uint32(gameType),
		CompletedMatchId: completedMatchID,
	}
	for i := range players {
		p := players[i]
		req.Players = append(req.Players, (&p).ToProto())
	}
	resp, err := c.Client.CreateGameAndRun(context.Background(), req)
	if err != nil {
		return 0, err
	}
	return resp.GameId, nil
}
