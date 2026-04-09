package game

import (
	"context"
	"errors"

	"github.com/CryptoElementals/common/conversion"
	"github.com/CryptoElementals/common/db"
	"github.com/CryptoElementals/common/pubsub"
	"github.com/CryptoElementals/common/room_server/worker"
	"github.com/CryptoElementals/common/room_server/worker/types"
	"github.com/CryptoElementals/common/rpc/proto"
	"google.golang.org/protobuf/types/known/emptypb"
)

// Publisher publishes game events (e.g. [pubsub.StreamPublisher] on Redis). Must be non-nil for production.
type Publisher = pubsub.Publisher

type ContractClient interface {
	SubmitTasks(tasks []types.RoomContractTask) error
	NotifyTxsCompleted(txs *proto.TransactionBatch)
}

type TxPoolEnqueuer interface {
	AddCreateRoom(evt *types.RequireGameCreationEvent)
	AddSetTurnReady(evt *types.RequireSetupNewTurnEvent)
	AddCommitment(evt *proto.SubmitPlayerCommitmentRequest) error
	AddCard(evt *proto.SubmitPlayerCardRequest) error
	ClearGameInfo(gameID int64)
}

type GameResultSettler interface {
	GameResultSettlement(event *types.GameCompletedEvent) error
}

type Service struct {
	ctx         context.Context
	gameManager *GameManager
}

func (s *Service) SubmitTransactions(txs *proto.TransactionBatch) error {
	s.gameManager.SubmitTransactions(txs)
	return nil
}

func NewService(
	ctx context.Context,
	workerManager *worker.WorkerManager,
	pub Publisher,
	gameArgsTemplateID uint,
	chainSvc ContractClient,
	poolBatchSize int,
	poolProcessingInterval int,
) *Service {
	mgr := NewGameManager(ctx, workerManager, pub, gameArgsTemplateID, chainSvc, poolBatchSize, poolProcessingInterval)
	return &Service{
		ctx:         ctx,
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
	return s.gameManager.GetActiveGame(playerAddress)
}

func (s *Service) LoadBattleInfoFromDB(gameID int64, roundNum uint32) (*proto.RoundResult, *proto.GameResult, error) {
	gameInfo, err := db.LoadGameByGameID(gameID)
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

func (s *Service) CreateGameAndRun(players []types.PlayerAddress, gameType uint, completedMatchID int64) (int64, error) {
	return s.gameManager.CreateGameAndRun(players, gameType, completedMatchID)
}

func (s *Service) SyncGamePhase(address types.PlayerAddress) error {
	return s.gameManager.SyncGamePhase(address)
}

func (s *Service) CreateGameAndRunRPC(ctx context.Context, req *proto.CreateGameAndRunRequest) (*proto.CreateGameAndRunResponse, error) {
	_ = ctx
	players := make([]types.PlayerAddress, 0, len(req.GetPlayers()))
	for _, p := range req.GetPlayers() {
		var a types.PlayerAddress
		a.FromProto(p)
		players = append(players, a)
	}
	gid, err := s.gameManager.CreateGameAndRun(players, uint(req.GetGameType()), req.GetCompletedMatchId())
	if err != nil {
		return nil, err
	}
	return &proto.CreateGameAndRunResponse{GameId: gid}, nil
}

func (s *Service) SyncGamePhaseRPC(ctx context.Context, req *proto.PlayerAddress) (*emptypb.Empty, error) {
	_ = ctx
	var addr types.PlayerAddress
	addr.FromProto(req)
	if err := s.gameManager.SyncGamePhase(addr); err != nil {
		return nil, err
	}
	return &emptypb.Empty{}, nil
}
