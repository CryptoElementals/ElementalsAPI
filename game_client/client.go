package gameclient

import (
	"context"

	"github.com/CryptoElementals/common/room_server/worker/types"
	"github.com/CryptoElementals/common/rpc/proto"
)

type GameClient interface {
	ConfirmBattle(ctx context.Context, addr *types.PlayerAddress, gameID uint, roundNumber uint) error
	ContinueGame(ctx context.Context, addr *types.PlayerAddress, gameID uint) error
	ExitQueue(ctx context.Context, addr *types.PlayerAddress) error
	GetBattleInfo(ctx context.Context, gameID uint, roundNumber uint) (*proto.GetBattleInfoResponse, error)
	GetGamePhase(ctx context.Context, addr *types.PlayerAddress) (*proto.GamePhase, error)
	JoinQueue(ctx context.Context, addr *types.PlayerAddress) error
	RefuseContinueGame(ctx context.Context, addr *types.PlayerAddress, gameID uint) error
	Subscribe(topic string, subscriberID string, evtChan chan *proto.Event, errChan chan error) error
}
