package chainclient

import (
	"context"
	"encoding/hex"
	"fmt"
	"math/big"
	"strings"
	"time"

	"github.com/CryptoElementals/common/log"
	"github.com/CryptoElementals/common/rpc/proto"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// Client calls ChainService for wallet operations.
type Client struct {
	conn   *grpc.ClientConn
	client proto.ChainServiceClient
}

// Dial connects to the chain server at addr.
func Dial(ctx context.Context, addr string) (*Client, error) {
	const maxAttempts = 60
	var lastErr error
	opts := []grpc.DialOption{
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithDefaultCallOptions(
			grpc.MaxCallRecvMsgSize(4*1024*1024),
			grpc.MaxCallSendMsgSize(4*1024*1024),
		),
	}
	for i := 0; i < maxAttempts; i++ {
		conn, err := grpc.NewClient(addr, opts...)
		if err == nil {
			return &Client{conn: conn, client: proto.NewChainServiceClient(conn)}, nil
		}
		lastErr = err
		log.Warnw("ledger: dial chain server, retrying", "addr", addr, "attempt", i+1, "err", err)
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(time.Second):
		}
	}
	return nil, fmt.Errorf("dial chain server at %s after %d attempts: %w", addr, maxAttempts, lastErr)
}

// Close closes the underlying gRPC connection.
func (c *Client) Close() error {
	if c != nil && c.conn != nil {
		return c.conn.Close()
	}
	return nil
}

// WithdrawResult is the outcome of a withdraw submission.
type WithdrawResult struct {
	TxHash           string
	CollectorAddress string
	LedgerID         uint64
}

// Withdraw submits a withdraw to chain-server.
func (c *Client) Withdraw(ctx context.Context, playerID int64, amountWei string, signature []byte) (*WithdrawResult, error) {
	amountWei = strings.TrimSpace(amountWei)
	if amountWei == "" {
		return nil, fmt.Errorf("amount_wei is required")
	}
	wei, ok := new(big.Int).SetString(amountWei, 10)
	if !ok || wei.Sign() <= 0 {
		return nil, fmt.Errorf("invalid amount_wei: %q", amountWei)
	}
	if len(signature) == 0 {
		return nil, fmt.Errorf("signature is required")
	}
	if c == nil || c.client == nil {
		return nil, fmt.Errorf("chain client is nil")
	}

	resp, err := c.client.Withdraw(ctx, &proto.WithdrawRequest{
		PlayerId:   playerID,
		AmountWei:  wei.String(),
		Signature:  signature,
	})
	if err != nil {
		return nil, err
	}
	return &WithdrawResult{
		TxHash:           strings.ToLower(strings.TrimSpace(resp.GetTxHash())),
		CollectorAddress: strings.ToLower(strings.TrimSpace(resp.GetCollectorAddress())),
		LedgerID:         resp.GetLedgerId(),
	}, nil
}

// DecodeWithdrawSignature parses a hex-encoded ECDSA signature.
func DecodeWithdrawSignature(hexSig string) ([]byte, error) {
	raw := strings.TrimSpace(hexSig)
	raw = strings.TrimPrefix(raw, "0x")
	if raw == "" {
		return nil, fmt.Errorf("signature is empty")
	}
	return hex.DecodeString(raw)
}
