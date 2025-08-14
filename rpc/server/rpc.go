package server

import (
	"context"

	"github.com/CryptoElementals/common/room_server/worker/types"
	"github.com/CryptoElementals/common/rpc/proto"
	"google.golang.org/protobuf/types/known/emptypb"
)

type Rpc struct {
	proto.UnimplementedRpcServiceServer
	chainHandler  ChainRequestHandler
	playerHandler PlayerRequestHandler
}

func NewRpc(
	chainHandler ChainRequestHandler,
	playerHandler PlayerRequestHandler,
) *Rpc {
	return &Rpc{
		chainHandler:  chainHandler,
		playerHandler: playerHandler,
	}
}

func (s *Rpc) JoinQueue(ctx context.Context, req *proto.PlayerAddress) (*emptypb.Empty, error) {
	addr := types.PlayerAddress{}
	addr.FromProto(req)
	return &emptypb.Empty{}, s.playerHandler.JoinQueue(addr)
}

func (s *Rpc) ExitQueue(ctx context.Context, req *proto.PlayerAddress) (*emptypb.Empty, error) {
	addr := types.PlayerAddress{}
	addr.FromProto(req)
	return &emptypb.Empty{}, s.playerHandler.ExitQueue(addr)
}

func (s *Rpc) GetGamePhase(ctx context.Context, req *proto.PlayerAddress) (*proto.GamePhase, error) {
	addr := types.PlayerAddress{}
	addr.FromProto(req)
	return s.playerHandler.GetGamePhase(addr)
}

func (s *Rpc) GetBattleInfo(ctx context.Context, req *proto.GetBattleInfoRequest) (*proto.GetBattleInfoResponse, error) {
	roundResult, gameResult, err := s.playerHandler.GetBattleInfo(ctx, req.GameID, req.RoundNumber)
	if err != nil {
		return nil, err
	}
	return &proto.GetBattleInfoResponse{
		RoundResult: roundResult,
		GameResult:  gameResult,
	}, nil
}

func (s *Rpc) ConfirmBattle(ctx context.Context, req *proto.ConfirmBattleRequest) (*emptypb.Empty, error) {
	addr := types.PlayerAddress{}
	addr.FromProto(req.PlayerAddress)
	return &emptypb.Empty{}, s.playerHandler.ConfirmBattle(addr, uint(req.GameID), req.RoundNumber)
}

func (s *Rpc) ContinueGame(ctx context.Context, req *proto.ContinueGameRequest) (*emptypb.Empty, error) {
	addr := types.PlayerAddress{}
	addr.FromProto(req.Player)
	return &emptypb.Empty{}, s.playerHandler.ContinueGame(addr, uint(req.LastGameID))
}

func (s *Rpc) RefuseContinueGame(ctx context.Context, req *proto.RefuseContinueGameRequest) (*emptypb.Empty, error) {
	addr := types.PlayerAddress{}
	addr.FromProto(req.Player)
	return &emptypb.Empty{}, s.playerHandler.RefuseContinueGame(addr, uint(req.LastGameID))
}

// chain related api
func (s *Rpc) SubmitTransactions(ctx context.Context, req *proto.TransactionBatch) (*emptypb.Empty, error) {
	err := s.chainHandler.SubmitTransactions(req)
	if err != nil {
		return nil, err
	}
	return &emptypb.Empty{}, nil
}

func (s *Rpc) GetPlayerToken(ctx context.Context, req *proto.GetPlayerTokenRequest) (*proto.GetPlayerTokenResponse, error) {
	return s.playerHandler.GetPlayerToken(req.WalletAddress)
}
func (s *Rpc) IsPlayerInQueue(ctx context.Context, req *proto.PlayerAddress) (*proto.IsPlayerInQueueResponse, error) {
	addr := types.PlayerAddress{}
	addr.FromProto(req)
	isInQueue := s.playerHandler.IsPlayerInQueue(addr)
	return &proto.IsPlayerInQueueResponse{
		IsInQueue: isInQueue,
	}, nil
}

func (s *Rpc) Surrender(ctx context.Context, req *proto.SurrenderRequest) (*emptypb.Empty, error) {
	addr := types.PlayerAddress{}
	addr.FromProto(req.Address)
	err := s.playerHandler.Surrender(addr, uint(req.GameID))
	if err != nil {
		return nil, err
	}
	return &emptypb.Empty{}, nil
}

func (s *Rpc) GetGameTimeoutConfig(context.Context, *emptypb.Empty) (*proto.TimeoutConfig, error) {
	return s.playerHandler.GetTimeoutConfig()
}

type ChainRequestHandler interface {
	SubmitTransactions(req *proto.TransactionBatch) error
}

type PlayerRequestHandler interface {
	JoinQueue(playerAddress types.PlayerAddress) error
	ExitQueue(playerAddress types.PlayerAddress) error
	RefuseContinueGame(playerAddress types.PlayerAddress, gameID uint) error
	ContinueGame(playerAddress types.PlayerAddress, gameID uint) error
	ConfirmBattle(playerAddress types.PlayerAddress, gameID uint, roundNum uint32) error
	IsPlayerInQueue(address types.PlayerAddress) bool
	Surrender(address types.PlayerAddress, gameID uint) error

	GetGamePhase(playerAddress types.PlayerAddress) (*proto.GamePhase, error)
	GetBattleInfo(ctx context.Context, gameID uint32, roundNum uint32) (*proto.RoundResult, *proto.GameResult, error)
	GetPlayerToken(walletAddress string) (*proto.GetPlayerTokenResponse, error)
	GetTimeoutConfig() (*proto.TimeoutConfig, error)
}
