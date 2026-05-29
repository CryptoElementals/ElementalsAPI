package game

import (
	"context"
	"errors"

	"github.com/CryptoElementals/common/conversion"
	"github.com/CryptoElementals/common/db"
	"github.com/CryptoElementals/common/pubsub"
	"github.com/CryptoElementals/common/room_server/worker/types"
	"github.com/CryptoElementals/common/rpc/proto"
	"google.golang.org/protobuf/types/known/emptypb"
)

// Publisher publishes game events (e.g. [pubsub.StreamPublisher] on Redis). Must be non-nil for production.
type Publisher = pubsub.Publisher

// RoomChain is implemented by the chain worker: tx pools and per-game chain assignment.
type RoomChain interface {
	TxPoolEnqueuer
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
	pub Publisher,
	gameArgsTemplateID uint,
	roomChain RoomChain,
) *Service {
	mgr := NewGameManager(ctx, pub, gameArgsTemplateID, roomChain)
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

func (s *Service) CreateGameAndRunRPC(ctx context.Context, req *proto.CreateGameAndRunRequest) (*proto.CreateGameAndRunResponse, error) {
	_ = ctx
	players := make([]types.PlayerAddress, 0, len(req.GetPlayers()))
	for _, p := range req.GetPlayers() {
		var a types.PlayerAddress
		a.FromProto(p)
		players = append(players, a)
	}
	gid, err := s.gameManager.CreateGameAndRun(
		players,
		proto.GameType(req.GetGameType()),
		req.GetCompletedMatchId(),
		req.GetTournamentId(),
		req.GetTierNo(),
	)
	if err != nil {
		return nil, err
	}
	return &proto.CreateGameAndRunResponse{GameId: gid}, nil
}

func (s *Service) SyncGamePhaseRPC(_ context.Context, req *proto.PlayerAddress) (*emptypb.Empty, error) {
	var addr types.PlayerAddress
	addr.FromProto(req)
	if err := s.gameManager.SyncGamePhase(addr); err != nil {
		return nil, err
	}
	return &emptypb.Empty{}, nil
}

func (s *Service) GetGamePhaseRPC(ctx context.Context, req *proto.PlayerAddress) (*proto.GamePhase, error) {
	_ = ctx
	var addr types.PlayerAddress
	addr.FromProto(req)
	return s.gameManager.GetGamePhase(addr)
}

func (s *Service) AbortAllActiveGamesRPC(ctx context.Context, _ *emptypb.Empty) (*proto.AbortAllActiveGamesResponse, error) {
	_ = ctx
	return s.gameManager.AbortAllActiveGames()
}
