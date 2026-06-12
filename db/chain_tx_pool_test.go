package db

import (
	"testing"
	"time"

	dao "github.com/CryptoElementals/common/models"
	"github.com/stretchr/testify/require"
)

func TestChainTxPoolInsertDuplicate(t *testing.T) {
	require.NoError(t, initMemDbSqlite())
	require.NoError(t, MigrateMemDb())

	row := &dao.ChainTxPoolItem{
		ChainID:             7,
		Kind:                dao.ChainTxPoolKindCommitment,
		GameID:              100,
		PlayerTemporaryAddr: "0xaaa",
		RoundNumber:         1,
		TurnNumber:          2,
		Payload:             []byte("a"),
	}
	require.NoError(t, InsertChainTxPoolItem(row))
	again := &dao.ChainTxPoolItem{
		ChainID:             7,
		Kind:                dao.ChainTxPoolKindCommitment,
		GameID:              100,
		PlayerTemporaryAddr: "0xaaa",
		RoundNumber:         1,
		TurnNumber:          2,
		Payload:             []byte("b"),
	}
	require.ErrorIs(t, InsertChainTxPoolItem(again), ErrChainTxPoolDuplicate)
}

func TestChainTxPoolCreateRoomDuplicate(t *testing.T) {
	require.NoError(t, initMemDbSqlite())
	require.NoError(t, MigrateMemDb())

	row1 := &dao.ChainTxPoolItem{
		ChainID:             7,
		Kind:                dao.ChainTxPoolKindCreateRoom,
		GameID:              200,
		PlayerTemporaryAddr: "",
		RoundNumber:         0,
		TurnNumber:          0,
		Payload:             []byte("first"),
	}
	require.NoError(t, InsertChainTxPoolItem(row1))

	row2 := &dao.ChainTxPoolItem{
		ChainID:             7,
		Kind:                dao.ChainTxPoolKindCreateRoom,
		GameID:              200,
		PlayerTemporaryAddr: "",
		RoundNumber:         0,
		TurnNumber:          0,
		Payload:             []byte("second"),
	}
	require.ErrorIs(t, InsertChainTxPoolItem(row2), ErrChainTxPoolDuplicate)

	pending, err := ListChainTxPoolPendingForChain(7)
	require.NoError(t, err)
	require.Len(t, pending, 1)
	require.Equal(t, []byte("first"), pending[0].Payload)
}

func TestChainTxPoolListAndDelete(t *testing.T) {
	require.NoError(t, initMemDbSqlite())
	require.NoError(t, MigrateMemDb())

	// create_room game 1
	require.NoError(t, InsertChainTxPoolItem(&dao.ChainTxPoolItem{
		ChainID: 1, Kind: dao.ChainTxPoolKindCreateRoom, GameID: 1,
		Payload: []byte("cr1"),
	}))
	// set_turn
	require.NoError(t, InsertChainTxPoolItem(&dao.ChainTxPoolItem{
		ChainID: 1, Kind: dao.ChainTxPoolKindSetTurnReady, GameID: 1,
		RoundNumber: 1, TurnNumber: 0, Payload: []byte("st"),
	}))
	// commitment
	require.NoError(t, InsertChainTxPoolItem(&dao.ChainTxPoolItem{
		ChainID: 1, Kind: dao.ChainTxPoolKindCommitment, GameID: 1,
		PlayerTemporaryAddr: "0xp", RoundNumber: 0, TurnNumber: 1, Payload: []byte("c1"),
	}))
	// card
	require.NoError(t, InsertChainTxPoolItem(&dao.ChainTxPoolItem{
		ChainID: 1, Kind: dao.ChainTxPoolKindCard, GameID: 1,
		PlayerTemporaryAddr: "0xp", RoundNumber: 0, TurnNumber: 1, Payload: []byte("k1"),
	}))

	pending, err := ListChainTxPoolPendingForChain(1)
	require.NoError(t, err)
	require.Len(t, pending, 4)
	kinds := make(map[uint8]struct{})
	for _, p := range pending {
		kinds[p.Kind] = struct{}{}
	}
	require.Len(t, kinds, 4)
	require.Contains(t, kinds, dao.ChainTxPoolKindCreateRoom)
	require.Contains(t, kinds, dao.ChainTxPoolKindSetTurnReady)
	require.Contains(t, kinds, dao.ChainTxPoolKindCommitment)
	require.Contains(t, kinds, dao.ChainTxPoolKindCard)

	ids := []uint{pending[0].ID, pending[1].ID}
	require.NoError(t, DeleteChainTxPoolItemsByIDs(ids))

	pending2, err := ListChainTxPoolPendingForChain(1)
	require.NoError(t, err)
	require.Len(t, pending2, 2)
}

