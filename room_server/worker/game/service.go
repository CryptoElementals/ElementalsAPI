package game

import (
	"context"

	"github.com/CryptoElementals/common/conversion"
	"github.com/CryptoElementals/common/rpc/proto"
	"github.com/CryptoElementals/common/room_server/worker"
	"github.com/CryptoElementals/common/room_server/worker/types"
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

func (s *Service) GetActiveGameInfo(playerAddress types.PlayerAddress) *proto.GameInfo {
	gameInfo := s.gameManager.GetActiveGame(playerAddress)
	if gameInfo == nil {
		return nil
	}
	return conversion.DbGameInfoToProtoGameInfo(gameInfo)
}

func (s *Service) ListGameInfo(playerAddress types.PlayerAddress) ([]*proto.GameInfo, error) {
	return nil, nil
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
