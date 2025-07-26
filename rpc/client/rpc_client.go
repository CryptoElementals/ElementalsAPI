package client

import (
	context "context"
	"fmt"
	"strings"
	"sync"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	pb "github.com/CryptoElementals/common/rpc/proto"
)

// RpcClient wraps the gRPC client for RpcService.
// Used for game and chain related RPC operations.
type RpcClient struct {
	client pb.RpcServiceClient
	conn   *grpc.ClientConn
	mu     sync.RWMutex
}

// NewRpcClient creates a new RpcClient.
func NewRpcClient(serverAddr string) (*RpcClient, error) {
	serverAddr = strings.TrimPrefix(serverAddr, "http://")
	serverAddr = strings.TrimPrefix(serverAddr, "https://")
	conn, err := grpc.NewClient(serverAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, fmt.Errorf("failed to connect: %v", err)
	}
	client := pb.NewRpcServiceClient(conn)
	return &RpcClient{
		client: client,
		conn:   conn,
	}, nil
}

// Close closes the gRPC connection.
func (c *RpcClient) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.conn.Close()
}

// JoinQueue calls the JoinQueue RPC method.
func (c *RpcClient) JoinQueue(ctx context.Context, in *pb.PlayerAddress) error {
	_, err := c.client.JoinQueue(ctx, in)
	return err
}

// ExitQueue calls the ExitQueue RPC method.
func (c *RpcClient) ExitQueue(ctx context.Context, in *pb.PlayerAddress) error {
	_, err := c.client.ExitQueue(ctx, in)
	return err
}

// GetGamePhase calls the GetGamePhase RPC method.
func (c *RpcClient) GetGamePhase(ctx context.Context, in *pb.PlayerAddress) (*pb.GamePhase, error) {
	return c.client.GetGamePhase(ctx, in)
}

// GetBattleInfo calls the GetBattleInfo RPC method.
func (c *RpcClient) GetBattleInfo(ctx context.Context, in *pb.GetBattleInfoRequest) (*pb.GetBattleInfoResponse, error) {
	return c.client.GetBattleInfo(ctx, in)
}

// ConfirmBattle calls the ConfirmBattle RPC method.
func (c *RpcClient) ConfirmBattle(ctx context.Context, in *pb.ConfirmBattleRequest) error {
	_, err := c.client.ConfirmBattle(ctx, in)
	return err
}

// SubmitTransactions calls the SubmitTransactions RPC method.
func (c *RpcClient) SubmitTransactions(ctx context.Context, in *pb.TransactionBatch) error {
	_, err := c.client.SubmitTransactions(ctx, in)
	return err
}
