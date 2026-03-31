package db

import (
	"sync"
	"testing"

	dao "github.com/CryptoElementals/common/models"
	"github.com/CryptoElementals/common/rpc/proto"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func TestWithGameMutationTx_loadsPreloadedGraph(t *testing.T) {
	setupGamePersistMemDB(t)

	ga := seedSampleGameArgs(t)
	game := &dao.Game{
		GameArgs: ga,
		Type:     1,
		Status:   proto.GameStatus_GAME_INIT,
		Players: []*dao.GamePlayerInfo{
			{PlayerId: 401, TemporaryAddress: "0xm1"},
			{PlayerId: 402, TemporaryAddress: "0xm2"},
		},
	}
	attachSampleTurnForPersistTest(game)
	require.NoError(t, InsertNewGameGraphCommit(game))

	err := WithGameMutationTx(game.ID, func(tx *gorm.DB, g *dao.Game) error {
		require.Len(t, g.Players, 2)
		require.NotNil(t, g.GameArgs)
		require.Len(t, g.Turns, 1)
		st := proto.GameStatus_GAME_RUNNING
		return UpdateGameFieldsTx(tx, g.ID, GameFieldsUpdate{Status: &st})
	})
	require.NoError(t, err)

	loaded, err := LoadGameByGameID(game.ID)
	require.NoError(t, err)
	require.Equal(t, proto.GameStatus_GAME_RUNNING, loaded.Status)
}

func TestWithGameMutationTx_concurrentSameGame(t *testing.T) {
	setupGamePersistMemDB(t)

	ga := seedSampleGameArgs(t)
	game := &dao.Game{
		GameArgs: ga,
		Type:     1,
		Status:   proto.GameStatus_GAME_INIT,
		Players: []*dao.GamePlayerInfo{
			{PlayerId: 501, TemporaryAddress: "0xn1"},
			{PlayerId: 502, TemporaryAddress: "0xn2"},
		},
	}
	attachSampleTurnForPersistTest(game)
	require.NoError(t, InsertNewGameGraphCommit(game))
	gid := game.ID

	var wg sync.WaitGroup
	wg.Add(2)
	var err1, err2 error
	go func() {
		defer wg.Done()
		err1 = WithGameMutationTx(gid, func(tx *gorm.DB, g *dao.Game) error {
			st := proto.GameStatus_GAME_RUNNING
			return UpdateGameFieldsTx(tx, gid, GameFieldsUpdate{Status: &st})
		})
	}()
	go func() {
		defer wg.Done()
		err2 = WithGameMutationTx(gid, func(tx *gorm.DB, g *dao.Game) error {
			st := proto.GameStatus_GAME_END
			return UpdateGameFieldsTx(tx, gid, GameFieldsUpdate{Status: &st})
		})
	}()
	wg.Wait()
	require.NoError(t, err1)
	require.NoError(t, err2)

	loaded, err := LoadGameByGameID(gid)
	require.NoError(t, err)
	require.Contains(t, []proto.GameStatus{proto.GameStatus_GAME_RUNNING, proto.GameStatus_GAME_END}, loaded.Status)
}
