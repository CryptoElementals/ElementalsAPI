package queue

import (
	"encoding/json"
	"time"

	"github.com/CryptoElementals/common/log"
	"github.com/CryptoElementals/common/room_server/worker/types"
	"github.com/CryptoElementals/common/timer"
)

// Set is a small in-memory set for bot membership during a single settlement pass.
type Set[T comparable] map[T]struct{}

func (s Set[T]) Add(value T) {
	s[value] = struct{}{}
}

func (s Set[T]) Contains(value T) bool {
	_, exists := s[value]
	return exists
}

func (q *Queue) isBotLocked(addr types.PlayerAddress) bool {
	isBot, err := q.botStore.IsBot(addr)
	if err != nil {
		log.Errorw("redis is bot check failed", "player", addr.String(), "err", err)
		return false
	}
	return isBot
}

func (q *Queue) popBotForMatchLocked() (types.PlayerAddress, bool) {
	addr, err := q.botStore.PopFreshIdleBotForMatch(time.Now().UnixMilli(), q.botFreshness.Milliseconds())
	if err != nil {
		log.Errorw("redis pop fresh idle bot failed", "err", err)
		return types.PlayerAddress{}, false
	}
	if addr == nil {
		return types.PlayerAddress{}, false
	}
	return *addr, true
}

func (q *Queue) releaseInGameBotLocked(addr types.PlayerAddress) bool {
	ok, err := q.botStore.ReleaseInGameBot(addr, time.Now().UnixMilli(), q.botFreshness.Milliseconds())
	if err != nil {
		log.Errorw("redis release in-game bot failed", "player", addr.String(), "err", err)
		return false
	}
	return ok
}

func (q *Queue) firstWaitingPlayerForBotLocked() (*types.PlayerAddress, error) {
	cutoff := time.Now().Add(-q.botWaitTime).UnixMilli()
	return q.lobbyState.FirstWaitingPlayerBefore(q.ctx, cutoff)
}

// botDispatchTickEvent schedules periodic bot-vs-human matchmaking for long-waiting players.
type botDispatchTickEvent struct{}

func (e *botDispatchTickEvent) EventType() string { return "queue_bot_dispatch_tick" }

func (e *botDispatchTickEvent) Marshal() []byte {
	b, _ := json.Marshal(e)
	return b
}

func (e *botDispatchTickEvent) Unmarshal(data []byte) error {
	return json.Unmarshal(data, e)
}

func (e *botDispatchTickEvent) String() string { return "queue_bot_dispatch_tick" }

func (q *Queue) registerBotDispatchTickHandler() {
	_ = timer.RegisterHandler(timer.ScopeLobby, &botDispatchTickEvent{}, func(_ timer.TimerEvent) error {
		return q.handleBotDispatchTick()
	})
}

func (q *Queue) scheduleNextBotDispatchTick() {
	if q.botWaitTime <= 0 {
		return
	}
	if err := timer.ProcessIn(timer.ScopeLobby, q.botWaitTime, &botDispatchTickEvent{}, true); err != nil {
		log.Errorw("schedule bot dispatch tick failed", "err", err)
	}
}

func (q *Queue) handleBotDispatchTick() error {
	if q.ctx.Err() != nil {
		return nil
	}
	q.lock.Lock()
	for {
		player, err := q.firstWaitingPlayerForBotLocked()
		if err != nil {
			log.Errorw("find waiting player for bot failed", "err", err)
			break
		}
		if player == nil {
			break
		}
		botPlayer, ok := q.popBotForMatchLocked()
		if !ok {
			break
		}
		log.Infow("found long waitting player, dispatch a bot", "player", player.String(), "bot", botPlayer.String())
		err = q.matchPlayers([]types.PlayerAddress{botPlayer, *player})
		if err != nil {
			log.Errorw("error match bot with player", "err", err, "bot", botPlayer.String(), "player", player.String())
			break
		}
	}
	q.lock.Unlock()

	if q.ctx.Err() == nil {
		q.scheduleNextBotDispatchTick()
	}
	return nil
}

func (q *Queue) addBotRoutine() {
	if q.botWaitTime <= 0 {
		return
	}
	q.scheduleNextBotDispatchTick()
}
