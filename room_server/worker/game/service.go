package game

import (
	"context"
	"errors"
	"sync"

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
	// SubmitTasks submits a pre-encoded batch of contract tasks to the chain.
	// Each task is an ABI-encoded payload compatible with RoomV3.batchSubmitTasks.
	SubmitTasks(tasks []types.RoomContractTask) error

	NotifyTxsCompleted(txs *proto.TransactionBatch)
}

// TxPoolEnqueuer is the interface Game uses to enqueue chain-related events (create room, set turn ready, commitments, cards).
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

// PlayerQueue is the subset of queue operations needed for player-facing RPC and PubSub lifecycle.
type PlayerQueue interface {
	HandleJoinQueueEvent(player *proto.PlayerAddress) error
	HandleExitQueueEvent(player *proto.PlayerAddress) error
	HandleConfirmMatch(req *proto.ConfirmMatchRequest) error
	HandleCancelMatch(req *proto.CancelMatchRequest) error
	IsPlayerInQueue(playerAddress types.PlayerAddress) bool
	IsPlayerPendingMatch(playerAddress types.PlayerAddress) bool
	RegisterBots(addrs ...*types.PlayerAddress) error
	UnregisterBots(addrs ...*types.PlayerAddress) error
}

type Service struct {
	ctx              context.Context
	gameManager      *GameManager
	gameArgsTemplate *dao.GameArgs
	playerMu         sync.Mutex
	connected        map[types.PlayerAddress]struct{}
	queue            PlayerQueue
}

// SubmitTransactions implements rpc/server.ChainRequestHandler by delegating
// transaction batch handling to the stateless GameManager.
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
		connected:        make(map[types.PlayerAddress]struct{}),
	}
}

// SetPlayerQueue wires the matchmaking / continue queue (must be called before serving player RPCs).
func (s *Service) SetPlayerQueue(q PlayerQueue) {
	s.playerMu.Lock()
	defer s.playerMu.Unlock()
	s.queue = q
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

func (s *Service) HandleGameMatchedEvent(evt *types.GameMatchedEvent) (uint, error) {
	return s.gameManager.HandleGameMatchedEvent(evt)
}

// CreatePvpGameAfterQueueConfirm is used by the matchmaking queue after both players confirm game_match.
func (s *Service) CreatePvpGameAfterQueueConfirm(players []types.PlayerAddress, gameType uint, completedMatchID int64) (uint, error) {
	return s.gameManager.CreatePvpGameAfterQueueConfirm(players, gameType, completedMatchID)
}

// GetGamePhase returns phase for RPC clients (queue, continue, and in-game), mirroring former player.Service.
func (s *Service) GetGamePhase(req *proto.PlayerAddress) (*proto.GamePhase, error) {
	var address types.PlayerAddress
	address.FromProto(req)
	s.playerMu.Lock()
	q := s.queue
	s.playerMu.Unlock()
	if q == nil {
		return nil, errors.New("queue not configured")
	}
	status := s.getPlayerStatusLocked(q, address)
	switch status {
	case proto.PlayerStatus_PLAYER_MATCHED:
		if q.IsPlayerPendingMatch(address) {
			return &proto.GamePhase{
				GameType: proto.GameType_PVP,
			}, nil
		}
		fallthrough
	case proto.PlayerStatus_PLAYER_IN_GAME:
		return s.gameManager.GetGamePhase(address)
	case proto.PlayerStatus_PLAYER_IN_QUEUE:
		return &proto.GamePhase{
			GameType: proto.GameType_PVP,
		}, nil
	case proto.PlayerStatus_PLAYER_UNKNOWN:
		return &proto.GamePhase{
			GameType: proto.GameType_PVP,
		}, nil
	default:
		return nil, errors.New("unknown player status")
	}
}

func (s *Service) getPlayerStatusLocked(q PlayerQueue, address types.PlayerAddress) proto.PlayerStatus {
	if q.IsPlayerInQueue(address) {
		return proto.PlayerStatus_PLAYER_IN_QUEUE
	}
	if q.IsPlayerPendingMatch(address) {
		return proto.PlayerStatus_PLAYER_MATCHED
	}
	return s.GetPlayerGameInfo(address)
}

// SyncGamePhase publishes the current game phase to the player's PubSub topic.
func (s *Service) SyncGamePhase(address types.PlayerAddress) error {
	return s.gameManager.SyncGamePhase(address)
}
