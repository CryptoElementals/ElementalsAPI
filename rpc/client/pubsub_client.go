package client

import (
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type Client struct {
	conn *grpc.ClientConn
	*PubSubClient
	*RpcClient
}

func NewClient(addr string) (*Client, error) {
	conn, err := grpc.NewClient(addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, err
	}
	return &Client{
		conn:         conn,
		PubSubClient: NewPubSubClient(conn),
		RpcClient:    NewRpcClient(conn),
	}, nil
}

func (c *Client) Close() error {
	c.PubSubClient.Close()
	return c.conn.Close()
}
