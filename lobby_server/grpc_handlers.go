package lobbyserver

import (
	"context"

	"github.com/CryptoElementals/common/lobby_server/worker/queue"
	tournament "github.com/CryptoElementals/common/lobby_server/worker/tournament"
	"github.com/CryptoElementals/common/log"
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

// HandleGameCompletedFromRoom runs queue settlement after the room publishes TYPE_GAME_COMPLETED (game id only).
func (s *GRPCServices) HandleGameCompletedFromRoom(gameID int64) error {
	if gameID == 0 {
		return nil
	}
	if err := s.queueSvc.GameResultSettlement(&types.GameCompletedEvent{GameID: gameID, GameType: types.GameTypePVP}); err != nil {
		return err
	}
	if s.tournamentSvc == nil {
		return nil
	}

	return nil
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

// HandleGameCompletedFromTournamentStream runs tournament settlement (game id only).
func (s *GRPCServices) HandleGameCompletedFromTournamentStream(gameID int64) error {
	if gameID == 0 {
		return nil
	}
	log.Debugw("lobby: handle game completed from tournament stream", "game_id", gameID)
	return s.tournamentSvc.GameResultSettlementHook(&types.GameCompletedEvent{GameID: gameID, GameType: types.GameTypeTournament})
}

func (s *GRPCServices) SetTournamentScheduling(ctx context.Context, req *proto.SetTournamentSchedulingRequest) (*proto.TournamentSchedulingStatus, error) {
	_ = ctx
	if s.tournamentSvc == nil {
		return nil, status.Error(codes.FailedPrecondition, "tournament service not initialized")
	}
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "nil request")
	}
	s.tournamentSvc.SetTournamentSchedulingEnabled(req.GetEnabled())
	return &proto.TournamentSchedulingStatus{
		Enabled: s.tournamentSvc.IsTournamentSchedulingEnabled(),
	}, nil
}

func (s *GRPCServices) GetTournamentSchedulingStatus(ctx context.Context, _ *emptypb.Empty) (*proto.TournamentSchedulingStatus, error) {
	_ = ctx
	if s.tournamentSvc == nil {
		return nil, status.Error(codes.FailedPrecondition, "tournament service not initialized")
	}
	return &proto.TournamentSchedulingStatus{
		Enabled: s.tournamentSvc.IsTournamentSchedulingEnabled(),
	}, nil
}
