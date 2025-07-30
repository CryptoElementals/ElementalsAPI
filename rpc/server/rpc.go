package server

import (
	"context"

	"github.com/CryptoElementals/common/room_server/worker/types"
	pb "github.com/CryptoElementals/common/rpc/proto"
	"google.golang.org/protobuf/types/known/emptypb"
)

type Rpc struct {
	pb.UnimplementedRpcServiceServer
	gameHandler        GameRequestHandler
	chainHandler       ChainRequestHandler
	playerHandler      PlayerRequestHandler
	playerTokenHandler PlayerTokenHandler
}

func NewRpc(gameHandler GameRequestHandler, chainHandler ChainRequestHandler, playerHandler PlayerRequestHandler, playerTokenHandler PlayerTokenHandler) *Rpc {
	return &Rpc{
		gameHandler:        gameHandler,
		chainHandler:       chainHandler,
		playerHandler:      playerHandler,
		playerTokenHandler: playerTokenHandler,
	}
}

func (s *Rpc) JoinQueue(ctx context.Context, req *pb.PlayerAddress) (*emptypb.Empty, error) {
	addr := types.PlayerAddress{}
	addr.FromProto(req)
	return &emptypb.Empty{}, s.playerHandler.JoinQueue(addr)
}

func (s *Rpc) ExitQueue(ctx context.Context, req *pb.PlayerAddress) (*emptypb.Empty, error) {
	addr := types.PlayerAddress{}
	addr.FromProto(req)
	return &emptypb.Empty{}, s.playerHandler.ExitQueue(addr)
}

func (s *Rpc) GetGamePhase(ctx context.Context, req *pb.PlayerAddress) (*pb.GamePhase, error) {
	addr := types.PlayerAddress{}
	addr.FromProto(req)
	return s.gameHandler.GetGamePhase(addr)
}

func (s *Rpc) GetBattleInfo(ctx context.Context, req *pb.GetBattleInfoRequest) (*pb.GetBattleInfoResponse, error) {
	roundResult, gameResult, err := s.gameHandler.GetBattleInfo(ctx, req.GameID, req.RoundNumber)
	if err != nil {
		return nil, err
	}
	return &pb.GetBattleInfoResponse{
		RoundResult: roundResult,
		GameResult:  gameResult,
	}, nil
}

func (s *Rpc) ConfirmBattle(ctx context.Context, req *pb.ConfirmBattleRequest) (*emptypb.Empty, error) {
	addr := types.PlayerAddress{}
	addr.FromProto(req.PlayerAddress)
	return &emptypb.Empty{}, s.playerHandler.ConfirmBattle(addr, uint(req.GameID), req.RoundNumber)
}

func (s *Rpc) ContinueGame(ctx context.Context, req *pb.ContinueGameRequest) (*emptypb.Empty, error) {
	addr := types.PlayerAddress{}
	addr.FromProto(req.Player)
	return &emptypb.Empty{}, s.playerHandler.ContinueGame(addr, uint(req.LastGameID))
}

func (s *Rpc) RefuseContinueGame(ctx context.Context, req *pb.RefuseContinueGameRequest) (*emptypb.Empty, error) {
	addr := types.PlayerAddress{}
	addr.FromProto(req.Player)
	return &emptypb.Empty{}, s.playerHandler.RefuseContinueGame(addr, uint(req.LastGameID))
}

// chain related api
func (s *Rpc) SubmitTransactions(ctx context.Context, req *pb.TransactionBatch) (*emptypb.Empty, error) {
	err := s.chainHandler.SubmitTransactions(req)
	if err != nil {
		return nil, err
	}
	return &emptypb.Empty{}, nil
}

func (s *Rpc) GetPlayerToken(ctx context.Context, req *pb.GetPlayerTokenRequest) (*pb.GetPlayerTokenResponse, error) {
	return s.playerTokenHandler.GetPlayerToken(req.WalletAddress)
}

type GameRequestHandler interface {
	GetBattleInfo(ctx context.Context, gameid uint32, roundNum uint32) (*pb.RoundResult, *pb.GameResult, error)
	GetGamePhase(playerAddress types.PlayerAddress) (*pb.GamePhase, error)
}

type PlayerTokenHandler interface {
	GetPlayerToken(walletAddress string) (*pb.GetPlayerTokenResponse, error)
}

type ChainRequestHandler interface {
	SubmitTransactions(req *pb.TransactionBatch) error
}

type PlayerRequestHandler interface {
	JoinQueue(playerAddress types.PlayerAddress) error
	ExitQueue(playerAddress types.PlayerAddress) error
	RefuseContinueGame(playerAddress types.PlayerAddress, gameID uint) error
	ContinueGame(playerAddress types.PlayerAddress, gameID uint) error
	ConfirmBattle(playerAddress types.PlayerAddress, gameID uint, roundNum uint32) error
}
