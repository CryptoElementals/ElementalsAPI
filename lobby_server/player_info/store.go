package player_info

import (
	"context"
	"time"

	"github.com/CryptoElementals/common/room_server/worker/types"
)

type Store interface {
	IsInQueue(ctx context.Context, player types.PlayerAddress) (bool, error)
	GetGameIDByPlayer(ctx context.Context, player types.PlayerAddress) (bool, int64, error)
	QueueJoinedAtMs(ctx context.Context, player types.PlayerAddress) (ms int64, ok bool, err error)
	MatchPlayersOrJoinQueue(ctx context.Context, player types.PlayerAddress) (matchID int64, err error)
	CountLongWaittingPlayers(ctx context.Context, waitting time.Duration) (int, error)
	MatchPlayerWithBot(ctx context.Context, bot types.PlayerAddress, waitting time.Duration) (int64, error)
	MatchPlayers(ctx context.Context, p1, p2 types.PlayerAddress) (int64, error)
	RemoveQueue(ctx context.Context, player types.PlayerAddress) error
	PendingMatchID(ctx context.Context, player types.PlayerAddress) (int64, bool, error)
	MarkPlayersInGame(ctx context.Context, gameID int64, players ...types.PlayerAddress) error
	MarkPlayersOutOfGame(ctx context.Context, players ...types.PlayerAddress) error
	CancelPendingPair(context.Context, int64) error
}

var _ Store = (*GormStore)(nil)
