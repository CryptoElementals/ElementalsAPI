package lobbyserver

import (
	"context"
	"fmt"

	"github.com/CryptoElementals/common/db"
	"github.com/CryptoElementals/common/lobby_server/worker/queue"
	tournament "github.com/CryptoElementals/common/lobby_server/worker/tournament"
	"github.com/CryptoElementals/common/room_server/worker/types"
	"github.com/CryptoElementals/common/rpc/proto"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
)

// GRPCServices implements LobbyService.
type GRPCServices struct {
	proto.UnimplementedLobbyServiceServer

	queueSvc      *queue.Service
	tournamentSvc *tournament.TournamentQueueService
	roomWorker    proto.RoomServiceClient
}

func NewGRPCServices(q *queue.Service, t *tournament.TournamentQueueService, rw proto.RoomServiceClient) *GRPCServices {
	return &GRPCServices{queueSvc: q, tournamentSvc: t, roomWorker: rw}
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
	if s.queueSvc.IsPlayerInGame(addr) {
		return nil, status.Error(codes.FailedPrecondition, "player already in game")
	}
	return &emptypb.Empty{}, s.queueSvc.HandleJoinQueueEvent(req)
}

func (s *GRPCServices) ExitQueue(ctx context.Context, req *proto.PlayerAddress) (*emptypb.Empty, error) {
	return &emptypb.Empty{}, s.queueSvc.HandleExitQueueEvent(req)
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
		return &proto.GetPlayerStatusResponse{Status: proto.PlayerStatus_PLAYER_PENDING_QUEUE_MATCH}, nil
	}
	if s.queueSvc.IsPlayerInGame(addr) {
		return &proto.GetPlayerStatusResponse{Status: proto.PlayerStatus_PLAYER_IN_GAME}, nil
	}
	return &proto.GetPlayerStatusResponse{Status: proto.PlayerStatus_PLAYER_UNKNOWN}, nil
}

func (s *GRPCServices) GetPlayerToken(ctx context.Context, req *proto.GetPlayerTokenRequest) (*proto.GetPlayerTokenResponse, error) {
	return s.queueSvc.GetPlayerToken(req.Id)
}

func (s *GRPCServices) RegisterBots(ctx context.Context, req *proto.RegisterBotsForLobbyRequest) (*emptypb.Empty, error) {
	addrs := playerAddressesFromRegisterBotsRequest(req)
	return &emptypb.Empty{}, s.queueSvc.RegisterBots(addrs...)
}

func (s *GRPCServices) UnregisterBots(ctx context.Context, req *proto.RegisterBotsForLobbyRequest) (*emptypb.Empty, error) {
	addrs := playerAddressesFromRegisterBotsRequest(req)
	return &emptypb.Empty{}, s.queueSvc.UnregisterBots(addrs...)
}

// HandleGameCompletedFromRoom runs queue and tournament settlement after the room publishes TYPE_GAME_COMPLETED (game id only; full row loaded here).
func (s *GRPCServices) HandleGameCompletedFromRoom(gameID int64) error {
	if gameID == 0 {
		return nil
	}
	g, err := db.LoadGameByGameID(gameID)
	if err != nil {
		return fmt.Errorf("load game %d: %w", gameID, err)
	}
	ev := &types.GameCompletedEvent{GameID: gameID, GameInfo: g}
	if err := s.queueSvc.GameResultSettlement(ev); err != nil {
		return err
	}
	if s.tournamentSvc == nil {
		return nil
	}
	return s.tournamentSvc.GameResultSettlementHook(ev)
}

func playerAddressesFromRegisterBotsRequest(req *proto.RegisterBotsForLobbyRequest) []*types.PlayerAddress {
	if req == nil {
		return nil
	}
	out := make([]*types.PlayerAddress, 0, len(req.Addresses))
	for _, p := range req.Addresses {
		var a types.PlayerAddress
		a.FromProto(p)
		out = append(out, &a)
	}
	return out
}

func (s *GRPCServices) JoinTournament(ctx context.Context, req *proto.JoinTournamentRequest) (*emptypb.Empty, error) {
	return &emptypb.Empty{}, s.tournamentSvc.HandleJoinTournamentEvent(req.TournamentID, req.PlayerAddress)
}
