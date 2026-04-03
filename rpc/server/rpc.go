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

type PlayerRequestHandler interface {
	ConfirmBattle(req *proto.ConfirmBattleRequest) error
	Surrender(req *proto.SurrenderRequest) error

	SubmitPlayerCommitment(req *proto.SubmitPlayerCommitmentRequest) error
	SubmitPlayerCard(req *proto.SubmitPlayerCardRequest) error
}
