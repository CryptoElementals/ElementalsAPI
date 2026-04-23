package queue

import (
	"encoding/json"
	"fmt"

	"github.com/CryptoElementals/common/db"
	"github.com/CryptoElementals/common/log"
	dao "github.com/CryptoElementals/common/models"
	"github.com/CryptoElementals/common/room_server/worker/types"
	"github.com/CryptoElementals/common/timer"
)

// pendingMatchConfirmationTimeoutEvent fires when a pending game_match was not confirmed in time (queue PVP or continue rematch).
type pendingMatchConfirmationTimeoutEvent struct {
	MatchID int64 `json:"match_id"`
}

func (e *pendingMatchConfirmationTimeoutEvent) EventType() string {
	return "pending_match_confirmation_timeout"
}

func (e *pendingMatchConfirmationTimeoutEvent) Marshal() []byte {
	b, _ := json.Marshal(e)
	return b
}

func (e *pendingMatchConfirmationTimeoutEvent) Unmarshal(data []byte) error {
	return json.Unmarshal(data, e)
}

func (e *pendingMatchConfirmationTimeoutEvent) String() string {
	return fmt.Sprintf("pending_match_confirmation_timeout{match=%d}", e.MatchID)
}

func (q *Queue) registerPendingMatchConfirmationTimeoutHandler() {
	_ = timer.RegisterHandler(timer.ScopeLobby, &pendingMatchConfirmationTimeoutEvent{}, func(evt timer.TimerEvent) error {
		return q.handlePendingMatchConfirmationTimeout(evt.(*pendingMatchConfirmationTimeoutEvent))
	})
}

func (q *Queue) handlePendingMatchConfirmationTimeout(evt *pendingMatchConfirmationTimeoutEvent) error {
	q.abortPendingMatch(evt.MatchID, true, true)
	return nil
}

// abortPendingMatch cancels DB row if still pending, clears Redis pending pair, unlocks human tokens, optionally notifies TYPE_MATCH_CANCELED.
func (q *Queue) abortPendingMatch(matchID int64, notifyMatchCanceled bool, fromTimeout bool) error {
	m, err := db.GetGameMatchByID(q.ctx, matchID)
	if err != nil {
		log.Errorw("abort pending match: load game_match", "match_id", matchID, "err", err)
		return err
	}
	if m.Status != dao.GameMatchStatusPending {
		return fmt.Errorf("Game status not GameMatchStatusPending")
	}
	players := []types.PlayerAddress{
		*types.NewPlayerAddress(m.Player1ID, m.Player1TempAddress),
		*types.NewPlayerAddress(m.Player2ID, m.Player2TempAddress),
	}
	if err := q.lobbyState.CancelPendingPair(q.ctx, matchID); err != nil {
		log.Errorw("cancel pending match (game_match + queue) failed", "match_id", matchID, "err", err)
		return err
	}
	for _, p := range players {
		if q.isPlayerBot(p) {
			continue
		}
		if err := q.unlockToken(&p); err != nil {
			log.Errorw("unlock token after abort pending match", "player", p.String(), "err", err)
			return err
		}
	}
	if notifyMatchCanceled {
		q.publishMatchCanceled(matchID, players, fromTimeout)
	}
	return nil
}

// tryStartContinueRematchAfterGame runs after GameResultSettlement for a finished human-vs-human game: lock tokens, insert continue game_match, publish TYPE_MATCHED with LastGameId, schedule cancel timer. Do not call when the game included bots.
func (q *Queue) tryStartContinueRematchAfterGame(gameID int64, gr *dao.GameResult) {
	if gr == nil || len(gr.PlayerResultInfos) != 2 {
		return
	}
	players := make([]types.PlayerAddress, 0, 2)
	for _, pri := range gr.PlayerResultInfos {
		if pri == nil {
			return
		}
		addr := types.NewPlayerAddress(pri.PlayerId, pri.TemporaryAddress)
		players = append(players, *addr)
	}
	var locked []types.PlayerAddress
	for _, pl := range players {
		if err := q.lockToken(&pl); err != nil {
			for _, u := range locked {
				_ = q.unlockToken(&u)
			}
			log.Errorw("continue rematch lock token failed", "game_id", gameID, "player", pl.String(), "err", err)
			return
		}
		locked = append(locked, pl)
	}
	matchID, err := q.createContinueRematchMatch(players, gameID)
	if err != nil {
		for _, u := range locked {
			_ = q.unlockToken(&u)
		}
		log.Errorw("continue rematch create failed", "game_id", gameID, "err", err)
		return
	}
	q.schedulePendingMatchConfirmationTimeout(matchID, q.continueRematchCancelTimeoutSec, q.continueRematchCancelRedundancySec)
}
