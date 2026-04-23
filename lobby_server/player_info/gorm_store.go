package player_info

import (
	"context"
	"strings"
	"time"

	"github.com/CryptoElementals/common/db"
	"github.com/CryptoElementals/common/room_server/worker/types"
)

// GormStore implements Store using player_queue_entries / player_game_entries via the db package.
type GormStore struct{}

// NewGormStore returns a Store that uses db.Get() (initialize db before use).
func NewGormStore() *GormStore {
	return &GormStore{}
}

func normPlayer(p types.PlayerAddress) types.PlayerAddress {
	p.TemporaryAddress = strings.ToLower(strings.TrimSpace(p.TemporaryAddress))
	return p
}

func toLobbyRef(p types.PlayerAddress) db.LobbyPlayerRef {
	p = normPlayer(p)
	return db.LobbyPlayerRef{PlayerID: p.Id, TempAddress: p.TemporaryAddress}
}

func (s *GormStore) IsInQueue(ctx context.Context, player types.PlayerAddress) (bool, error) {
	p := normPlayer(player)
	return db.LobbyIsInQueue(ctx, p.Id, p.TemporaryAddress)
}

func (s *GormStore) GetGameIDByPlayer(ctx context.Context, player types.PlayerAddress) (bool, int64, error) {
	p := normPlayer(player)
	return db.LobbyGetGameIDByPlayer(ctx, p.Id, p.TemporaryAddress)
}

func (s *GormStore) QueueJoinedAtMs(ctx context.Context, player types.PlayerAddress) (ms int64, ok bool, err error) {
	p := normPlayer(player)
	return db.LobbyQueueJoinedAtMs(ctx, p.Id, p.TemporaryAddress)
}

func (s *GormStore) MatchPlayersOrJoinQueue(ctx context.Context, player types.PlayerAddress) (int64, error) {
	p := normPlayer(player)
	return db.LobbyMatchPlayersOrJoinQueue(ctx, p.Id, p.TemporaryAddress, uint(types.GameTypePVP))
}

func (s *GormStore) CountLongWaittingPlayers(ctx context.Context, waitting time.Duration) (int, error) {
	return db.LobbyCountLongWaitingQueuedPlayers(ctx, waitting)
}

func (s *GormStore) MatchPlayerWithBot(ctx context.Context, bot types.PlayerAddress, waitting time.Duration) (int64, error) {
	return db.LobbyMatchEarliestQueuedPlayerWithBot(ctx, toLobbyRef(bot), uint(types.GameTypePVP), waitting)
}

func (s *GormStore) MatchPlayers(ctx context.Context, p1, p2 types.PlayerAddress) (int64, error) {
	return db.LobbyMatchPair(ctx, toLobbyRef(p1), toLobbyRef(p2), uint(types.GameTypePVP))
}

func (s *GormStore) RemoveQueue(ctx context.Context, player types.PlayerAddress) error {
	p := normPlayer(player)
	return db.LobbyRemoveFromQueue(ctx, p.Id, p.TemporaryAddress)
}

func (s *GormStore) PendingMatchID(ctx context.Context, player types.PlayerAddress) (int64, bool, error) {
	p := normPlayer(player)
	return db.LobbyPendingMatchID(ctx, p.Id, p.TemporaryAddress)
}

func (s *GormStore) MarkPlayersInGame(ctx context.Context, gameID int64, players ...types.PlayerAddress) error {
	refs := make([]db.LobbyPlayerRef, 0, len(players))
	for _, pl := range players {
		refs = append(refs, toLobbyRef(pl))
	}
	return db.LobbyMarkPlayersInGame(ctx, gameID, refs)
}

func (s *GormStore) MarkPlayersOutOfGame(ctx context.Context, players ...types.PlayerAddress) error {
	refs := make([]db.LobbyPlayerRef, 0, len(players))
	for _, pl := range players {
		refs = append(refs, toLobbyRef(pl))
	}
	return db.LobbyMarkPlayersOutOfGame(ctx, refs)
}

func (s *GormStore) CancelPendingPair(ctx context.Context, matchID int64) error {
	return db.LobbyCancelPendingMatch(ctx, matchID)
}
