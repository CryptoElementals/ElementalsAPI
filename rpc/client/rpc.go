package client

import (
	"context"

	"github.com/CryptoElementals/common/room_server/worker/types"
	"github.com/CryptoElementals/common/rpc/proto"
	"google.golang.org/grpc"
)

type RpcClient struct {
	room      proto.RoomServiceClient
	lobby     proto.LobbyServiceClient
	conn      *grpc.ClientConn
	lobbyConn *grpc.ClientConn
}

func NewRpcClient(roomConn, lobbyConn *grpc.ClientConn, lobby proto.LobbyServiceClient) *RpcClient {
	return &RpcClient{
		room:      proto.NewRoomServiceClient(roomConn),
		lobby:     lobby,
		conn:      roomConn,
		lobbyConn: lobbyConn,
	}
}

// NewRpcClientRoomOnly connects only to the room server (e.g. scanner batch submit). Lobby RPCs return errLobbyNotConfigured.
func NewRpcClientRoomOnly(roomAddr string) (*RpcClient, error) {
	conn, err := grpc.NewClient(roomAddr, defaultGRPCDialOptions()...)
	if err != nil {
		return nil, err
	}
	return &RpcClient{
		room:      proto.NewRoomServiceClient(conn),
		lobby:     nil,
		conn:      conn,
		lobbyConn: nil,
	}, nil
}

func (c *RpcClient) Close() error {
	if c.lobbyConn != nil {
		_ = c.lobbyConn.Close()
	}
	if c.conn != nil {
		return c.conn.Close()
	}
	return nil
}

func (c *RpcClient) lobbyRequired() error {
	if c.lobby == nil {
		return errLobbyNotConfigured
	}
	return nil
}

var errLobbyNotConfigured = &lobbyConfigError{}

type lobbyConfigError struct{}

func (e *lobbyConfigError) Error() string {
	return "lobby gRPC client not configured (pass lobby address to rpc/client.NewClient)"
}

func (c *RpcClient) JoinQueue(ctx context.Context, addr *types.PlayerAddress) error {
	if err := c.lobbyRequired(); err != nil {
		return err
	}
	_, err := c.lobby.JoinQueue(ctx, addr.ToProto())
	return err
}

func (c *RpcClient) ExitQueue(ctx context.Context, addr *types.PlayerAddress) error {
	if err := c.lobbyRequired(); err != nil {
		return err
	}
	_, err := c.lobby.ExitQueue(ctx, addr.ToProto())
	return err
}

func (c *RpcClient) ConfirmBattle(ctx context.Context, addr *types.PlayerAddress, gameID, roundNumber, turnNumber uint) error {
	_, err := c.room.ConfirmBattle(ctx, &proto.ConfirmBattleRequest{
		PlayerAddress: addr.ToProto(),
		GameID:        uint32(gameID),
		RoundNumber:   uint32(roundNumber),
		TurnNumber:    uint32(turnNumber),
	})
	return err
}

func (c *RpcClient) ConfirmMatch(ctx context.Context, addr *types.PlayerAddress, matchID int64) error {
	if err := c.lobbyRequired(); err != nil {
		return err
	}
	_, err := c.lobby.ConfirmMatch(ctx, &proto.ConfirmMatchRequest{
		PlayerAddress: addr.ToProto(),
		MatchId:       matchID,
	})
	return err
}

func (c *RpcClient) CancelMatch(ctx context.Context, addr *types.PlayerAddress, matchID int64) error {
	if err := c.lobbyRequired(); err != nil {
		return err
	}
	_, err := c.lobby.CancelMatch(ctx, &proto.CancelMatchRequest{
		PlayerAddress: addr.ToProto(),
		MatchId:       matchID,
	})
	return err
}

func (c *RpcClient) SubmitTransactions(ctx context.Context, in *proto.TransactionBatch) error {
	_, err := c.room.SubmitTransactions(ctx, in)
	return err
}

func (c *RpcClient) GetPlayerToken(ctx context.Context, playerId int64) (*proto.GetPlayerTokenResponse, error) {
	if err := c.lobbyRequired(); err != nil {
		return nil, err
	}
	return c.lobby.GetPlayerToken(ctx, &proto.GetPlayerTokenRequest{Id: playerId})
}

func (c *RpcClient) GetPlayerStatus(ctx context.Context, addr *types.PlayerAddress) (*proto.GetPlayerStatusResponse, error) {
	if err := c.lobbyRequired(); err != nil {
		return nil, err
	}
	return c.lobby.GetPlayerStatus(ctx, addr.ToProto())
}

func (c *RpcClient) Surrender(ctx context.Context, addr *types.PlayerAddress, gameID uint) error {
	_, err := c.room.Surrender(ctx, &proto.SurrenderRequest{
		GameID:  uint32(gameID),
		Address: addr.ToProto(),
	})
	return err
}

func (c *RpcClient) SubmitPlayerCommitment(ctx context.Context, addr *types.PlayerAddress, roundNumber uint32, commitment []byte, turnNumber uint32, signature []byte, gameID uint) error {
	_, err := c.room.SubmitPlayerCommitment(ctx, &proto.SubmitPlayerCommitmentRequest{
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
	_, err := c.room.SubmitPlayerCard(ctx, &proto.SubmitPlayerCardRequest{
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
