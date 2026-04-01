package server

import (
	"context"

	"github.com/CryptoElementals/common/rpc/proto"
	"google.golang.org/protobuf/types/known/emptypb"
)

type Rpc struct {
	proto.UnimplementedRpcServiceServer
	chainHandler   ChainRequestHandler
	playerHandler  PlayerRequestHandler
	gamePhase      GamePhaseHandler
}

func NewRpc(
	chainHandler ChainRequestHandler,
	playerHandler PlayerRequestHandler,
	gamePhase GamePhaseHandler,
) *Rpc {
	return &Rpc{
		chainHandler:  chainHandler,
		playerHandler: playerHandler,
		gamePhase:     gamePhase,
	}
}

func (s *Rpc) GetGamePhase(ctx context.Context, req *proto.PlayerAddress) (*proto.GamePhase, error) {
	return s.gamePhase.GetGamePhase(req)
}

func (s *Rpc) GetBattleInfo(ctx context.Context, req *proto.GetBattleInfoRequest) (*proto.GetBattleInfoResponse, error) {
	return s.playerHandler.GetBattleInfo(ctx, req)
}

func (s *Rpc) ConfirmBattle(ctx context.Context, req *proto.ConfirmBattleRequest) (*emptypb.Empty, error) {
	return &emptypb.Empty{}, s.playerHandler.ConfirmBattle(req)
}

func (s *Rpc) SubmitTransactions(ctx context.Context, req *proto.TransactionBatch) (*emptypb.Empty, error) {
	return &emptypb.Empty{}, s.chainHandler.SubmitTransactions(req)
}

func (s *Rpc) Surrender(ctx context.Context, req *proto.SurrenderRequest) (*emptypb.Empty, error) {
	return &emptypb.Empty{}, s.playerHandler.Surrender(req)
}

func (s *Rpc) SubmitPlayerCommitment(ctx context.Context, req *proto.SubmitPlayerCommitmentRequest) (*emptypb.Empty, error) {
	return &emptypb.Empty{}, s.playerHandler.SubmitPlayerCommitment(req)
}

func (s *Rpc) SubmitPlayerCard(ctx context.Context, req *proto.SubmitPlayerCardRequest) (*emptypb.Empty, error) {
	return &emptypb.Empty{}, s.playerHandler.SubmitPlayerCard(req)
}

type ChainRequestHandler interface {
	SubmitTransactions(req *proto.TransactionBatch) error
}

// GamePhaseHandler returns persisted in-game phase from the room worker (no lobby / queue resolution).
type GamePhaseHandler interface {
	GetGamePhase(req *proto.PlayerAddress) (*proto.GamePhase, error)
}

type PlayerRequestHandler interface {
	ConfirmBattle(req *proto.ConfirmBattleRequest) error
	Surrender(req *proto.SurrenderRequest) error

	GetBattleInfo(ctx context.Context, req *proto.GetBattleInfoRequest) (*proto.GetBattleInfoResponse, error)

	SubmitPlayerCommitment(req *proto.SubmitPlayerCommitmentRequest) error
	SubmitPlayerCard(req *proto.SubmitPlayerCardRequest) error
}
