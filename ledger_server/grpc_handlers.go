package ledgerserver

import (
	"context"

	"github.com/CryptoElementals/common/rpc/proto"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// GRPCServices implements LedgerService.
type GRPCServices struct {
	proto.UnimplementedLedgerServiceServer

	svc *Service
}

func NewGRPCServices(svc *Service) *GRPCServices {
	return &GRPCServices{svc: svc}
}

func (s *GRPCServices) SubmitChainEvents(ctx context.Context, req *proto.SubmitChainEventsRequest) (*proto.SubmitChainEventsResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "nil request")
	}
	resp, err := s.svc.SubmitChainEvents(ctx, req)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "%v", err)
	}
	return resp, nil
}
