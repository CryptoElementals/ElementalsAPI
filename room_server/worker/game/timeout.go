package game

import (
	"time"

	"github.com/CryptoElementals/common/log"
	"github.com/CryptoElementals/common/room_server/worker/types"
	"github.com/CryptoElementals/common/rpc/proto"
)

type timerEvent struct {
	currentGameStatus proto.GameStatus
	currentRound      uint32
	currentTurnNumber uint32
	currentTurnStatus proto.TurnStatus
}

// timeoutFromCurentRound calculates the timeout duration for the current round state
func (g *Game) timeoutFromCurentRound() time.Duration {
	if g.gameInfo.Status == proto.GameStatus_GAME_END {
		return 0
	}
	timeoutDuration := int64(0)
	switch g.gameInfo.Status {
	case proto.GameStatus_GAME_INIT:
		// game waitting confirmed for the first round
		timeoutDuration = g.gameInfo.GameArgs.ConfirmationTimeout + g.gameInfo.GameArgs.ConfirmationTimeoutRedundancy
	case proto.GameStatus_GAME_RUNNING:
		switch g.currentRound.turnStatus {
		case proto.TurnStatus_TURN_WAITTING_BATTLE_CONFIRMATION,
			proto.TurnStatus_TURN_ROUND_COMPLETED,
			proto.TurnStatus_TURN_WAITTING_SETUP_ON_CHAIN:
			// waitting for confimation
			timeoutDuration = g.gameInfo.GameArgs.ConfirmationTimeout + g.gameInfo.GameArgs.ConfirmationTimeoutRedundancy
		case proto.TurnStatus_TURN_WAITTING_COMMITMENTS:
			// turn submitting commitments
			timeoutDuration = g.gameInfo.GameArgs.CommitmentSubmissionTimeout + g.gameInfo.GameArgs.CommitmentSubmissionTimeoutRedundancy
		case proto.TurnStatus_TURN_WAITTING_CARDS:
			// turn submitting cards
			timeoutDuration = g.gameInfo.GameArgs.CardSubmissionTimeout + g.gameInfo.GameArgs.CardSubmissionTimeoutRedundancy
		}
	case proto.GameStatus_GAME_END:
		return 0
	}

	timeout := time.Second * time.Duration(timeoutDuration)
	currentTurn := g.currentRound.getCurrentTurn()
	if currentTurn != nil && currentTurn.TurnStartAt > 0 {
		timeout -= time.Since(time.Unix(currentTurn.TurnStartAt, 0))
	}
	return timeout
}

// sendTimerEventByCurrentRound schedules a timer event based on the current round state
func (g *Game) sendTimerEventByCurrentRound() {
	timeout := g.timeoutFromCurentRound()
	if timeout == 0 {
		return
	}
	timerEvent := &timerEvent{
		currentGameStatus: g.gameInfo.Status,
		currentRound:      g.currentRound.round.RoundNumber,
		currentTurnNumber: g.currentRound.getCurrentTurnNumber(),
		currentTurnStatus: g.currentRound.turnStatus,
	}
	log.Debugw("send timer event",
		"game id", g.gameInfo.ID,
		"round", timerEvent.currentRound,
		"turn", timerEvent.currentTurnNumber,
		"turn status", timerEvent.currentTurnStatus,
		"timeout", timeout.Seconds(),
	)
	time.AfterFunc(timeout, func() {
		g.workerMangerService.SendEvent(g.workerID(), types.NewEvent(g.workerID(), timerEvent))
	})
}

// handleTimerEvent handles timeout events for the game
func (g *Game) handleTimerEvent(event *timerEvent) {
	if g.gameInfo.Status == proto.GameStatus_GAME_END {
		return
	}
	// stale event
	if g.currentRound.round.RoundNumber != event.currentRound {
		return
	}
	// status changed go ahead
	if g.currentRound.turnStatus != event.currentTurnStatus {
		return
	}
	// turn number changed go ahead
	if g.currentRound.getCurrentTurnNumber() != event.currentTurnNumber {
		return
	}
	log.Infow("timer event triggered",
		"game id", g.gameInfo.ID,
		"round", g.currentRound.round.RoundNumber,
		"turn", g.currentRound.getCurrentTurnNumber(),
		"turn status", g.currentRound.turnStatus,
		"turn number", g.currentRound.getCurrentTurnNumber(),
		"game status", g.gameInfo.Status)
	// game init only exists at the very beginning, once both players confirms, it turns to game running
	if g.gameInfo.Status == proto.GameStatus_GAME_INIT {
		err := g.handleGameAbortInit()
		if err != nil {
			log.Errorf("abort game failed, err: %s", err.Error())
		}
		return
	}
	switch g.currentRound.turnStatus {
	case proto.TurnStatus_TURN_WAITTING_SETUP_ON_CHAIN:
		// For timeout during setup or battle confirmation, abort the game
		err := g.handleGameAbortInternalError()
		if err != nil {
			log.Errorf("abort game failed, err: %s", err.Error())
		}
		return
	case proto.TurnStatus_TURN_WAITTING_COMMITMENTS,
		proto.TurnStatus_TURN_WAITTING_CARDS,
		proto.TurnStatus_TURN_WAITTING_BATTLE_CONFIRMATION:
		err := g.handleTurnEnd()
		if err != nil {
			log.Errorf("handle turn end failed, err: %s", err.Error())
		}
		return
	case proto.TurnStatus_TURN_ROUND_COMPLETED:
		log.Infow("turn round completed, but get timer event",
			"game id", g.gameInfo.ID,
			"round", g.currentRound.round.RoundNumber,
			"turn", g.currentRound.getCurrentTurnNumber())
		err := g.handleTurnEnd()
		if err != nil {
			log.Errorf("handle turn end failed, err: %s", err.Error())
		}
		return
	}
}
