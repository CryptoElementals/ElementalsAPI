package player_info

import (
	"context"

	"github.com/CryptoElementals/common/room_server/worker/types"
)

// Store is the lobby queue and match-state persistence contract (RedisStore, GormStore, …).
type Store interface {
	IsInQueue(ctx context.Context, player types.PlayerAddress) (bool, error)
	QueueJoinedAtMs(ctx context.Context, player types.PlayerAddress) (ms int64, ok bool, err error)
	ListQueuedPlayers(ctx context.Context) ([]types.PlayerAddress, error)
	AddQueue(ctx context.Context, player types.PlayerAddress, nowMs int64) (bool, error)
	RemoveQueue(ctx context.Context, player types.PlayerAddress) error
	SetPendingPair(ctx context.Context, matchID int64, p1, p2 types.PlayerAddress) (bool, error)
	CancelPendingPair(ctx context.Context, matchID int64, p1, p2 types.PlayerAddress) (bool, error)
	FinalizeConfirmedPair(ctx context.Context, matchID int64, p1, p2 types.PlayerAddress) (bool, error)
	MarkPlayersInGame(ctx context.Context, gameID int64, players ...types.PlayerAddress) error
	MarkPlayersOutOfGame(ctx context.Context, players ...types.PlayerAddress) error
	IsInGame(ctx context.Context, player types.PlayerAddress) (bool, error)
	PendingMatchID(ctx context.Context, player types.PlayerAddress) (int64, bool, error)
	JoinQueueOrGetMatchCandidate(ctx context.Context, player types.PlayerAddress, nowMs int64) (*types.PlayerAddress, bool, error)
	FirstWaitingPlayerBefore(ctx context.Context, cutoffMs int64) (*types.PlayerAddress, error)
}

var (
	_ Store = (*RedisStore)(nil)
	_ Store = (*GormStore)(nil)
)
