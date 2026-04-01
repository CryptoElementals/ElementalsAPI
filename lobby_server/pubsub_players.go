package lobbyserver

import (
	"github.com/CryptoElementals/common/lobby_server/worker/queue"
	"github.com/CryptoElementals/common/room_server/worker/types"
)

// lobbyPubSubBots implements rpc/server.PubSubBotHooks for lobby PubSub (queue bot registration only).
type lobbyPubSubBots struct {
	q *queue.Service
}

func (l *lobbyPubSubBots) AddBotPlayer(address types.PlayerAddress) error {
	return l.q.RegisterBots(&address)
}

func (l *lobbyPubSubBots) RemoveBotPlayer(address types.PlayerAddress) {
	_ = l.q.UnregisterBots(&address)
}
