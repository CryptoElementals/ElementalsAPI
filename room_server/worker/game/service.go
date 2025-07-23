package game

import (
	"context"
	"errors"

	"github.com/CryptoElementals/common/conversion"
	"github.com/CryptoElementals/common/room_server/worker"
	"github.com/CryptoElementals/common/room_server/worker/types"
	"github.com/CryptoElementals/common/rpc/proto"
)

type Service struct {
	ctx         context.Context
	gameManager *GameManager
}

func NewService(ctx context.Context, workerManager *worker.WorkerManager) *Service {
	return &Service{
		ctx:         ctx,
		gameManager: NewGameManager(ctx, workerManager),
	}
}

func (s *Service) Start() error {
	return s.gameManager.Start()
}

func (s *Service) GetActiveGameInfo(playerAddress types.PlayerAddress) *proto.GameInfo {
	gameInfo := s.gameManager.GetActiveGame(playerAddress)
	if gameInfo == nil {
		return nil
	}
	return conversion.DbGameInfoToProtoGameInfo(gameInfo)
}

func (s *Service) GetBattleInfo(_ context.Context, gameID uint32, roundNum uint32) (*proto.RoundResult, error) {
	game := s.gameManager.GetActiveGameByID(uint(gameID))
	if game == nil {
		return nil, errors.New("game not found")
	}
	res := game.GetBattleInfo(roundNum)
	if res != nil {
		return nil, errors.New("round not found")
	}
	return res, nil
}

func (s *Service) IsPlayerInGame(playerAddress *types.PlayerAddress) bool {
	return s.gameManager.IsPlayerInGame(*playerAddress)
}

func (s *Service) GetPlayerGameInfo(playerAddress types.PlayerAddress) proto.PlayerStatus {
	gameInfo := s.gameManager.GetActiveGame(playerAddress)
	if gameInfo == nil {
		return proto.PlayerStatus_PLAYER_KNOWN
	}
	if gameInfo.Status == proto.GameStatus_GAME_INIT {
		return proto.PlayerStatus_PLAYER_MATCHED
	}
	return proto.PlayerStatus_PLAYER_IN_GAME
}
