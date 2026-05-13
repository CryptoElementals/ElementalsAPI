package lobbyserver

import (
	"context"
	"strings"
	"time"

	"github.com/CryptoElementals/common/conversion"
	"github.com/CryptoElementals/common/db"
	"github.com/CryptoElementals/common/lobby_server/worker/queue"
	tournament "github.com/CryptoElementals/common/lobby_server/worker/tournament"
	"github.com/CryptoElementals/common/log"
	dao "github.com/CryptoElementals/common/models"
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
	tournament, st, err := s.tournamentSvc.GetActiveTournamentByPlayer(req)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "tournament status: %v", err)
	}
	if st == dao.TournamentParticipantStatusInProgress || st == dao.TournamentParticipantStatusQueued {
		if tournament.RegistrationDeadline.Before(time.Now().Add(time.Minute * 5)) {
			return nil, status.Error(codes.FailedPrecondition, "player in tournament")
		}
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
	_ = ctx
	if req == nil {
		return &proto.GetPlayerStatusResponse{Status: proto.PlayerStatus_PLAYER_UNKNOWN}, nil
	}
	var addr types.PlayerAddress
	addr.FromProto(req)
	if s.tournamentSvc != nil {
		_, st, err := s.tournamentSvc.GetActiveTournamentByPlayer(req)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "tournament status: %v", err)
		}
		switch st {
		case dao.TournamentParticipantStatusInProgress:
			// Tournament: Status only; omit Detail oneof.
			return &proto.GetPlayerStatusResponse{Status: proto.PlayerStatus_PLAYER_TOURNAMENT_IN_PROGRESS}, nil
		case dao.TournamentParticipantStatusQueued:
			return &proto.GetPlayerStatusResponse{Status: proto.PlayerStatus_PLAYER_TOURNAMENT_QUEUED}, nil
		}
	}
	resp, err := s.queueSvc.GetPlayerStatusResponse(addr)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "queue player status: %v", err)
	}
	return resp, nil
}

func (s *GRPCServices) GetPlayerToken(ctx context.Context, req *proto.GetPlayerTokenRequest) (*proto.GetPlayerTokenResponse, error) {
	return s.queueSvc.GetPlayerToken(req.Id)
}

func (s *GRPCServices) GetOrCreateUserProfileByAddress(ctx context.Context, req *proto.GetOrCreateUserProfileByAddressRequest) (*proto.UserProfileResponse, error) {
	_ = ctx
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "nil request")
	}
	address := strings.ToLower(strings.TrimSpace(req.GetAddress()))
	if address == "" {
		return nil, status.Error(codes.InvalidArgument, "address is empty")
	}
	profile, err := db.GetOrCreateUserProfile(address)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "get or create profile by address failed: %v", err)
	}
	return &proto.UserProfileResponse{
		PlayerID:      profile.PlayerID,
		Address:       profile.Address,
		Email:         profile.Email,
		Name:          profile.Name,
		AvatarURL:     profile.AvatarURL,
		BackgroundURL: profile.BackgroundURL,
	}, nil
}

func (s *GRPCServices) GetOrCreateUserProfileByEmail(ctx context.Context, req *proto.GetOrCreateUserProfileByEmailRequest) (*proto.UserProfileResponse, error) {
	_ = ctx
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "nil request")
	}
	email := strings.TrimSpace(req.GetEmail())
	if email == "" {
		return nil, status.Error(codes.InvalidArgument, "email is empty")
	}
	profile, err := db.GetOrCreateUserProfileByEmail(email, strings.TrimSpace(req.GetName()))
	if err != nil {
		return nil, status.Errorf(codes.Internal, "get or create profile by email failed: %v", err)
	}
	return &proto.UserProfileResponse{
		PlayerID:      profile.PlayerID,
		Address:       profile.Address,
		Email:         profile.Email,
		Name:          profile.Name,
		AvatarURL:     profile.AvatarURL,
		BackgroundURL: profile.BackgroundURL,
	}, nil
}

func (s *GRPCServices) EnsureUserToken(ctx context.Context, req *proto.EnsureUserTokenRequest) (*emptypb.Empty, error) {
	_ = ctx
	if req == nil || req.GetPlayerID() == 0 {
		return nil, status.Error(codes.InvalidArgument, "invalid player id")
	}
	if _, err := db.EnsureUserTokenByPlayerID(req.GetPlayerID()); err != nil {
		return nil, status.Errorf(codes.Internal, "ensure user token failed: %v", err)
	}
	return &emptypb.Empty{}, nil
}

func (s *GRPCServices) CreditUserTokens(ctx context.Context, req *proto.CreditUserTokensRequest) (*proto.GetPlayerTokenResponse, error) {
	_ = ctx
	if req == nil || req.GetPlayerID() == 0 {
		return nil, status.Error(codes.InvalidArgument, "invalid player id")
	}
	userToken, err := db.CreditUserTokenAmount(req.GetPlayerID(), req.GetDelta())
	if err != nil {
		return nil, status.Errorf(codes.Internal, "credit user token failed: %v", err)
	}
	return conversion.DbUserTokenToProtoGetPlayerTokenResponse(userToken), nil
}

func (s *GRPCServices) SetUserTokenAmount(ctx context.Context, req *proto.SetUserTokenAmountRequest) (*proto.GetPlayerTokenResponse, error) {
	_ = ctx
	if req == nil || req.GetPlayerID() == 0 {
		return nil, status.Error(codes.InvalidArgument, "invalid player id")
	}
	userToken, err := db.SetUserTokenAmount(req.GetPlayerID(), req.GetTokenAmount())
	if err != nil {
		return nil, status.Errorf(codes.Internal, "set user token failed: %v", err)
	}
	return conversion.DbUserTokenToProtoGetPlayerTokenResponse(userToken), nil
}

func (s *GRPCServices) CreateBotAccount(ctx context.Context, req *proto.CreateBotAccountRequest) (*proto.UserProfileWithTokenResponse, error) {
	_ = ctx
	if req == nil || req.GetPlayerID() == 0 {
		return nil, status.Error(codes.InvalidArgument, "invalid player id")
	}
	profile, token, err := db.CreateBot(
		req.GetPlayerID(),
		strings.TrimSpace(req.GetName()),
		strings.TrimSpace(req.GetAvatarURL()),
		strings.TrimSpace(req.GetBackgroundURL()),
		req.GetTokenAmount(),
		req.GetPoints(),
	)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "create bot account failed: %v", err)
	}
	return &proto.UserProfileWithTokenResponse{
		Profile: &proto.UserProfileResponse{
			PlayerID:      profile.PlayerID,
			Address:       profile.Address,
			Email:         profile.Email,
			Name:          profile.Name,
			AvatarURL:     profile.AvatarURL,
			BackgroundURL: profile.BackgroundURL,
		},
		Token: conversion.DbUserTokenToProtoGetPlayerTokenResponse(token),
	}, nil
}

// HandleGameCompletedFromRoom runs queue settlement after the room publishes TYPE_GAME_COMPLETED (game id only).
func (s *GRPCServices) HandleGameCompletedFromRoom(gameID int64) error {
	if gameID == 0 {
		return nil
	}
	if err := s.queueSvc.GameResultSettlement(&types.GameCompletedEvent{GameID: gameID, GameType: proto.GameType_PVP}); err != nil {
		return err
	}
	if s.tournamentSvc == nil {
		return nil
	}

	return nil
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
	return s.tournamentSvc.GameResultSettlementHook(&types.GameCompletedEvent{GameID: gameID, GameType: proto.GameType_TOURNAMENT})
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
