package game

import (
	"context"

	"github.com/CryptoElementals/common/rpc/proto"
	"github.com/CryptoElementals/common/worker"
)

type GameManager struct {
	ctx   context.Context
	rooms map[string]*worker.Worker
}

func NewGameManager(ctx context.Context) *RoomManager {
	return &RoomManager{
		ctx:   ctx,
		rooms: make(map[string]*worker.Worker),
	}
}

func (r *RoomManager) CreateGame(id string, players []*proto.PlayerAddress) *Game {
	room := &Game{
		id:      id,
		players: players,
	}
	r.rooms[id] = room.roomWorker
	return room
}
