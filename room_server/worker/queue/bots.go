package queue

import (
	"fmt"
	"time"

	"github.com/CryptoElementals/common/log"
	"github.com/CryptoElementals/common/room_server/worker/types"
)

type Set[T comparable] map[T]struct{}

func NewSet[T comparable]() Set[T] {
	return make(Set[T])
}

func (s Set[T]) Add(value T) {
	s[value] = struct{}{}
}

func (s Set[T]) Remove(value T) {
	delete(s, value)
}

func (s Set[T]) Contains(value T) bool {
	_, exists := s[value]
	return exists
}

func (s Set[T]) Size() int {
	return len(s)
}

func (s Set[T]) Clear() {
	for k := range s {
		delete(s, k)
	}
}

func (s Set[T]) ToSlice() []T {
	slice := make([]T, 0, len(s))
	for k := range s {
		slice = append(slice, k)
	}
	return slice
}

func (s Set[T]) RandomElement() (T, bool) {
	for t := range s {
		return t, true
	}
	return *new(T), false
}

func (s Set[T]) PopRandom() (T, bool) {
	for t := range s {
		delete(s, t)
		return t, true
	}
	return *new(T), false
}

func (q *Queue) RegisterBots(addrs ...*types.PlayerAddress) error {
	q.lock.Lock()
	defer q.lock.Unlock()
	log.Infow("register bots", "bots", types.ToJsonLoggable(addrs))
	for _, addr := range addrs {
		q.botsSet.Add(*addr)
		err := q.lockToken(addr)
		if err != nil {
			return fmt.Errorf("lock token failed, err: %w, addr: %s", err, addr.String())
		}
	}
	return nil
}

func (q *Queue) UnregisterBots(addrs ...*types.PlayerAddress) error {
	q.lock.Lock()
	defer q.lock.Unlock()
	log.Infow("unregister bots", "bots", types.ToJsonLoggable(addrs))
	for _, addr := range addrs {
		if !q.botsSet.Contains(*addr) {
			continue
		}
		q.botsSet.Remove(*addr)
		err := q.unlockToken(addr)
		if err != nil {
			return fmt.Errorf("lock token failed, err: %w, addr: %s", err, addr.String())
		}
	}
	return nil
}

func (q *Queue) addBotRoutine() {
	go func() {
		ticker := time.NewTicker(30 * time.Second)
		for {
			select {
			case <-q.ctx.Done():
				return
			case <-ticker.C:
				q.lock.Lock()
				for player, joinQueueTime := range q.queue {
					if q.botsSet.Size() == 0 {
						break
					}
					waittingTime := time.Since(joinQueueTime)
					if waittingTime >= 30*time.Second {
						botPlayer, _ := q.botsSet.PopRandom()
						log.Infow("find long waitting player, dispatch a bot", "player", player.String(), "waitting seconds", int(waittingTime.Seconds()), "bot", botPlayer.String())
						err := q.matchPlayers([]types.PlayerAddress{botPlayer, player})
						if err != nil {
							log.Errorw("error match bot with player", "err", err, "bot", botPlayer.String(), "player", player.String())
							break
						}
					}
				}
				q.lock.Unlock()
			}
		}
	}()
}
