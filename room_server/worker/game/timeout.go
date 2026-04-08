package game

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/CryptoElementals/common/log"
	"github.com/CryptoElementals/common/rpc/proto"
	"github.com/CryptoElementals/common/timer"
)

type timerEvent struct {
	GameID            int64
	currentGameStatus proto.GameStatus
	currentRound      uint32
	currentTurnNumber uint32
	currentTurnStatus proto.TurnStatus
}

// timerEvent implements timer.TimerEvent so it can be scheduled via timer.ProcessIn.

func (e *timerEvent) EventType() string { return "game_timer" }

func (e *timerEvent) Marshal() []byte {
	data, _ := json.Marshal(struct {
		GameID            int64            `json:"game_id"`
		CurrentGameStatus proto.GameStatus `json:"current_game_status"`
		CurrentRound      uint32           `json:"current_round"`
		CurrentTurnNumber uint32           `json:"current_turn_number"`
		CurrentTurnStatus proto.TurnStatus `json:"current_turn_status"`
	}{e.GameID, e.currentGameStatus, e.currentRound, e.currentTurnNumber, e.currentTurnStatus})
	return data
}

func (e *timerEvent) Unmarshal(data []byte) error {
	aux := struct {
		GameID            int64            `json:"game_id"`
		CurrentGameStatus proto.GameStatus `json:"current_game_status"`
		CurrentRound      uint32           `json:"current_round"`
		CurrentTurnNumber uint32           `json:"current_turn_number"`
		CurrentTurnStatus proto.TurnStatus `json:"current_turn_status"`
	}{}
	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}
	e.GameID = aux.GameID
	e.currentGameStatus = aux.CurrentGameStatus
	e.currentRound = aux.CurrentRound
	e.currentTurnNumber = aux.CurrentTurnNumber
	e.currentTurnStatus = aux.CurrentTurnStatus
	return nil
}

func (e *timerEvent) String() string {
	return fmt.Sprintf("game_timer{game=%d, round=%d, turn=%d, status=%s}",
		e.GameID, e.currentRound, e.currentTurnNumber, e.currentTurnStatus)
}

// registerTimerFunction registers the game timer handler with the global timer package.
// The handler routes the fired event directly into GameManager handling path.
func (m *GameManager) registerTimerFunction() {
	_ = timer.RegisterHandler(timer.ScopeRoom, &timerEvent{}, func(evt timer.TimerEvent) error {
		te := evt.(*timerEvent)
		return m.HandleTimerEvent(m.ctx, te)
	})
}

// timeoutFromCurentRound calculates the timeout duration for the current round state
func (g *Game) timeoutFromCurentRound() time.Duration {
	if g.gameInfo.Status == proto.GameStatus_GAME_END {
		return 0
	}
	ga := g.gameInfo.GameArgs
	timeoutDuration := int64(0)
	switch g.gameInfo.Status {
	case proto.GameStatus_GAME_INIT:
		// game waitting confirmed for the first round
		timeoutDuration = ga.ConfirmationTimeout + ga.ConfirmationTimeoutRedundancy
	case proto.GameStatus_GAME_RUNNING:
		switch g.currentRound.getTurnStatus() {
		case proto.TurnStatus_TURN_WAITTING_BATTLE_CONFIRMATION,
			proto.TurnStatus_TURN_ROUND_COMPLETED,
			proto.TurnStatus_TURN_WAITTING_SETUP_ON_CHAIN:
			// waitting for confimation
			timeoutDuration = ga.ConfirmationTimeout + ga.ConfirmationTimeoutRedundancy
		case proto.TurnStatus_TURN_WAITTING_COMMITMENTS:
			// turn submitting commitments
			timeoutDuration = ga.CommitmentSubmissionTimeout + ga.CommitmentSubmissionTimeoutRedundancy
		case proto.TurnStatus_TURN_WAITTING_CARDS:
			// turn submitting cards
			timeoutDuration = ga.CardSubmissionTimeout + ga.CardSubmissionTimeoutRedundancy
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
		GameID:            g.gameInfo.ID,
		currentGameStatus: g.gameInfo.Status,
		currentRound:      g.currentRound.roundNumber,
		currentTurnNumber: g.currentRound.getCurrentTurnNumber(),
		currentTurnStatus: g.currentRound.getTurnStatus(),
	}
	log.Debugw("send timer event",
		"game id", g.gameInfo.ID,
		"round", timerEvent.currentRound,
		"turn", timerEvent.currentTurnNumber,
		"turn status", timerEvent.currentTurnStatus,
		"timeout", timeout.Seconds(),
	)
	if err := timer.ProcessIn(timer.ScopeRoom, timeout, timerEvent); err != nil {
		log.Errorw("schedule game timer failed", "game id", g.gameInfo.ID, "err", err)
	}
}

// handleTimerEvent handles timeout events for the game
func (g *Game) handleTimerEvent(event *timerEvent) {
	if g.gameInfo.Status == proto.GameStatus_GAME_END {
		return
	}
	// stale event
	if g.currentRound.roundNumber != event.currentRound {
		return
	}
	// status changed go ahead
	if g.currentRound.getTurnStatus() != event.currentTurnStatus {
		return
	}
	// turn number changed go ahead
	if g.currentRound.getCurrentTurnNumber() != event.currentTurnNumber {
		return
	}
	log.Infow("timer event triggered",
		"game id", g.gameInfo.ID,
		"round", g.currentRound.roundNumber,
		"turn", g.currentRound.getCurrentTurnNumber(),
		"turn status", g.currentRound.getTurnStatus(),
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
	switch g.currentRound.getTurnStatus() {
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
			"round", g.currentRound.roundNumber,
			"turn", g.currentRound.getCurrentTurnNumber())
		err := g.handleTurnEnd()
		if err != nil {
			log.Errorf("handle turn end failed, err: %s", err.Error())
		}
		return
	}
}
