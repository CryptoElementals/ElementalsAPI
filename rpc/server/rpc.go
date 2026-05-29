package server

import (
	"context"

	"github.com/CryptoElementals/common/rpc/proto"
	"google.golang.org/protobuf/types/known/emptypb"
)

type Rpc struct {
	proto.UnimplementedRoomServiceServer
	gameHandler GameProcessHandler
}

func NewRpc(gameHandler GameProcessHandler) *Rpc {
	return &Rpc{
		gameHandler:   gameHandler,
	}
}

func (s *Rpc) ConfirmBattle(ctx context.Context, req *proto.ConfirmBattleRequest) (*emptypb.Empty, error) {
	return &emptypb.Empty{}, s.gameHandler.ConfirmBattle(req)
}

func (s *Rpc) SubmitTransactions(ctx context.Context, req *proto.TransactionBatch) (*emptypb.Empty, error) {
	return &emptypb.Empty{}, s.gameHandler.SubmitTransactions(req)
}

func (s *Rpc) Surrender(ctx context.Context, req *proto.SurrenderRequest) (*emptypb.Empty, error) {
	return &emptypb.Empty{}, s.gameHandler.Surrender(req)
}

func (s *Rpc) SubmitPlayerCommitment(ctx context.Context, req *proto.SubmitPlayerCommitmentRequest) (*emptypb.Empty, error) {
	return &emptypb.Empty{}, s.gameHandler.SubmitPlayerCommitment(req)
}

func (s *Rpc) SubmitPlayerCard(ctx context.Context, req *proto.SubmitPlayerCardRequest) (*emptypb.Empty, error) {
	return &emptypb.Empty{}, s.gameHandler.SubmitPlayerCard(req)
}

func (s *Rpc) CreateGameAndRun(ctx context.Context, req *proto.CreateGameAndRunRequest) (*proto.CreateGameAndRunResponse, error) {
	return s.gameHandler.CreateGameAndRunRPC(ctx, req)
}

func (s *Rpc) SyncGamePhase(ctx context.Context, req *proto.PlayerAddress) (*emptypb.Empty, error) {
	return s.gameHandler.SyncGamePhaseRPC(ctx, req)
}

func (s *Rpc) GetGamePhase(ctx context.Context, req *proto.PlayerAddress) (*proto.GamePhase, error) {
	return s.gameHandler.GetGamePhaseRPC(ctx, req)
}

func (s *Rpc) AbortAllActiveGames(ctx context.Context, req *emptypb.Empty) (*proto.AbortAllActiveGamesResponse, error) {
	return s.gameHandler.AbortAllActiveGamesRPC(ctx, req)
}

type GameProcessHandler interface {
	SubmitTransactions(req *proto.TransactionBatch) error
	ConfirmBattle(req *proto.ConfirmBattleRequest) error
	Surrender(req *proto.SurrenderRequest) error

	SubmitPlayerCommitment(req *proto.SubmitPlayerCommitmentRequest) error
	SubmitPlayerCard(req *proto.SubmitPlayerCardRequest) error
	CreateGameAndRunRPC(ctx context.Context, req *proto.CreateGameAndRunRequest) (*proto.CreateGameAndRunResponse, error)
	SyncGamePhaseRPC(ctx context.Context, req *proto.PlayerAddress) (*emptypb.Empty, error)
	GetGamePhaseRPC(ctx context.Context, req *proto.PlayerAddress) (*proto.GamePhase, error)
	AbortAllActiveGamesRPC(ctx context.Context, req *emptypb.Empty) (*proto.AbortAllActiveGamesResponse, error)
}
