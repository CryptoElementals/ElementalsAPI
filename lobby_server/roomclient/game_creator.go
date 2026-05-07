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

func (c *GameCreator) CreateGameAndRun(players []types.PlayerAddress, gameType proto.GameType, completedMatchID int64) (int64, error) {
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

func (c *GameCreator) CreatePVPGameAndRun(players []types.PlayerAddress, completedMatchID int64) (int64, error) {
	if c.Client == nil {
		return 0, status.Error(codes.Unavailable, "room worker client not configured")
	}
	req := &proto.CreateGameAndRunRequest{
		GameType:         uint32(proto.GameType_PVP),
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

func (c *GameCreator) CreateTournamentGameAndRun(players []types.PlayerAddress, completedMatchID int64, tournamentID int64, tierNo int64) (int64, error) {
	if c.Client == nil {
		return 0, status.Error(codes.Unavailable, "room worker client not configured")
	}
	req := &proto.CreateGameAndRunRequest{
		GameType:         uint32(proto.GameType_TOURNAMENT),
		CompletedMatchId: completedMatchID,
		TournamentId:     tournamentID,
		TierNo:           tierNo,
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
