package chainclient

import (
	"context"
	"fmt"
	"time"

	"github.com/CryptoElementals/common/log"
	"github.com/CryptoElementals/common/room_server/worker/types"
	"github.com/CryptoElementals/common/rpc/proto"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// Client implements game.TxPoolEnqueuer via ChainService gRPC.
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
		log.Warnw("room: dial chain server, retrying", "addr", addr, "attempt", i+1, "err", err)
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
	if c.conn != nil {
		return c.conn.Close()
	}
	return nil
}

func (c *Client) AddCreateRoom(evt *types.RequireGameCreationEvent) {
	if c == nil || c.client == nil || evt == nil {
		return
	}
	req := &proto.RequireGameCreationEvent{
		GameId:         evt.GameID,
		InitialHp:      evt.InitialHP,
		RoundTimeout:   evt.RoundTimeout,
		MaxRoundNumber: evt.MaxRoundNumber,
		TournamentId:   evt.TournamentID,
		TierNo:         evt.TierNo,
	}
	for i := range evt.Players {
		p := evt.Players[i]
		req.Players = append(req.Players, (&p).ToProto())
	}
	if _, err := c.client.AddCreateRoom(context.Background(), req); err != nil {
		log.Errorw("chainclient AddCreateRoom", "gameID", evt.GameID, "err", err)
	}
}

func (c *Client) AddSetTurnReady(evt *types.RequireSetupNewTurnEvent) {
	if c == nil || c.client == nil || evt == nil {
		return
	}
	req := &proto.RequireSetupNewTurnEvent{
		GameId:      evt.GameID,
		RoundNumber: evt.RoundNumber,
		TurnNumber:  evt.TurnNumber,
	}
	if _, err := c.client.AddSetTurnReady(context.Background(), req); err != nil {
		log.Errorw("chainclient AddSetTurnReady", "gameID", evt.GameID, "err", err)
	}
}

func (c *Client) AddCommitment(evt *proto.SubmitPlayerCommitmentRequest) error {
	if c == nil || c.client == nil {
		return fmt.Errorf("chain client not configured")
	}
	if evt == nil {
		return fmt.Errorf("nil request")
	}
	_, err := c.client.AddCommitment(context.Background(), evt)
	return err
}

func (c *Client) AddCard(evt *proto.SubmitPlayerCardRequest) error {
	if c == nil || c.client == nil {
		return fmt.Errorf("chain client not configured")
	}
	if evt == nil {
		return fmt.Errorf("nil request")
	}
	_, err := c.client.AddCard(context.Background(), evt)
	return err
}

func (c *Client) ClearGameInfo(gameID int64) {
	if c == nil || c.client == nil || gameID == 0 {
		return
	}
	if _, err := c.client.ClearGameInfo(context.Background(), &proto.ClearGameInfoRequest{GameId: gameID}); err != nil {
		log.Errorw("chainclient ClearGameInfo", "gameID", gameID, "err", err)
	}
}
