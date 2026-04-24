package player_info

import (
	"context"
	"time"

	dao "github.com/CryptoElementals/common/models"
	"github.com/CryptoElementals/common/room_server/worker/types"
)

type Store interface {
	IsInQueue(ctx context.Context, player types.PlayerAddress) (bool, error)
	GetGameIDByPlayer(ctx context.Context, player types.PlayerAddress) (bool, int64, error)
	QueueJoinedAtMs(ctx context.Context, player types.PlayerAddress) (ms int64, ok bool, err error)
	// MatchPlayersOrJoinQueue returns the new pending game_match when a pair is formed, or nil if the player was only enqueued.
	MatchPlayersOrJoinQueue(ctx context.Context, player types.PlayerAddress) (*dao.GameMatch, error)
	CountLongWaittingPlayers(ctx context.Context, waitting time.Duration) (int, error)
	MatchPlayerWithBot(ctx context.Context, bot types.PlayerAddress, waitting time.Duration) (*dao.GameMatch, error)
	// MatchPlayers creates a pending PvP pair. lastGameID 0 is normal matchmaking; non-zero sets last_game_id (continue rematch).
	MatchPlayers(ctx context.Context, p1, p2 types.PlayerAddress, lastGameID int64) (*dao.GameMatch, error)
	RemoveQueue(ctx context.Context, player types.PlayerAddress) error
	PendingMatchID(ctx context.Context, player types.PlayerAddress) (int64, bool, error)
	MarkPlayersInGame(ctx context.Context, gameID int64, players ...types.PlayerAddress) error
	MarkPlayersOutOfGame(ctx context.Context, players ...types.PlayerAddress) error
	CancelPendingPair(context.Context, int64) error
}

var _ Store = (*GormStore)(nil)
