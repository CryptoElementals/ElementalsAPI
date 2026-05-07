package game

import (
	"context"
	"testing"

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

func (nopPublisher) Publish(_ context.Context, _ *proto.PublishRequest) (*proto.PublishResponse, error) {
	return &proto.PublishResponse{Success: true}, nil
}

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
