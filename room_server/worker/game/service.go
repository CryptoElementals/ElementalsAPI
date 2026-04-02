package game

import (
	"context"
	"errors"

	"github.com/CryptoElementals/common/conversion"
	"github.com/CryptoElementals/common/db"
	dao "github.com/CryptoElementals/common/models"
	"github.com/CryptoElementals/common/room_server/worker"
	"github.com/CryptoElementals/common/room_server/worker/protopub"
	"github.com/CryptoElementals/common/room_server/worker/types"
	"github.com/CryptoElementals/common/rpc/proto"
)

// Publisher publishes game events to clients (e.g. gRPC PubSub). Must be non-nil for production.
type Publisher = protopub.Publisher

type ContractClient interface {
	SubmitTasks(tasks []types.RoomContractTask) error
	NotifyTxsCompleted(txs *proto.TransactionBatch)
}

type TxPoolEnqueuer interface {
	AddCreateRoom(evt *types.RequireGameCreationEvent)
	AddSetTurnReady(evt *types.RequireSetupNewTurnEvent)
	AddCommitment(evt *proto.SubmitPlayerCommitmentRequest) error
	AddCard(evt *proto.SubmitPlayerCardRequest) error
	ClearGameInfo(gameID uint)
}

type GameResultSettler interface {
	GameResultSettlement(event *types.GameCompletedEvent) error
}

type Service struct {
	ctx              context.Context
	gameManager      *GameManager
	gameArgsTemplate *dao.GameArgs
}

func (s *Service) SubmitTransactions(txs *proto.TransactionBatch) error {
	return s.gameManager.SubmitTransactions(txs)
}

func NewService(
	ctx context.Context,
	workerManager *worker.WorkerManager,
	pub Publisher,
	argsTemplate *dao.GameArgs,
	chainSvc ContractClient,
	poolBatchSize int,
) *Service {
	mgr := NewGameManager(ctx, workerManager, pub, argsTemplate, chainSvc, poolBatchSize)
	return &Service{
		ctx:              ctx,
		gameManager:      mgr,
		gameArgsTemplate: argsTemplate,
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
	return s.gameManager.GetActiveGame(playerAddress)
}

func (s *Service) GetBattleInfo(_ context.Context, req *proto.GetBattleInfoRequest) (*proto.GetBattleInfoResponse, error) {
	roundResult, gameResult, err := s.LoadBattleInfoFromDB(req.GameID, req.RoundNumber)
	if err != nil {
		return nil, err
	}
	return &proto.GetBattleInfoResponse{
		RoundResult: roundResult,
		GameResult:  gameResult,
	}, nil
}

func (s *Service) LoadBattleInfoFromDB(gameID uint32, roundNum uint32) (*proto.RoundResult, *proto.GameResult, error) {
	gameInfo, err := db.LoadGameByGameID(uint(gameID))
	if err != nil {
		return nil, nil, err
	}
	synth := conversion.RoundByNumber(gameInfo, roundNum)
	if synth != nil {
		roundRes := conversion.DbRoundToRoundResult(synth, gameInfo)
		roundRes.RoundConfirmTimeout = uint64(gameInfo.GameArgs.ConfirmationTimeout)
		var gameRes *proto.GameResult
		if gameInfo.GameResult != nil {
			gameRes = conversion.DbGameResultToProtoGameResult(gameInfo.GameResult)
			gameRes.GameContinueTimeout = uint64(gameInfo.GameArgs.GameContinueTimeout)
		}
		return roundRes, gameRes, nil
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

func (s *Service) CreateGameAndRun(players []types.PlayerAddress, gameType uint, completedMatchID int64) (uint, error) {
	return s.gameManager.CreateGameAndRun(players, gameType, completedMatchID)
}

// GetGamePhase returns persisted in-game phase from DB/runtime (rpc/server.GamePhaseHandler). Queue / pending state is on the lobby API.
func (s *Service) GetGamePhase(req *proto.PlayerAddress) (*proto.GamePhase, error) {
	var address types.PlayerAddress
	address.FromProto(req)
	return s.gameManager.GetGamePhase(address)
}

func (s *Service) SyncGamePhase(address types.PlayerAddress) error {
	return s.gameManager.SyncGamePhase(address)
}
