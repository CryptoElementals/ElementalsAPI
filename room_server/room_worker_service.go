package roomserver

import (
	"context"

	"github.com/CryptoElementals/common/room_server/worker/game"
	"github.com/CryptoElementals/common/room_server/worker/types"
	"github.com/CryptoElementals/common/rpc/proto"
)

type roomWorkerService struct {
	proto.UnimplementedRoomWorkerServiceServer
	game *game.Service
}

func newRoomWorkerService(g *game.Service) *roomWorkerService {
	return &roomWorkerService{game: g}
}

func (s *roomWorkerService) CreatePvpGameAfterQueueConfirm(ctx context.Context, req *proto.CreatePvpGameAfterQueueConfirmRequest) (*proto.CreatePvpGameAfterQueueConfirmResponse, error) {
	players := make([]types.PlayerAddress, 0, len(req.GetPlayers()))
	for _, p := range req.GetPlayers() {
		var a types.PlayerAddress
		a.FromProto(p)
		players = append(players, a)
	}
	gid, err := s.game.CreatePvpGameAfterQueueConfirm(players, uint(req.GetGameType()), req.GetCompletedMatchId())
	if err != nil {
		return nil, err
	}
	return &proto.CreatePvpGameAfterQueueConfirmResponse{GameId: uint32(gid)}, nil
}

func (s *roomWorkerService) HandleGameMatchedEvent(ctx context.Context, req *proto.HandleGameMatchedEventRequest) (*proto.HandleGameMatchedEventResponse, error) {
	evt := &types.GameMatchedEvent{
		ConfirmationTimeout: req.GetConfirmationTimeout(),
		GameType:            uint(req.GetGameType()),
	}
	for _, p := range req.GetPlayers() {
		var a types.PlayerAddress
		a.FromProto(p)
		evt.Players = append(evt.Players, a)
	}
	gid, err := s.game.HandleGameMatchedEvent(evt)
	if err != nil {
		return nil, err
	}
	return &proto.HandleGameMatchedEventResponse{GameId: uint32(gid)}, nil
}

func (s *roomWorkerService) GetPlayerGameStatus(ctx context.Context, req *proto.PlayerAddress) (*proto.GetPlayerGameStatusResponse, error) {
	var addr types.PlayerAddress
	addr.FromProto(req)
	st := s.game.GetPlayerGameInfo(addr)
	return &proto.GetPlayerGameStatusResponse{Status: st}, nil
}
