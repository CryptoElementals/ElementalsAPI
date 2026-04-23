package player_info

import (
	"context"
	"strings"

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

func (s *GormStore) QueueJoinedAtMs(ctx context.Context, player types.PlayerAddress) (ms int64, ok bool, err error) {
	p := normPlayer(player)
	return db.LobbyQueueJoinedAtMs(ctx, p.Id, p.TemporaryAddress)
}

func (s *GormStore) ListQueuedPlayers(ctx context.Context) ([]types.PlayerAddress, error) {
	rows, err := db.LobbyListQueuedPlayers(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]types.PlayerAddress, 0, len(rows))
	for _, r := range rows {
		out = append(out, *types.NewPlayerAddress(r.PlayerID, r.TempAddress))
	}
	return out, nil
}

func (s *GormStore) AddQueue(ctx context.Context, player types.PlayerAddress, nowMs int64) (bool, error) {
	p := normPlayer(player)
	return db.LobbyAddQueue(ctx, p.Id, p.TemporaryAddress, nowMs)
}

func (s *GormStore) RemoveQueue(ctx context.Context, player types.PlayerAddress) error {
	p := normPlayer(player)
	return db.LobbyRemoveFromQueue(ctx, p.Id, p.TemporaryAddress)
}

func (s *GormStore) SetPendingPair(ctx context.Context, matchID int64, p1, p2 types.PlayerAddress) (bool, error) {
	return db.LobbySetPendingPair(ctx, matchID, toLobbyRef(p1), toLobbyRef(p2))
}

func (s *GormStore) CancelPendingPair(ctx context.Context, matchID int64, p1, p2 types.PlayerAddress) (bool, error) {
	return db.LobbyCancelPendingPair(ctx, matchID, toLobbyRef(p1), toLobbyRef(p2))
}

func (s *GormStore) FinalizeConfirmedPair(ctx context.Context, matchID int64, p1, p2 types.PlayerAddress) (bool, error) {
	return db.LobbyFinalizeConfirmedPair(ctx, matchID, toLobbyRef(p1), toLobbyRef(p2))
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

func (s *GormStore) IsInGame(ctx context.Context, player types.PlayerAddress) (bool, error) {
	p := normPlayer(player)
	return db.LobbyIsInGame(ctx, p.Id, p.TemporaryAddress)
}

func (s *GormStore) PendingMatchID(ctx context.Context, player types.PlayerAddress) (int64, bool, error) {
	p := normPlayer(player)
	return db.LobbyPendingMatchID(ctx, p.Id, p.TemporaryAddress)
}

func (s *GormStore) JoinQueueOrGetMatchCandidate(ctx context.Context, player types.PlayerAddress, nowMs int64) (*types.PlayerAddress, bool, error) {
	p := normPlayer(player)
	cand, queued, err := db.LobbyJoinQueueOrGetMatchCandidate(ctx, p.Id, p.TemporaryAddress, nowMs)
	if err != nil || cand == nil {
		return nil, queued, err
	}
	return types.NewPlayerAddress(cand.PlayerID, cand.TempAddress), queued, nil
}

func (s *GormStore) FirstWaitingPlayerBefore(ctx context.Context, cutoffMs int64) (*types.PlayerAddress, error) {
	ref, err := db.LobbyFirstWaitingPlayerBefore(ctx, cutoffMs)
	if err != nil || ref == nil {
		return nil, err
	}
	return types.NewPlayerAddress(ref.PlayerID, ref.TempAddress), nil
}
