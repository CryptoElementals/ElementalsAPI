package client

import (
	"context"

	"github.com/CryptoElementals/common/room_server/worker/types"

	pb "github.com/CryptoElementals/common/rpc/proto"
	"google.golang.org/grpc"
)

type RpcClient struct {
	client pb.RpcServiceClient
	conn   *grpc.ClientConn
}

func NewRpcClient(conn *grpc.ClientConn) *RpcClient {
	return &RpcClient{
		client: pb.NewRpcServiceClient(conn),
		conn:   conn,
	}
}

func (c *RpcClient) JoinQueue(ctx context.Context, addr *types.PlayerAddress) error {
	_, err := c.client.JoinQueue(ctx, addr.ToProto())
	return err
}

func (c *RpcClient) ExitQueue(ctx context.Context, addr *types.PlayerAddress) error {
	_, err := c.client.ExitQueue(ctx, addr.ToProto())
	return err
}

func (c *RpcClient) GetGamePhase(ctx context.Context, addr *types.PlayerAddress) (*pb.GamePhase, error) {
	return c.client.GetGamePhase(ctx, addr.ToProto())
}

func (c *RpcClient) GetBattleInfo(ctx context.Context, gameID, roundNumber uint) (*pb.GetBattleInfoResponse, error) {
	return c.client.GetBattleInfo(ctx, &pb.GetBattleInfoRequest{
		GameID:      uint32(gameID),
		RoundNumber: uint32(roundNumber),
	})
}

func (c *RpcClient) ConfirmBattle(ctx context.Context, addr *types.PlayerAddress, gameID, roundNumber uint) error {
	_, err := c.client.ConfirmBattle(ctx, &pb.ConfirmBattleRequest{
		PlayerAddress: addr.ToProto(),
		GameID:        uint32(gameID),
		RoundNumber:   uint32(roundNumber),
	})
	return err
}

// chain related api
func (c *RpcClient) SubmitTransactions(ctx context.Context, in *pb.TransactionBatch) error {
	_, err := c.client.SubmitTransactions(ctx, in)
	return err
}
