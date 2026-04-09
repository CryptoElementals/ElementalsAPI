package queue

import (
	"encoding/json"
	"fmt"
	"time"

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

func (e *pendingMatchConfirmationTimeoutEvent) EventType() string { return "pending_match_confirmation_timeout" }

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

func (q *Queue) schedulePendingMatchConfirmationTimeout(matchID int64, timeoutSec, redundancySec int64) {
	if timeoutSec <= 0 {
		return
	}
	d := time.Duration(timeoutSec+redundancySec) * time.Second
	if err := timer.ProcessIn(timer.ScopeLobby, d, &pendingMatchConfirmationTimeoutEvent{MatchID: matchID}); err != nil {
		log.Errorw("schedule pending match confirmation timeout failed", "match_id", matchID, "err", err)
	}
}

func (q *Queue) handlePendingMatchConfirmationTimeout(evt *pendingMatchConfirmationTimeoutEvent) error {
	q.lock.Lock()
	defer q.lock.Unlock()
	q.abortPendingMatchLocked(evt.MatchID, true, true)
	return nil
}

// abortPendingMatchLocked clears in-memory pending state, cancels DB row if still pending, unlocks human tokens, optionally notifies TYPE_MATCH_CANCELED. Caller must hold q.lock.
func (q *Queue) abortPendingMatchLocked(matchID int64, notifyMatchCanceled bool, fromTimeout bool) {
	var players []types.PlayerAddress
	for p, mid := range q.pendingMatchByPlayer {
		if mid == matchID {
			players = append(players, p)
		}
	}
	if len(players) == 0 {
		return
	}
	m, err := db.GetGameMatchByID(q.ctx, matchID)
	if err != nil {
		log.Errorw("abort pending match: load game_match", "match_id", matchID, "err", err)
	}
	if m != nil && m.Status == dao.GameMatchStatusPending {
		if err := db.CancelPendingGameMatch(q.ctx, matchID); err != nil {
			log.Errorw("cancel pending game_match failed", "match_id", matchID, "err", err)
		}
	}
	for _, p := range players {
		delete(q.pendingMatchByPlayer, p)
	}
	for _, p := range players {
		if q.botMgr.isBot(p) {
			continue
		}
		if err := q.unlockToken(&p); err != nil {
			log.Errorw("unlock token after abort pending match", "player", p.String(), "err", err)
		}
	}
	if notifyMatchCanceled {
		q.publishMatchCanceled(matchID, players, fromTimeout)
	}
}

// tryStartContinueRematchAfterGame runs after GameResultSettlement for a finished game: lock tokens, insert continue game_match, publish TYPE_MATCHED with LastGameId, schedule cancel timer. Caller must hold q.lock.
func (q *Queue) tryStartContinueRematchAfterGame(game *dao.Game, bots Set[types.PlayerAddress]) {
	if len(game.Players) != 2 {
		return
	}
	hasHuman := false
	players := make([]types.PlayerAddress, 0, 2)
	for _, p := range game.Players {
		if p == nil {
			return
		}
		addr := types.NewPlayerAddress(p.PlayerId, p.TemporaryAddress)
		players = append(players, *addr)
		if !bots.Contains(*addr) {
			hasHuman = true
		}
	}
	if !hasHuman {
		return
	}
	var locked []types.PlayerAddress
	for _, pl := range players {
		if q.botMgr.isBot(pl) {
			continue
		}
		if err := q.lockToken(&pl); err != nil {
			for _, u := range locked {
				_ = q.unlockToken(&u)
			}
			log.Errorw("continue rematch lock token failed", "game_id", game.ID, "player", pl.String(), "err", err)
			return
		}
		locked = append(locked, pl)
	}
	matchID, err := q.createContinueRematchMatch(players, game.ID)
	if err != nil {
		for _, u := range locked {
			_ = q.unlockToken(&u)
		}
		log.Errorw("continue rematch create failed", "game_id", game.ID, "err", err)
		return
	}
	q.schedulePendingMatchConfirmationTimeout(matchID, q.continueRematchCancelTimeoutSec, q.continueRematchCancelRedundancySec)
}
