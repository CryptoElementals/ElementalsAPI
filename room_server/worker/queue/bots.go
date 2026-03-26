package queue

import (
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

type botManager struct {
	idle   Set[types.PlayerAddress]
	inGame Set[types.PlayerAddress]
}

func newBotManager() *botManager {
	return &botManager{
		idle:   make(Set[types.PlayerAddress]),
		inGame: make(Set[types.PlayerAddress]),
	}
}

func (m *botManager) isBot(addr types.PlayerAddress) bool {
	return m.inGame.Contains(addr) || m.idle.Contains(addr)
}

func (m *botManager) addBot(addr types.PlayerAddress) {
	m.idle.Add(addr)
}

func (m *botManager) removeBot(addr types.PlayerAddress) {
	m.idle.Remove(addr)
	m.inGame.Remove(addr)
}

func (m *botManager) popBotForMatch() (types.PlayerAddress, bool) {
	addr, ok := m.idle.PopRandom()
	if !ok {
		return addr, ok
	}
	m.inGame.Add(addr)
	return addr, ok
}

func (m *botManager) releaseInGameBot(addr types.PlayerAddress) bool {
	if m.inGame.Contains(addr) {
		m.inGame.Remove(addr)
		m.idle.Add(addr)
		return true
	}
	return false
}

func (m *botManager) inGameCount() int {
	return m.inGame.Size()
}

func (m *botManager) idleCount() int {
	return m.idle.Size()
}

func (m *botManager) isIdle(addr types.PlayerAddress) bool {
	return m.idle.Contains(addr)
}

func (m *botManager) isInGame(addr types.PlayerAddress) bool {
	return m.inGame.Contains(addr)
}

func (q *Queue) RegisterBots(addrs ...*types.PlayerAddress) error {
	q.lock.Lock()
	defer q.lock.Unlock()
	log.Infow("register bots", "bots", types.ToJsonLoggable(addrs))
	for _, addr := range addrs {
		q.botMgr.addBot(*addr)
	}
	return nil
}

func (q *Queue) UnregisterBots(addrs ...*types.PlayerAddress) error {
	q.lock.Lock()
	defer q.lock.Unlock()
	log.Infow("unregister bots", "bots", types.ToJsonLoggable(addrs))
	for _, addr := range addrs {
		q.botMgr.removeBot(*addr)
	}
	return nil
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
				for player, joinQueueTime := range q.queue {
					if q.botMgr.idleCount() == 0 {
						break
					}
					waittingTime := time.Since(joinQueueTime)
					if waittingTime >= q.botWaitTime {
						botPlayer, _ := q.botMgr.popBotForMatch()
						log.Infow("found long waitting player, dispatch a bot", "player", player.String(), "waitting seconds", int(waittingTime.Seconds()), "bot", botPlayer.String())
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
