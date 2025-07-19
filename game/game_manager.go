package game

import (
	"context"

	"github.com/CryptoElementals/common/worker"
)

type GameManager struct {
	ctx   context.Context
	rooms map[string]*worker.Worker
}

func NewGameManager(ctx context.Context) *GameManager {
	return &GameManager{
		ctx:   ctx,
		rooms: make(map[string]*worker.Worker),
	}
}

// func (r *GameManager) CreateGame(id string, players []*proto.PlayerAddress) *worker.Worker {
// 	room := &Game{
// 		id:      id,
// 		players: players,
// 	}
// 	r.rooms[id] = room.roomWorker
// 	return room
// }
