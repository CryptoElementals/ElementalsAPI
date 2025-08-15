package game

import (
	"context"
	"errors"

	"github.com/CryptoElementals/common/config"
	"github.com/CryptoElementals/common/conversion"
	"github.com/CryptoElementals/common/db"
	dao "github.com/CryptoElementals/common/models"
	"github.com/CryptoElementals/common/room_server/worker"
	"github.com/CryptoElementals/common/room_server/worker/types"
	"github.com/CryptoElementals/common/rpc/proto"
)

type ContractClient interface {
	CreateRoomContract(evt *types.RequireContractCreationEvent) error
	SetRoundReady(evt *types.RequireSetupNewRoundEvent) error
}

type GameHandler interface {
	HandleGameContinueEvent(evt *types.GameContinueEvent) error
	HandleGameCompletedEvent(evt *types.GameCompletedEvent) error
}

type GameResultSettler interface {
	GameResultSettlement(event *types.GameCompletedEvent) error
}

type Service struct {
	ctx         context.Context
	gameManager *GameManager
}

func NewService(
	ctx context.Context,
	workerManager *worker.WorkerManager,
	gameConfig *config.GameParamConfig,
	chainSvc ContractClient,
	shouldRecover bool) *Service {
	gameArgs := dao.GameArgs{
		MaxRounds: gameConfig.MaxRounds,
		InitialHP: gameConfig.InitialHP,

		GameMatchTimeout:    gameConfig.GameMatchTimeout,
		RoundConfirmTimeout: gameConfig.RoundConfirmTimeout,
		RoundTimeout:        gameConfig.RoundTimeout,
		ContinueTimeout:     gameConfig.ContinueTimeout,

		GameMatchTimeoutRedundancy:    gameConfig.GameMatchTimeoutRedundancy,
		RoundConfirmTimeoutRedundancy: gameConfig.RoundConfirmTimeoutRedundancy,
		RoundTimeoutRedundancy:        gameConfig.RoundTimeoutRedundancy,
		ContinueTimeoutRedundancy:     gameConfig.ContinueTimeoutRedundancy,
	}
	return &Service{
		ctx:         ctx,
		gameManager: NewGameManager(ctx, workerManager, gameArgs, chainSvc, shouldRecover),
	}
}

func (s *Service) SetGameResultSettler(settler GameResultSettler) {
	s.gameManager.gameResultSettler = settler
}

func (s *Service) Start() error {
	return s.gameManager.Start()
}

func (s *Service) GetActiveGameInfo(playerAddress types.PlayerAddress) *proto.GameInfo {
	gameInfo := s.gameManager.GetActiveGame(playerAddress)
	return gameInfo
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
			roundRes := conversion.DbRoundToRoundResult(round)
			var gameRes *proto.GameResult
			roundRes.RoundConfirmTimeout = uint64(gameInfo.GameArgs.RoundConfirmTimeout)
			if gameInfo.GameResult != nil {
				gameRes = conversion.DbGameResultToProtoGameResult(gameInfo.GameResult)
				gameRes.GameContinueTimeout = uint64(gameInfo.GameArgs.ContinueTimeout)
			}
			return roundRes, gameRes, nil
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
		return proto.PlayerStatus_PLAYER_UNKNOWN
	}
	if gameInfo.Status == proto.GameStatus_GAME_INIT {
		return proto.PlayerStatus_PLAYER_MATCHED
	}
	return proto.PlayerStatus_PLAYER_IN_GAME
}

func (s *Service) HandleGameMatchedEvent(evt *types.GameMatchedEvent) (uint, error) {
	return s.gameManager.HandleGameMatchedEvent(evt)
}

func (s *Service) HandleGameContinueEvent(evt *types.GameContinueEvent) error {
	return s.gameManager.HandleGameContinueEvent(evt)
}

func (s *Service) GetGamePhase(address types.PlayerAddress) (*proto.GamePhase, error) {
	return s.gameManager.GetGamePhase(address)
}
