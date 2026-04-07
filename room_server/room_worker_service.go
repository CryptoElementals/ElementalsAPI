package roomserver

import (
	"context"

	"github.com/CryptoElementals/common/room_server/worker/game"
	"github.com/CryptoElementals/common/room_server/worker/types"
	"github.com/CryptoElementals/common/rpc/proto"
	"google.golang.org/protobuf/types/known/emptypb"
)

type roomWorkerService struct {
	proto.UnimplementedRoomWorkerServiceServer
	game *game.Service
}

func newRoomWorkerService(g *game.Service) *roomWorkerService {
	return &roomWorkerService{game: g}
}

func (s *roomWorkerService) CreateGameAndRun(ctx context.Context, req *proto.CreateGameAndRunRequest) (*proto.CreateGameAndRunResponse, error) {
	players := make([]types.PlayerAddress, 0, len(req.GetPlayers()))
	for _, p := range req.GetPlayers() {
		var a types.PlayerAddress
		a.FromProto(p)
		players = append(players, a)
	}
	gid, err := s.game.CreateGameAndRun(players, uint(req.GetGameType()), req.GetCompletedMatchId())
	if err != nil {
		return nil, err
	}
	return &proto.CreateGameAndRunResponse{GameId: uint32(gid)}, nil
}

func (s *roomWorkerService) SyncGamePhase(ctx context.Context, req *proto.PlayerAddress) (*emptypb.Empty, error) {
	var addr types.PlayerAddress
	addr.FromProto(req)
	if err := s.game.SyncGamePhase(addr); err != nil {
		return nil, err
	}
	return &emptypb.Empty{}, nil
}
