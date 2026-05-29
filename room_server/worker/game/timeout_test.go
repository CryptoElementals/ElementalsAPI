package game

import (
	"context"
	"testing"

	"github.com/CryptoElementals/common/pubsub"
	"github.com/CryptoElementals/common/rpc/proto"
	dao "github.com/CryptoElementals/common/models"
	"github.com/CryptoElementals/common/room_server/worker/types"
	"github.com/stretchr/testify/require"
)

type noopTxPool struct{}

func (noopTxPool) AddCreateRoom(_ *types.RequireGameCreationEvent)           {}
func (noopTxPool) AddSetTurnReady(_ *types.RequireSetupNewTurnEvent)         {}
func (noopTxPool) AddCommitment(_ *proto.SubmitPlayerCommitmentRequest) error { return nil }
func (noopTxPool) AddCard(_ *proto.SubmitPlayerCardRequest) error             { return nil }
func (noopTxPool) ClearGameInfo(_ int64)                                      {}

type nopPublisher struct{}

func (nopPublisher) Publish(_ context.Context, _ *proto.Event) (*proto.PublishResponse, error) {
	return &proto.PublishResponse{Success: true}, nil
}

func (nopPublisher) Topic() string { return pubsub.TopicRoom }

func newTimeoutTestGame() *Game {
	gameArgs := &dao.GameArgs{
		InitialHP:                             3000,
		ConfirmationTimeout:                   60,
		CommitmentSubmissionTimeout:           30,
		CardSubmissionTimeout:                 30,
		ConfirmationTimeoutRedundancy:         10,
		CommitmentSubmissionTimeoutRedundancy: 5,
		CardSubmissionTimeoutRedundancy:       5,
	}
	players := []types.PlayerAddress{
		{Id: 1, TemporaryAddress: "0x1"},
		{Id: 2, TemporaryAddress: "0x2"},
	}
	return NewGame(context.Background(), players, nopPublisher{}, noopTxPool{}, nil, proto.GameType_PVP, gameArgs)
}

func TestTimeoutFromCurrentRound_IgnoresUnknownStatus(t *testing.T) {
	g := newTimeoutTestGame()
	g.gameInfo.Status = proto.GameStatus_GAME_UNKNOWN

	require.Zero(t, g.timeoutFromCurentRound())
}

func setPlayerTurnStatuses(t *testing.T, g *Game, statuses ...proto.PlayerTurnStatus) {
	t.Helper()
	curTurn := g.currentRound.getCurrentTurn()
	require.NotNil(t, curTurn, "current turn required")
	i := 0
	for _, p := range g.currentRound.gamePlayers {
		if i >= len(statuses) {
			break
		}
		pti := p.getCurrentPlayerTurnInfo()
		require.NotNil(t, pti)
		pti.PlayerStatus = statuses[i]
		i++
	}
}

func TestAnyPlayerPendingOnChain_CommitmentsPhase(t *testing.T) {
	g := newTimeoutTestGame()
	g.currentRound.setTurnStatus(proto.TurnStatus_TURN_WAITTING_COMMITMENTS)

	setPlayerTurnStatuses(t, g,
		proto.PlayerTurnStatus_PLAYER_TURN_UNKNOWN,
		proto.PlayerTurnStatus_PLAYER_TURN_UNKNOWN,
	)
	require.False(t, g.anyPlayerPendingOnChain())

	setPlayerTurnStatuses(t, g,
		proto.PlayerTurnStatus_PLAYER_TURN_COMMITMENT_ON_CHAIN,
		proto.PlayerTurnStatus_PLAYER_TURN_COMMITMENT_SUBMITTED,
	)
	require.True(t, g.anyPlayerPendingOnChain())
}

func TestAnyPlayerPendingOnChain_CardsPhase(t *testing.T) {
	g := newTimeoutTestGame()
	g.currentRound.setTurnStatus(proto.TurnStatus_TURN_WAITTING_CARDS)

	setPlayerTurnStatuses(t, g,
		proto.PlayerTurnStatus_PLAYER_TURN_COMMITMENT_ON_CHAIN,
		proto.PlayerTurnStatus_PLAYER_TURN_COMMITMENT_ON_CHAIN,
	)
	require.False(t, g.anyPlayerPendingOnChain())

	setPlayerTurnStatuses(t, g,
		proto.PlayerTurnStatus_PLAYER_TURN_CARD_ON_CHAIN,
		proto.PlayerTurnStatus_PLAYER_TURN_CARD_SUBMITTED,
	)
	require.True(t, g.anyPlayerPendingOnChain())
}

