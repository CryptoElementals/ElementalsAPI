package queue

import (
	"time"

	"github.com/CryptoElementals/common/log"
	"github.com/CryptoElementals/common/room_server/worker/types"
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

func (q *Queue) addBotRoutine() {
	if q.botWaitTime <= 0 {
		return
	}
	go func() {
		ticker := time.NewTicker(q.botWaitTime)
		for {
			select {
			case <-q.ctx.Done():
				return
			case <-ticker.C:
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
			}
		}
	}()
}
