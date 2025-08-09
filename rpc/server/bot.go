package server

import (
	"context"

	"github.com/CryptoElementals/common/rpc/proto"
)

type BotHandler interface {
	AddBot() *proto.PlayerAddress
}

type BotRpcServer struct {
	proto.UnimplementedBotServiceServer
	BotHandler
}

func NewBotRpcServer(h BotHandler) *BotRpcServer {
	return &BotRpcServer{
		BotHandler: h,
	}
}

func (s *BotRpcServer) AnddNewBot(ctx context.Context, req *proto.AddNewBotRequest) (*proto.AddNewBotResponse, error) {
	addr := s.BotHandler.AddBot()
	return &proto.AddNewBotResponse{
		PlayerAddress: addr,
	}, nil
}