func TestAnyPlayerPendingOnChain_OtherTurnStatuses(t *testing.T) {
	g := newTimeoutTestGame()
	setPlayerTurnStatuses(t, g,
		proto.PlayerTurnStatus_PLAYER_TURN_CARD_SUBMITTED,
		proto.PlayerTurnStatus_PLAYER_TURN_CARD_SUBMITTED,
	)

	g.currentRound.setTurnStatus(proto.TurnStatus_TURN_WAITTING_BATTLE_CONFIRMATION)
	require.False(t, g.anyPlayerPendingOnChain())

	g.currentRound.setTurnStatus(proto.TurnStatus_TURN_WAITTING_SETUP_ON_CHAIN)
	require.False(t, g.anyPlayerPendingOnChain())
}

func TestHandleTimerEvent_CardsPhasePendingOnChain_Aborts(t *testing.T) {
	g := newTimeoutTestGame()
	g.gameInfo.Status = proto.GameStatus_GAME_RUNNING
	g.currentRound.setTurnStatus(proto.TurnStatus_TURN_WAITTING_CARDS)
	setPlayerTurnStatuses(t, g,
		proto.PlayerTurnStatus_PLAYER_TURN_COMMITMENT_ON_CHAIN,
		proto.PlayerTurnStatus_PLAYER_TURN_CARD_SUBMITTED,
	)
	require.True(t, g.anyPlayerPendingOnChain())

	event := &timerEvent{
		GameID:            g.gameInfo.ID,
		currentGameStatus: g.gameInfo.Status,
		currentRound:      g.currentRound.roundNumber,
		currentTurnNumber: g.currentRound.getCurrentTurnNumber(),
		currentTurnStatus: g.currentRound.getTurnStatus(),
	}
	func() {
		defer func() { _ = recover() }()
		g.handleTimerEvent(event)
	}()

	require.Equal(t, proto.GameStatus_GAME_END, g.gameInfo.Status)
	require.NotNil(t, g.gameInfo.GameResult)
	require.Equal(t, proto.GameResultType_GAME_ABORTED, g.gameInfo.GameResult.GameResultType)
}

func TestHandleTimerEvent_CardsPhaseNoPendingOnChain_DoesNotAbort(t *testing.T) {
	g := newTimeoutTestGame()
	g.gameInfo.Status = proto.GameStatus_GAME_RUNNING
	g.currentRound.setTurnStatus(proto.TurnStatus_TURN_WAITTING_CARDS)
	setPlayerTurnStatuses(t, g,
		proto.PlayerTurnStatus_PLAYER_TURN_COMMITMENT_ON_CHAIN,
		proto.PlayerTurnStatus_PLAYER_TURN_COMMITMENT_ON_CHAIN,
	)
	require.False(t, g.anyPlayerPendingOnChain())

	event := &timerEvent{
		GameID:            g.gameInfo.ID,
		currentGameStatus: g.gameInfo.Status,
		currentRound:      g.currentRound.roundNumber,
		currentTurnNumber: g.currentRound.getCurrentTurnNumber(),
		currentTurnStatus: g.currentRound.getTurnStatus(),
	}
	g.handleTimerEvent(event)

	if g.gameInfo.GameResult != nil {
		require.NotEqual(t, proto.GameResultType_GAME_ABORTED, g.gameInfo.GameResult.GameResultType)
	}
	require.NotEqual(t, proto.GameStatus_GAME_END, g.gameInfo.Status)
}

func TestHandleTimerEvent_IgnoresUnknownStatus(t *testing.T) {
	g := newTimeoutTestGame()
	g.gameInfo.Status = proto.GameStatus_GAME_UNKNOWN
	g.currentRound.setTurnStatus(proto.TurnStatus_TURN_WAITTING_SETUP_ON_CHAIN)
	event := &timerEvent{
		GameID:            g.gameInfo.ID,
		currentGameStatus: g.gameInfo.Status,
		currentRound:      g.currentRound.roundNumber,
		currentTurnNumber: g.currentRound.getCurrentTurnNumber(),
		currentTurnStatus: g.currentRound.getTurnStatus(),
	}

	g.handleTimerEvent(event)

	require.Equal(t, proto.GameStatus_GAME_UNKNOWN, g.gameInfo.Status)
	require.Equal(t, proto.TurnStatus_TURN_WAITTING_SETUP_ON_CHAIN, g.currentRound.getTurnStatus())
}
