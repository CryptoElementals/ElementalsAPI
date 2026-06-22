package client

import (
	"context"
	"fmt"

	pb "github.com/CryptoElementals/common/rpc/proto"
	"google.golang.org/protobuf/types/known/emptypb"
)

// RequestWithdraw calls ledger-server RequestWithdraw for the given server type.
func RequestWithdraw(ctx context.Context, serverType string, req *pb.RequestWithdrawRequest) (*pb.RequestWithdrawResponse, error) {
	cl := LedgerClientForType(serverType)
	if cl == nil {
		return nil, fmt.Errorf("ledger client is not initialized for server type %q", serverType)
	}
	return cl.RequestWithdraw(ctx, req)
}

// ListChainTokenLedgers calls ledger-server ListChainTokenLedgers for the given server type.
func ListChainTokenLedgers(ctx context.Context, serverType string, req *pb.ListChainTokenLedgersRequest) (*pb.ListChainTokenLedgersResponse, error) {
	cl := LedgerClientForType(serverType)
	if cl == nil {
		return nil, fmt.Errorf("ledger client is not initialized for server type %q", serverType)
	}
	return cl.ListChainTokenLedgers(ctx, req)
}

// GetTokenUnitRates calls ledger-server GetTokenUnitRates for the given server type.
func GetTokenUnitRates(ctx context.Context, serverType string) (*pb.GetTokenUnitRatesResponse, error) {
	cl := LedgerClientForType(serverType)
	if cl == nil {
		return nil, fmt.Errorf("ledger client is not initialized for server type %q", serverType)
	}
	return cl.GetTokenUnitRates(ctx, &emptypb.Empty{})
}

// GetWithdrawableTokenAmount calls ledger-server GetWithdrawableTokenAmount for the given server type.
func GetWithdrawableTokenAmount(ctx context.Context, serverType string, req *pb.GetWithdrawableTokenAmountRequest) (*pb.GetWithdrawableTokenAmountResponse, error) {
	cl := LedgerClientForType(serverType)
	if cl == nil {
		return nil, fmt.Errorf("ledger client is not initialized for server type %q", serverType)
	}
	return cl.GetWithdrawableTokenAmount(ctx, req)
}
