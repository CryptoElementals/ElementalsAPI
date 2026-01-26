package client

import (
	"context"

	"github.com/CryptoElementals/common/room_server/worker/types"

	"github.com/CryptoElementals/common/rpc/proto"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/protobuf/types/known/emptypb"
)

type RpcClient struct {
	client proto.RpcServiceClient
	conn   *grpc.ClientConn
}

func NewRpcClient(conn *grpc.ClientConn) *RpcClient {
	return &RpcClient{
		client: proto.NewRpcServiceClient(conn),
		conn:   conn,
	}
}

func NewRpcClientWithAddr(addr string) (*RpcClient, error) {
	conn, err := grpc.NewClient(addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, err
	}
	return &RpcClient{
		client: proto.NewRpcServiceClient(conn),
		conn:   conn,
	}, nil
}

func (c *RpcClient) Close() error {
	return c.conn.Close()
}

func (c *RpcClient) JoinQueue(ctx context.Context, addr *types.PlayerAddress) error {
	_, err := c.client.JoinQueue(ctx, addr.ToProto())
	return err
}

func (c *RpcClient) ExitQueue(ctx context.Context, addr *types.PlayerAddress) error {
	_, err := c.client.ExitQueue(ctx, addr.ToProto())
	return err
}

func (c *RpcClient) GetGamePhase(ctx context.Context, addr *types.PlayerAddress) (*proto.GamePhase, error) {
	return c.client.GetGamePhase(ctx, addr.ToProto())
}

func (c *RpcClient) GetBattleInfo(ctx context.Context, gameID, roundNumber uint) (*proto.GetBattleInfoResponse, error) {
	return c.client.GetBattleInfo(ctx, &proto.GetBattleInfoRequest{
		GameID:      uint32(gameID),
		RoundNumber: uint32(roundNumber),
	})
}

func (c *RpcClient) ConfirmBattle(ctx context.Context, addr *types.PlayerAddress, gameID, roundNumber, turnNumber uint) error {
	_, err := c.client.ConfirmBattle(ctx, &proto.ConfirmBattleRequest{
		PlayerAddress: addr.ToProto(),
		GameID:        uint32(gameID),
		RoundNumber:   uint32(roundNumber),
		TurnNumber:    uint32(turnNumber),
	})
	return err
}

// chain related api
func (c *RpcClient) SubmitTransactions(ctx context.Context, in *proto.TransactionBatch) error {
	_, err := c.client.SubmitTransactions(ctx, in)
	return err
}

func (c *RpcClient) ContinueGame(ctx context.Context, addr *types.PlayerAddress, gameID uint) error {
	_, err := c.client.ContinueGame(ctx, &proto.ContinueGameRequest{
		Player:     addr.ToProto(),
		LastGameID: uint32(gameID),
	})
	return err
}

func (c *RpcClient) RefuseContinueGame(ctx context.Context, addr *types.PlayerAddress, gameID uint) error {
	_, err := c.client.RefuseContinueGame(ctx, &proto.RefuseContinueGameRequest{
		Player:     addr.ToProto(),
		LastGameID: uint32(gameID),
	})
	return err
}

func (c *RpcClient) GetPlayerToken(ctx context.Context, playerId int64) (*proto.GetPlayerTokenResponse, error) {
	token, err := c.client.GetPlayerToken(ctx, &proto.GetPlayerTokenRequest{
		Id: playerId,
	})
	if err != nil {
		return nil, err
	}
	return token, nil
}

func (c *RpcClient) IsPlayerInQueue(ctx context.Context, addr types.PlayerAddress) (bool, error) {
	resp, err := c.client.IsPlayerInQueue(ctx, addr.ToProto())
	if err != nil {
		return false, err
	}
	return resp.IsInQueue, nil
}

func (c *RpcClient) GetPlayerStatus(ctx context.Context, addr *types.PlayerAddress) (*proto.GetPlayerStatusResponse, error) {
	return c.client.GetPlayerStatus(ctx, addr.ToProto())
}

func (c *RpcClient) Surrender(ctx context.Context, addr *types.PlayerAddress, gameID uint) error {
	_, err := c.client.Surrender(ctx, &proto.SurrenderRequest{
		GameID:  uint32(gameID),
		Address: addr.ToProto(),
	})
	return err
}

func (c *RpcClient) GetRoundTimeoutConfig(ctx context.Context) (*proto.TimeoutConfig, error) {
	timeoutCfg, err := c.client.GetGameTimeoutConfig(ctx, &emptypb.Empty{})
	if err != nil {
		return nil, err
	}
	return timeoutCfg, nil
}

func (c *RpcClient) SubmitPlayerCommitment(ctx context.Context, addr *types.PlayerAddress, roundNumber uint32, commitment []byte, turnNumber uint32, signature []byte, gameID uint) error {
	_, err := c.client.SubmitPlayerCommitment(ctx, &proto.SubmitPlayerCommitmentRequest{
		GameID:      uint32(gameID),
		Address:     addr.ToProto(),
		RoundNumber: roundNumber,
		Commitment:  commitment,
		TurnNumber:  uint32(turnNumber),
		Signature:   signature,
	})
	return err
}

func (c *RpcClient) SubmitPlayerCard(ctx context.Context, addr *types.PlayerAddress, roundNumber uint32, salt []byte, card uint, turnNumber uint32, signature []byte, gameID uint) error {
	_, err := c.client.SubmitPlayerCard(ctx, &proto.SubmitPlayerCardRequest{
		GameID:      uint32(gameID),
		Address:     addr.ToProto(),
		RoundNumber: roundNumber,
		Salt:        salt,
		Card:        uint32(card),
		TurnNumber:  uint32(turnNumber),
		Signature:   signature,
	})
	return err
}
