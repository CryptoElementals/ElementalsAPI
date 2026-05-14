package client

import (
	"fmt"

	"google.golang.org/grpc"

	pb "github.com/CryptoElementals/common/rpc/proto"
	"github.com/CryptoElementals/common/stream"
)

// Client combines room/lobby gRPC with Redis stream event subscription.
type Client struct {
	roomConn  *grpc.ClientConn
	lobbyConn *grpc.ClientConn
	*PubSubClient
	*RpcClient
}

// NewClient connects room and lobby over gRPC and uses Redis streams for PubSub-shaped APIs. Requires redis.Init.
func NewClient(roomAddr, lobbyAddr string) (*Client, error) {
	opts := defaultGRPCDialOptions()
	roomConn, err := grpc.NewClient(roomAddr, opts...)
	if err != nil {
		return nil, err
	}
	lobbyConn, err := grpc.NewClient(lobbyAddr, opts...)
	if err != nil {
		_ = roomConn.Close()
		return nil, err
	}
	st, err := stream.NewRedisStream()
	if err != nil {
		_ = roomConn.Close()
		_ = lobbyConn.Close()
		return nil, fmt.Errorf("redis stream: %w", err)
	}
	lobbySvc := pb.NewLobbyServiceClient(lobbyConn)
	return &Client{
		roomConn:     roomConn,
		lobbyConn:    lobbyConn,
		PubSubClient: NewPubSubClient(st),
		RpcClient:    NewRpcClient(roomConn, lobbyConn, lobbySvc),
	}, nil
}

// Close 关闭客户端连接
func (c *Client) Close() error {
	if c.PubSubClient != nil {
		_ = c.PubSubClient.Close()
	}
	if c.lobbyConn != nil {
		_ = c.lobbyConn.Close()
	}
	if c.roomConn != nil {
		return c.roomConn.Close()
	}
	return nil
}