func TestChainTxPoolListByChainMatchesPerChainList(t *testing.T) {
	require.NoError(t, initMemDbSqlite())
	require.NoError(t, MigrateMemDb())

	require.NoError(t, InsertChainTxPoolItem(&dao.ChainTxPoolItem{
		ChainID: 1, Kind: dao.ChainTxPoolKindCreateRoom, GameID: 10, Payload: []byte("a"),
	}))
	require.NoError(t, InsertChainTxPoolItem(&dao.ChainTxPoolItem{
		ChainID: 2, Kind: dao.ChainTxPoolKindCreateRoom, GameID: 20, Payload: []byte("b"),
	}))

	by, err := ListChainTxPoolPendingByChain()
	require.NoError(t, err)
	p1, err := ListChainTxPoolPendingForChain(1)
	require.NoError(t, err)
	p2, err := ListChainTxPoolPendingForChain(2)
	require.NoError(t, err)
	require.Equal(t, p1, by[1])
	require.Equal(t, p2, by[2])
}

func TestChainTxPoolDeleteForGame(t *testing.T) {
	require.NoError(t, initMemDbSqlite())
	require.NoError(t, MigrateMemDb())

	require.NoError(t, InsertChainTxPoolItem(&dao.ChainTxPoolItem{
		ChainID: 1, Kind: dao.ChainTxPoolKindCommitment, GameID: 99,
		PlayerTemporaryAddr: "0xa", RoundNumber: 0, TurnNumber: 1, Payload: []byte("c"),
	}))

	require.NoError(t, DeleteChainTxPoolItemsForGame(99))
	var n int64
	require.NoError(t, Get().Model(&dao.ChainTxPoolItem{}).Where("game_id = ?", 99).Count(&n).Error)
	require.EqualValues(t, 0, n)
}

func TestClaimChainTxPoolBatchForChainDrainsByLimit(t *testing.T) {
	require.NoError(t, initMemDbSqlite())
	require.NoError(t, MigrateMemDb())

	for i := 0; i < 3; i++ {
		require.NoError(t, InsertChainTxPoolItem(&dao.ChainTxPoolItem{
			ChainID:             1,
			Kind:                dao.ChainTxPoolKindCommitment,
			GameID:              int64(100 + i),
			PlayerTemporaryAddr: "0xa",
			RoundNumber:         1,
			TurnNumber:          uint32(i),
			Payload:             []byte{byte(i + 1)},
		}))
	}

	claimTimeout := time.Second
	b1, err := ClaimChainTxPoolBatchForChain(1, 2, claimTimeout)
	require.NoError(t, err)
	require.Len(t, b1, 2)

	var n int64
	require.NoError(t, Get().Model(&dao.ChainTxPoolItem{}).Where("chain_id = ?", 1).Count(&n).Error)
	require.EqualValues(t, 3, n)

	require.NoError(t, DeleteChainTxPoolItemsByIDs([]uint{b1[0].ID, b1[1].ID}))

	b2, err := ClaimChainTxPoolBatchForChain(1, 2, claimTimeout)
	require.NoError(t, err)
	require.Len(t, b2, 1)

	require.NoError(t, DeleteChainTxPoolItemsByIDs([]uint{b2[0].ID}))

	require.NoError(t, Get().Model(&dao.ChainTxPoolItem{}).Where("chain_id = ?", 1).Count(&n).Error)
	require.EqualValues(t, 0, n)

	b3, err := ClaimChainTxPoolBatchForChain(1, 2, claimTimeout)
	require.NoError(t, err)
	require.Len(t, b3, 0)
}

func TestClaimChainTxPoolBatchForChainStaleReclaim(t *testing.T) {
	require.NoError(t, initMemDbSqlite())
	require.NoError(t, MigrateMemDb())

	require.NoError(t, InsertChainTxPoolItem(&dao.ChainTxPoolItem{
		ChainID:             1,
		Kind:                dao.ChainTxPoolKindCommitment,
		GameID:              100,
		PlayerTemporaryAddr: "0xa",
		RoundNumber:         1,
		TurnNumber:          1,
		Payload:             []byte{1},
	}))

	claimTimeout := 50 * time.Millisecond
	b1, err := ClaimChainTxPoolBatchForChain(1, 10, claimTimeout)
	require.NoError(t, err)
	require.Len(t, b1, 1)

	b2, err := ClaimChainTxPoolBatchForChain(1, 10, claimTimeout)
	require.NoError(t, err)
	require.Len(t, b2, 0)

	time.Sleep(claimTimeout + 30*time.Millisecond)
	require.NoError(t, ReleaseStaleChainTxPoolClaims(time.Now().Add(-claimTimeout)))

	b3, err := ClaimChainTxPoolBatchForChain(1, 10, claimTimeout)
	require.NoError(t, err)
	require.Len(t, b3, 1)
	require.Equal(t, b1[0].ID, b3[0].ID)
}
