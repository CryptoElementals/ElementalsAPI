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

func (q *Queue) RegisterBots(addrs ...*types.PlayerAddress) error {
	q.lock.Lock()
	defer q.lock.Unlock()
	log.Infow("register bots", "bots", types.ToJsonLoggable(addrs))
	bots := make([]types.PlayerAddress, 0, len(addrs))
	for _, addr := range addrs {
		bots = append(bots, *addr)
	}
	if err := q.botStore.RegisterBots(bots...); err != nil {
		log.Errorw("redis register bots failed", "err", err)
		return err
	}
	return nil
}

func (q *Queue) UnregisterBots(addrs ...*types.PlayerAddress) error {
	q.lock.Lock()
	defer q.lock.Unlock()
	log.Infow("unregister bots", "bots", types.ToJsonLoggable(addrs))
	bots := make([]types.PlayerAddress, 0, len(addrs))
	for _, addr := range addrs {
		bots = append(bots, *addr)
	}
	if err := q.botStore.UnregisterBots(bots...); err != nil {
		log.Errorw("redis unregister bots failed", "err", err)
		return err
	}
	return nil
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
	addr, err := q.botStore.PopIdleBotForMatch()
	if err != nil {
		log.Errorw("redis pop idle bot failed", "err", err)
		return types.PlayerAddress{}, false
	}
	if addr == nil {
		return types.PlayerAddress{}, false
	}
	return *addr, true
}

func (q *Queue) releaseInGameBotLocked(addr types.PlayerAddress) bool {
	ok, err := q.botStore.ReleaseInGameBot(addr)
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
