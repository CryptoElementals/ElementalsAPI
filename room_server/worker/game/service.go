package game

import (
	"context"
	"errors"

	"github.com/CryptoElementals/common/conversion"
	"github.com/CryptoElementals/common/db"
	"github.com/CryptoElementals/common/room_server/worker"
	"github.com/CryptoElementals/common/room_server/worker/types"
	"github.com/CryptoElementals/common/rpc/proto"
)

type Service struct {
	ctx         context.Context
	gameManager *GameManager
}

func NewService(ctx context.Context, workerManager *worker.WorkerManager,
	initialHP int64, roundTimeout int64, maxRounds int64, chainSvc ContractClient) *Service {
	return &Service{
		ctx:         ctx,
		gameManager: NewGameManager(ctx, workerManager, initialHP, roundTimeout, maxRounds, chainSvc),
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

func (s *Service) GetBattleInfo(_ context.Context, gameID uint32, roundNum uint32) (*proto.RoundResult, *proto.GameResult, error) {
	game := s.gameManager.GetActiveGameByID(uint(gameID))
	if game == nil {
		// it is a cold game now
		return s.LoadBattleInfoFromDB(gameID, roundNum)
	}
	roundRes, gameRes := game.GetBattleInfo(roundNum)
	if roundRes == nil {
		return nil, nil, errors.New("round not found")
	}
	return roundRes, gameRes, nil
}

func (s *Service) LoadBattleInfoFromDB(gameID uint32, roundNum uint32) (*proto.RoundResult, *proto.GameResult, error) {
	gameInfo, err := db.LoadGameByGameID(uint(gameID))
	if err != nil {
		return nil, nil, err
	}
	for _, round := range gameInfo.Rounds {
		if round.RoundNumber == roundNum {
			return conversion.DbRoundToRoundResult(round), conversion.DbGameResultToProtoGameResult(gameInfo.GameResult), nil
		}
	}
	return nil, nil, errors.New("round not found")
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

func (s *Service) HandleGameMatchedEvent(evt *types.GameMatchedEvent) error {
	return s.gameManager.HandleGameMatchedEvent(evt)
}
