package server

import (
	"context"

	"github.com/CryptoElementals/common/rpc/proto"
	"google.golang.org/protobuf/types/known/emptypb"
)

type Rpc struct {
	proto.UnimplementedRpcServiceServer
	chainHandler  ChainRequestHandler
	playerHandler PlayerRequestHandler
}

func NewRpc(
	chainHandler ChainRequestHandler,
	playerHandler PlayerRequestHandler,
) *Rpc {
	return &Rpc{
		chainHandler:  chainHandler,
		playerHandler: playerHandler,
	}
}

func (s *Rpc) JoinQueue(ctx context.Context, req *proto.PlayerAddress) (*emptypb.Empty, error) {
	return &emptypb.Empty{}, s.playerHandler.JoinQueue(req)
}

func (s *Rpc) ExitQueue(ctx context.Context, req *proto.PlayerAddress) (*emptypb.Empty, error) {
	return &emptypb.Empty{}, s.playerHandler.ExitQueue(req)
}

func (s *Rpc) GetGamePhase(ctx context.Context, req *proto.PlayerAddress) (*proto.GamePhase, error) {
	return s.playerHandler.GetGamePhase(req)
}

func (s *Rpc) GetBattleInfo(ctx context.Context, req *proto.GetBattleInfoRequest) (*proto.GetBattleInfoResponse, error) {
	return s.playerHandler.GetBattleInfo(ctx, req)
}

func (s *Rpc) ConfirmBattle(ctx context.Context, req *proto.ConfirmBattleRequest) (*emptypb.Empty, error) {
	return &emptypb.Empty{}, s.playerHandler.ConfirmBattle(req)
}

func (s *Rpc) ContinueGame(ctx context.Context, req *proto.ContinueGameRequest) (*emptypb.Empty, error) {
	return &emptypb.Empty{}, s.playerHandler.ContinueGame(req)
}

func (s *Rpc) RefuseContinueGame(ctx context.Context, req *proto.RefuseContinueGameRequest) (*emptypb.Empty, error) {
	return &emptypb.Empty{}, s.playerHandler.RefuseContinueGame(req)
}

func (s *Rpc) SubmitTransactions(ctx context.Context, req *proto.TransactionBatch) (*emptypb.Empty, error) {
	return &emptypb.Empty{}, s.chainHandler.SubmitTransactions(req)
}

func (s *Rpc) GetPlayerToken(ctx context.Context, req *proto.GetPlayerTokenRequest) (*proto.GetPlayerTokenResponse, error) {
	return s.playerHandler.GetPlayerToken(req)
}

func (s *Rpc) IsPlayerInQueue(ctx context.Context, req *proto.PlayerAddress) (*proto.IsPlayerInQueueResponse, error) {
	return s.playerHandler.IsPlayerInQueue(req)
}

func (s *Rpc) GetPlayerStatus(ctx context.Context, req *proto.PlayerAddress) (*proto.GetPlayerStatusResponse, error) {
	return s.playerHandler.GetPlayerStatus(req)
}

func (s *Rpc) Surrender(ctx context.Context, req *proto.SurrenderRequest) (*emptypb.Empty, error) {
	return &emptypb.Empty{}, s.playerHandler.Surrender(req)
}

func (s *Rpc) GetGameTimeoutConfig(context.Context, *emptypb.Empty) (*proto.TimeoutConfig, error) {
	return s.playerHandler.GetTimeoutConfig()
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

type PlayerRequestHandler interface {
	JoinQueue(req *proto.PlayerAddress) error
	ExitQueue(req *proto.PlayerAddress) error
	RefuseContinueGame(req *proto.RefuseContinueGameRequest) error
	ContinueGame(req *proto.ContinueGameRequest) error
	ConfirmBattle(req *proto.ConfirmBattleRequest) error
	Surrender(req *proto.SurrenderRequest) error

	IsPlayerInQueue(req *proto.PlayerAddress) (*proto.IsPlayerInQueueResponse, error)
	GetGamePhase(req *proto.PlayerAddress) (*proto.GamePhase, error)
	GetPlayerStatus(req *proto.PlayerAddress) (*proto.GetPlayerStatusResponse, error)
	GetBattleInfo(ctx context.Context, req *proto.GetBattleInfoRequest) (*proto.GetBattleInfoResponse, error)
	GetPlayerToken(req *proto.GetPlayerTokenRequest) (*proto.GetPlayerTokenResponse, error)
	GetTimeoutConfig() (*proto.TimeoutConfig, error)

	SubmitPlayerCommitment(req *proto.SubmitPlayerCommitmentRequest) error
	SubmitPlayerCard(req *proto.SubmitPlayerCardRequest) error
}
