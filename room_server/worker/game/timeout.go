package game

import (
	"time"

	"github.com/CryptoElementals/common/log"
	"github.com/CryptoElementals/common/room_server/worker/types"
	"github.com/CryptoElementals/common/rpc/proto"
)

type timerEvent struct {
	currentGameStatus  proto.GameStatus
	currentRound       uint32
	currentTurnNumber  uint32
	currentRoundStatus proto.RoundStatus
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
		timeoutDuration = g.gameInfo.GameArgs.GameMatchTimeout + g.gameInfo.GameArgs.GameMatchTimeoutRedundancy
	case proto.GameStatus_GAME_RUNNING:
		switch g.currentRound.round.Status {
		case proto.RoundStatus_ROUND_WAITTING_BATTLE_CONFIRMATION,
			proto.RoundStatus_ROUND_COMPLETED,
			proto.RoundStatus_ROUND_WAITTING_SETUP_ON_CHAIN:
			// waitting for confimation
			timeoutDuration = g.gameInfo.GameArgs.RoundConfirmTimeout + g.gameInfo.GameArgs.RoundConfirmTimeoutRedundancy
		case proto.RoundStatus_ROUND_WAITTING_COMMITMENTS, proto.RoundStatus_ROUND_WAITTING_CARDS:
			// round submitting cards
			timeoutDuration = g.gameInfo.GameArgs.RoundTimeout + g.gameInfo.GameArgs.RoundTimeoutRedundancy
		}
	case proto.GameStatus_GAME_END:
		return 0
	}

	timeout := time.Second * time.Duration(timeoutDuration)
	if g.currentRound.round.SetupOnChainAt != 0 {
		timeout -= time.Since(time.Unix(g.currentRound.round.SetupOnChainAt, 0))
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
		currentGameStatus:  g.gameInfo.Status,
		currentRound:       g.currentRound.round.RoundNumber,
		currentTurnNumber:  g.getCurrentTurnNumber(),
		currentRoundStatus: g.currentRound.round.Status,
	}
	log.Debugw("send timer event",
		"game id", g.gameInfo.ID,
		"round", timerEvent.currentRound,
		"turn", timerEvent.currentTurnNumber,
		"round status", timerEvent.currentRoundStatus,
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
	if g.currentRound.round.Status != event.currentRoundStatus {
		return
	}
	// turn number changed go ahead
	if g.getCurrentTurnNumber() != event.currentTurnNumber {
		return
	}
	log.Infow("timer event triggered",
		"game id", g.gameInfo.ID,
		"round", g.currentRound.round.RoundNumber,
		"turn", g.getCurrentTurnNumber(),
		"round status", g.currentRound.round.Status,
		"turn number", g.getCurrentTurnNumber(),
		"game status", g.gameInfo.Status)
	// game init only exists at the very beginning, once both players confirms, it turns to game running
	if g.gameInfo.Status == proto.GameStatus_GAME_INIT {
		err := g.handleGameAbortInit()
		if err != nil {
			log.Errorf("abort game failed, err: %s", err.Error())
		}
		return
	}
	switch g.currentRound.round.Status {
	case proto.RoundStatus_ROUND_WAITTING_SETUP_ON_CHAIN, proto.RoundStatus_ROUND_WAITTING_BATTLE_CONFIRMATION:
		// For timeout during setup or battle confirmation, abort the game
		g.handleGameAbortInternalError()
	case proto.RoundStatus_ROUND_WAITTING_COMMITMENTS, proto.RoundStatus_ROUND_WAITTING_CARDS:
		g.handleTurnEnd()
	}
}
