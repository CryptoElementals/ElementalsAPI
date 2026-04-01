package lobbyserver

import (
	"context"
	"fmt"

	"github.com/CryptoElementals/common/db"
	"github.com/CryptoElementals/common/lobby_server/worker/queue"
	"github.com/CryptoElementals/common/lobby_server/worker/turnament"
	"github.com/CryptoElementals/common/room_server/worker/types"
	"github.com/CryptoElementals/common/rpc/proto"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
)

// GRPCServices implements LobbyService and LobbySettlementService.
type GRPCServices struct {
	proto.UnimplementedLobbyServiceServer
	proto.UnimplementedLobbySettlementServiceServer

	queueSvc   *queue.Service
	tournSvc   *turnament.TournamentQueueService
	roomWorker proto.RoomWorkerServiceClient
}

func NewGRPCServices(q *queue.Service, t *turnament.TournamentQueueService, rw proto.RoomWorkerServiceClient) *GRPCServices {
	return &GRPCServices{queueSvc: q, tournSvc: t, roomWorker: rw}
}

func (s *GRPCServices) JoinQueue(ctx context.Context, req *proto.PlayerAddress) (*emptypb.Empty, error) {
	var addr types.PlayerAddress
	addr.FromProto(req)
	if s.queueSvc.IsPlayerInQueue(addr) {
		return nil, status.Error(codes.AlreadyExists, "player already in queue")
	}
	if s.queueSvc.IsPlayerPendingMatch(addr) {
		return nil, status.Error(codes.FailedPrecondition, "player pending match confirmation")
	}
	if s.roomWorker != nil {
		gs, err := s.roomWorker.GetPlayerGameStatus(ctx, req)
		if err != nil {
			return nil, err
		}
		if gs != nil && gs.Status != proto.PlayerStatus_PLAYER_UNKNOWN {
			return nil, status.Errorf(codes.FailedPrecondition, "player cannot join queue, status: %s", gs.Status.String())
		}
	}
	return &emptypb.Empty{}, s.queueSvc.HandleJoinQueueEvent(req)
}

func (s *GRPCServices) ExitQueue(ctx context.Context, req *proto.PlayerAddress) (*emptypb.Empty, error) {
	return &emptypb.Empty{}, s.queueSvc.HandleExitQueueEvent(req)
}

func (s *GRPCServices) IsPlayerInQueue(ctx context.Context, req *proto.PlayerAddress) (*proto.IsPlayerInQueueResponse, error) {
	var addr types.PlayerAddress
	addr.FromProto(req)
	return &proto.IsPlayerInQueueResponse{IsInQueue: s.queueSvc.IsPlayerInQueue(addr)}, nil
}

func (s *GRPCServices) IsPlayerPendingMatch(ctx context.Context, req *proto.PlayerAddress) (*proto.IsPlayerPendingMatchResponse, error) {
	var addr types.PlayerAddress
	addr.FromProto(req)
	return &proto.IsPlayerPendingMatchResponse{Pending: s.queueSvc.IsPlayerPendingMatch(addr)}, nil
}

func (s *GRPCServices) ConfirmMatch(ctx context.Context, req *proto.ConfirmMatchRequest) (*emptypb.Empty, error) {
	return &emptypb.Empty{}, s.queueSvc.HandleConfirmMatch(req)
}

func (s *GRPCServices) CancelMatch(ctx context.Context, req *proto.CancelMatchRequest) (*emptypb.Empty, error) {
	return &emptypb.Empty{}, s.queueSvc.HandleCancelMatch(req)
}

func (s *GRPCServices) GetPlayerStatus(ctx context.Context, req *proto.PlayerAddress) (*proto.GetPlayerStatusResponse, error) {
	var addr types.PlayerAddress
	addr.FromProto(req)
	if s.queueSvc.IsPlayerInQueue(addr) {
		return &proto.GetPlayerStatusResponse{Status: proto.PlayerStatus_PLAYER_IN_QUEUE}, nil
	}
	if s.queueSvc.IsPlayerPendingMatch(addr) {
		return &proto.GetPlayerStatusResponse{Status: proto.PlayerStatus_PLAYER_MATCHED}, nil
	}
	if s.roomWorker == nil {
		return &proto.GetPlayerStatusResponse{Status: proto.PlayerStatus_PLAYER_UNKNOWN}, nil
	}
	gs, err := s.roomWorker.GetPlayerGameStatus(ctx, req)
	if err != nil {
		return nil, err
	}
	if gs == nil {
		return &proto.GetPlayerStatusResponse{Status: proto.PlayerStatus_PLAYER_UNKNOWN}, nil
	}
	return &proto.GetPlayerStatusResponse{Status: gs.Status}, nil
}

func (s *GRPCServices) GetPlayerToken(ctx context.Context, req *proto.GetPlayerTokenRequest) (*proto.GetPlayerTokenResponse, error) {
	return s.queueSvc.GetPlayerToken(req.Id)
}

func (s *GRPCServices) RegisterBots(ctx context.Context, req *proto.RegisterBotsForLobbyRequest) (*emptypb.Empty, error) {
	addrs := make([]*types.PlayerAddress, 0, len(req.Addresses))
	for _, p := range req.Addresses {
		var a types.PlayerAddress
		a.FromProto(p)
		addrs = append(addrs, &a)
	}
	return &emptypb.Empty{}, s.queueSvc.RegisterBots(addrs...)
}

func (s *GRPCServices) UnregisterBots(ctx context.Context, req *proto.RegisterBotsForLobbyRequest) (*emptypb.Empty, error) {
	addrs := make([]*types.PlayerAddress, 0, len(req.Addresses))
	for _, p := range req.Addresses {
		var a types.PlayerAddress
		a.FromProto(p)
		addrs = append(addrs, &a)
	}
	return &emptypb.Empty{}, s.queueSvc.UnregisterBots(addrs...)
}

func (s *GRPCServices) NotifyGameCompleted(ctx context.Context, req *proto.NotifyGameCompletedRequest) (*emptypb.Empty, error) {
	if req == nil {
		return &emptypb.Empty{}, nil
	}
	g, err := db.LoadGameByGameID(uint(req.GameId))
	if err != nil {
		return nil, fmt.Errorf("load game %d: %w", req.GameId, err)
	}
	ev := &types.GameCompletedEvent{GameID: uint(req.GameId), GameInfo: g}
	if err := s.queueSvc.GameResultSettlement(ev); err != nil {
		return nil, err
	}
	if s.tournSvc != nil {
		if err := s.tournSvc.GameResultSettlementHook(ev); err != nil {
			return nil, err
		}
	}
	return &emptypb.Empty{}, nil
}
