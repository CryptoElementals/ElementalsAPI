package game

import (
	"github.com/CryptoElementals/common/rpc/proto"
	"github.com/CryptoElementals/common/worker"
)

type RoomState = proto.GameStatus

type Game struct {
	id              string
	contractAddress string
	roomWorker      *worker.Worker
	status          *proto.GameInfo
	players         []*proto.PlayerAddress
}

func NewGame(id string, players []*proto.PlayerAddress) *Game {
	return &Game{
		id:      id,
		players: players,
	}
}

func (g *Game) SaveGameStatus(status *proto.GameInfo) {
	g.status = status
}