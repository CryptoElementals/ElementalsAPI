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
	// SubmitTasks submits a pre-encoded batch of contract tasks to the chain.
	// Each task is an ABI-encoded payload compatible with RoomV3.batchSubmitTasks.
	SubmitTasks(tasks []types.RoomContractTask) error
}

// TxPoolEnqueuer is the interface Game uses to enqueue chain-related events (create room, set turn ready, commitments, cards).
type TxPoolEnqueuer interface {
	AddCreateRoom(evt *types.RequireGameCreationEvent)
	AddSetTurnReady(evt *types.RequireSetupNewTurnEvent)
	AddCommitment(evt *types.SubmitPlayerCommitment) error
	AddCard(evt *types.SubmitPlayerCard) error
	ClearGameInfo(gameID uint)
}

// GameHandler was previously used by long-lived per-game workers to notify GameManager on completion.
// With per-game workers removed and all events routed through the game manager worker, this indirection
// is no longer needed.

type GameResultSettler interface {
	GameResultSettlement(event *types.GameCompletedEvent) error
}

type Service struct {
	gameManager *GameManager
}

func NewService(
	ctx context.Context,
	workerManager *worker.WorkerManager,
	gameConfig *config.GameParamConfig,
	chainSvc ContractClient,
	poolBatchSize int,
	shouldRecover bool) *Service {
	gameArgs := dao.GameArgs{
		MaxRounds:         gameConfig.MaxRounds,
		InitialHP:         gameConfig.InitialHP,
		InitialMultiplier: int64(gameConfig.InitialMultiplier),

		ConfirmationTimeout:         gameConfig.ConfirmationTimeout,
		CommitmentSubmissionTimeout: gameConfig.CommitmentSubmissionTimeout,
		CardSubmissionTimeout:       gameConfig.CardSubmissionTimeout,
		GameContinueTimeout:         gameConfig.GameContinueTimeout,

		ConfirmationTimeoutRedundancy:         gameConfig.ConfirmationTimeoutRedundancy,
		CommitmentSubmissionTimeoutRedundancy: gameConfig.CommitmentSubmissionTimeoutRedundancy,
		CardSubmissionTimeoutRedundancy:       gameConfig.CardSubmissionTimeoutRedundancy,
		GameContinueTimeoutRedundancy:         gameConfig.GameContinueTimeoutRedundancy,

		PoolProcessingInterval: gameConfig.PoolProcessingInterval,
	}
	mgr := NewGameManager(ctx, workerManager, gameArgs, chainSvc, poolBatchSize)
	// Register GameManager as a worker so it can receive game-related events (e.g. from chain service).
	workerManager.SpwanWorker(ctx, types.GAME_MANAGER_ID, types.WORKER_TYPE_GAME_MANAGER, mgr)
	return &Service{
		gameManager: mgr,
	}
}

func (s *Service) SetGameResultSettler(settler GameResultSettler) {
	s.gameManager.gameResultSettler = settler
}

func (s *Service) Start() error {
	return s.gameManager.Start()
}

func (s *Service) Stop() {
	s.gameManager.Stop()
}

func (s *Service) GetActiveGameInfo(playerAddress types.PlayerAddress) *proto.GameInfo {
	gameInfo := s.gameManager.GetActiveGame(playerAddress)
	return gameInfo
}

func (s *Service) GetBattleInfo(_ context.Context, gameID uint32, roundNum uint32) (*proto.RoundResult, *proto.GameResult, error) {
	// Always load battle info from DB; do not rely on per-game workers.
	return s.LoadBattleInfoFromDB(gameID, roundNum)
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
			roundRes.RoundConfirmTimeout = uint64(gameInfo.GameArgs.ConfirmationTimeout)
			if gameInfo.GameResult != nil {
				gameRes = conversion.DbGameResultToProtoGameResult(gameInfo.GameResult)
				gameRes.GameContinueTimeout = uint64(gameInfo.GameArgs.GameContinueTimeout)
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

// SyncGamePhase sends the current game phase directly to the player worker
func (s *Service) SyncGamePhase(address types.PlayerAddress) error {
	return s.gameManager.SyncGamePhase(address)
}

// HandleSubmitPlayerCommitment handles a player commitment submission
func (s *Service) HandleSubmitPlayerCommitment(evt *types.SubmitPlayerCommitment) error {
	return s.gameManager.HandleSubmitPlayerCommitment(evt)
}

// HandleSubmitPlayerCard handles a player card submission
func (s *Service) HandleSubmitPlayerCard(evt *types.SubmitPlayerCard) error {
	return s.gameManager.HandleSubmitPlayerCard(evt)
}
