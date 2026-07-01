package ledgerserver

import (
	"context"
	"errors"
	"strings"

	"github.com/CryptoElementals/common/db"
	"github.com/CryptoElementals/common/rpc/proto"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
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

func (s *GRPCServices) RequestWithdraw(ctx context.Context, req *proto.RequestWithdrawRequest) (*proto.RequestWithdrawResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "nil request")
	}
	resp, err := s.svc.RequestWithdraw(ctx, req)
	if err != nil {
		if errors.Is(err, db.ErrInsufficientAvailableBalance) || errors.Is(err, db.ErrAuditingWithdrawInProgress) {
			return nil, status.Error(codes.InvalidArgument, err.Error())
		}
		if errors.Is(err, context.DeadlineExceeded) || errors.Is(err, context.Canceled) {
			return nil, status.Errorf(codes.DeadlineExceeded, "%v", err)
		}
		msg := err.Error()
		if strings.Contains(msg, "not configured") || strings.Contains(msg, "required") || strings.Contains(msg, "invalid") {
			return nil, status.Error(codes.InvalidArgument, msg)
		}
		return nil, status.Errorf(codes.Internal, "%v", err)
	}
	return resp, nil
}

func (s *GRPCServices) ListChainTokenLedgers(ctx context.Context, req *proto.ListChainTokenLedgersRequest) (*proto.ListChainTokenLedgersResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "nil request")
	}
	resp, err := s.svc.ListChainTokenLedgers(ctx, req)
	if err != nil {
		if strings.Contains(err.Error(), "required") {
			return nil, status.Error(codes.InvalidArgument, err.Error())
		}
		return nil, status.Errorf(codes.Internal, "%v", err)
	}
	return resp, nil
}

func (s *GRPCServices) AuditWithdraw(ctx context.Context, req *proto.AuditWithdrawRequest) (*proto.AuditWithdrawResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "nil request")
	}
	resp, err := s.svc.AuditWithdraw(ctx, req)
	if err != nil {
		msg := err.Error()
		if strings.Contains(msg, "required") || strings.Contains(msg, "invalid") || strings.Contains(msg, "exceeds") {
			return nil, status.Error(codes.InvalidArgument, msg)
		}
		if strings.Contains(msg, "not found") {
			return nil, status.Error(codes.FailedPrecondition, msg)
		}
		if strings.Contains(msg, "not configured") {
			return nil, status.Error(codes.InvalidArgument, msg)
		}
		if strings.Contains(msg, "chain withdraw") {
			return nil, status.Errorf(codes.Internal, "%v", err)
		}
		return nil, status.Errorf(codes.Internal, "%v", err)
	}
	return resp, nil
}

func (s *GRPCServices) GetWithdrawableTokenAmount(ctx context.Context, req *proto.GetWithdrawableTokenAmountRequest) (*proto.GetWithdrawableTokenAmountResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "nil request")
	}
	resp, err := s.svc.GetWithdrawableTokenAmount(ctx, req)
	if err != nil {
		if strings.Contains(err.Error(), "required") {
			return nil, status.Error(codes.InvalidArgument, err.Error())
		}
		return nil, status.Errorf(codes.Internal, "%v", err)
	}
	return resp, nil
}

func (s *GRPCServices) GetTokenUnitRates(ctx context.Context, _ *emptypb.Empty) (*proto.GetTokenUnitRatesResponse, error) {
	_ = ctx
	return s.svc.GetTokenUnitRates(), nil
}

func (s *GRPCServices) ConvertTokenAmount(ctx context.Context, req *proto.ConvertTokenAmountRequest) (*proto.ConvertTokenAmountResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "nil request")
	}
	resp, err := s.svc.ConvertTokenAmount(req)
	if err != nil {
		if strings.Contains(err.Error(), "required") || strings.Contains(err.Error(), "invalid") || strings.Contains(err.Error(), "unsupported") || strings.Contains(err.Error(), "empty") || strings.Contains(err.Error(), "positive") || strings.Contains(err.Error(), "small") {
			return nil, status.Error(codes.InvalidArgument, err.Error())
		}
		return nil, status.Errorf(codes.Internal, "%v", err)
	}
	return resp, nil
}
