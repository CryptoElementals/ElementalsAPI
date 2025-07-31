package game

import (
	"github.com/CryptoElementals/common/room_server/fsm"
	"github.com/CryptoElementals/common/rpc/proto"
)

type gameStatusConverter struct{}

func (gameStatusConverter) String(s proto.GameStatus) string {
	return proto.PlayerStatus_name[int32(s)]
}
func (gameStatusConverter) Parse(s string) proto.GameStatus {
	return proto.GameStatus(proto.PlayerStatus_value[s])
}

type gameFsm struct {
	stateMachine *fsm.FSM[proto.GameStatus, gameStatusConverter]
	game         *Game
}
