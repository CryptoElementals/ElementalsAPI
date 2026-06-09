package chainserver

import (
	"context"
	"errors"

	"github.com/CryptoElementals/common/chain_server/worker"
	"github.com/CryptoElementals/common/rpc/proto"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
)

// GRPCServices implements ChainService.
type GRPCServices struct {
	proto.UnimplementedChainServiceServer

	chain *worker.Chain
}

func NewGRPCServices(chain *worker.Chain) *GRPCServices {
	return &GRPCServices{chain: chain}
}

func (s *GRPCServices) AddCreateRoom(ctx context.Context, req *proto.RequireGameCreationEvent) (*emptypb.Empty, error) {
	_ = ctx
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "nil request")
	}
	s.chain.AddCreateRoom(worker.RequireGameCreationFromProto(req))
	return &emptypb.Empty{}, nil
}

func (s *GRPCServices) AddSetTurnReady(ctx context.Context, req *proto.RequireSetupNewTurnEvent) (*emptypb.Empty, error) {
	_ = ctx
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "nil request")
	}
	s.chain.AddSetTurnReady(worker.RequireSetupNewTurnFromProto(req))
	return &emptypb.Empty{}, nil
}

func (s *GRPCServices) AddCommitment(ctx context.Context, req *proto.SubmitPlayerCommitmentRequest) (*emptypb.Empty, error) {
	_ = ctx
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "nil request")
	}
	if err := s.chain.AddCommitment(req); err != nil {
		return nil, status.Errorf(codes.Internal, "add commitment: %v", err)
	}
	return &emptypb.Empty{}, nil
}

func (s *GRPCServices) AddCard(ctx context.Context, req *proto.SubmitPlayerCardRequest) (*emptypb.Empty, error) {
	_ = ctx
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "nil request")
	}
	if err := s.chain.AddCard(req); err != nil {
		return nil, status.Errorf(codes.Internal, "add card: %v", err)
	}
	return &emptypb.Empty{}, nil
}

func (s *GRPCServices) ClearGameInfo(ctx context.Context, req *proto.ClearGameInfoRequest) (*emptypb.Empty, error) {
	_ = ctx
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "nil request")
	}
	s.chain.ClearGameInfo(req.GetGameId())
	return &emptypb.Empty{}, nil
}

func (s *GRPCServices) CollectToken(ctx context.Context, req *proto.CollectTokenRequest) (*proto.CollectTokenResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "nil request")
	}
	if req.GetPlayerId() <= 0 {
		return nil, status.Error(codes.InvalidArgument, "player_id must be positive")
	}
	if req.GetPlayerAddress() == "" {
		return nil, status.Error(codes.InvalidArgument, "player_address is required")
	}
	if req.GetTokenAmount() == "" {
		return nil, status.Error(codes.InvalidArgument, "token_amount is required")
	}
	res, err := s.chain.CollectToken(ctx, req.GetPlayerId(), req.GetPlayerAddress(), req.GetTokenAmount())
	if err != nil {
		if errors.Is(err, worker.ErrWalletChainNotConfigured) {
			return nil, status.Error(codes.FailedPrecondition, err.Error())
		}
		return nil, status.Errorf(codes.Internal, "collect token: %v", err)
	}
	return &proto.CollectTokenResponse{
		TxHash:   res.TxHash,
		LedgerId: res.LedgerID,
	}, nil
}
