package db

import (
	"testing"

	dao "github.com/CryptoElementals/common/models"
	"github.com/CryptoElementals/common/rpc/proto"
	"github.com/stretchr/testify/require"
)

func setupGamePersistMemDB(t *testing.T) {
	t.Helper()
	require.NoError(t, Init(&Config{Development: true}))
	require.NoError(t, MigrateMemDb())
}

func sampleGameArgs() *dao.GameArgs {
	return &dao.GameArgs{
		MaxRounds:                             3,
		InitialHP:                             3000,
		InitialMultiplier:                     1,
		ConfirmationTimeout:                   60,
		CommitmentSubmissionTimeout:           60,
		CardSubmissionTimeout:                 60,
		GameContinueTimeout:                   120,
		ConfirmationTimeoutRedundancy:         10,
		CommitmentSubmissionTimeoutRedundancy: 10,
		CardSubmissionTimeoutRedundancy:       10,
		GameContinueTimeoutRedundancy:         10,
		MaxTurnsPerRound:                      3,
	}
}

// seedSampleGameArgs inserts a template game_args row (mirrors production: id exists before InsertNewGameGraph).
func seedSampleGameArgs(t *testing.T) *dao.GameArgs {
	t.Helper()
	ga := sampleGameArgs()
	require.NoError(t, Get().Create(ga).Error)
	require.NotZero(t, ga.ID)
	return ga
}

func TestGamePersist_InsertAndGranularUpdates(t *testing.T) {
	setupGamePersistMemDB(t)

	ga := seedSampleGameArgs(t)
	game := &dao.Game{
		GameArgs: ga,
		Type:     1,
		Status:   proto.GameStatus_GAME_INIT,
		Players: []*dao.GamePlayerInfo{
			{PlayerId: 101, TemporaryAddress: "0xaaa"},
			{PlayerId: 102, TemporaryAddress: "0xbbb"},
		},
	}
	attachSampleTurnForPersistTest(game)

	require.NoError(t, InsertNewGameGraphCommit(game))
	require.NotZero(t, game.ID)
	require.NotZero(t, ga.ID)

	loaded, err := LoadGameByGameID(game.ID)
	require.NoError(t, err)
	require.Len(t, loaded.Players, 2)
	require.Len(t, loaded.Turns, 1)
	require.Len(t, loaded.Turns[0].PlayerTurnInfos, 2)

	st := proto.GameStatus_GAME_RUNNING
	require.NoError(t, UpdateGameFieldsCommit(game.ID, GameFieldsUpdate{Status: &st}))

	turn := loaded.Turns[0]
	turn.TurnStartAt = 1700000000
	turn.TurnStatus = uint32(proto.TurnStatus_TURN_WAITTING_COMMITMENTS)
	require.NoError(t, SaveTurnCommit(turn))

	pti := loaded.Turns[0].PlayerTurnInfos[0]
	pti.PlayerStatus = proto.PlayerTurnStatus_PLAYER_TURN_COMMITMENT_SUBMITTED
	if pti.TurnSubmittedCard == nil {
		pti.TurnSubmittedCard = &dao.TurnSubmittedCard{}
	}
	pti.TurnSubmittedCard.CommitmentHash = []byte{1, 2, 3}
	pti.TurnSubmittedCard.CardEffects = []*dao.CardEffect{
		{Type: proto.BattleEffectType_ATTACK, Value: 10, Description: "test"},
	}
	require.NoError(t, SavePlayerTurnInfoCommit(pti))

	loaded2, err := LoadGameByGameID(game.ID)
	require.NoError(t, err)
	require.Equal(t, proto.GameStatus_GAME_RUNNING, loaded2.Status)
	require.Equal(t, int64(1700000000), loaded2.Turns[0].TurnStartAt)
	var found *dao.PlayerTurnInfo
	for _, p := range loaded2.Turns[0].PlayerTurnInfos {
		if p.PlayerID == 101 {
			found = p
			break
		}
	}
	require.NotNil(t, found)
	require.Len(t, found.TurnSubmittedCard.CardEffects, 1)

	gr := &dao.GameResult{
		Multiplier:             2,
		WinnerPlayerId:         101,
		WinnerTemporaryAddress: "0xaaa",
		GameResultType:         proto.GameResultType_GAME_KO,
		BattleReward: &dao.BattleReward{
			SystemFee: 1,
			PlayerRewards: []*dao.PlayerReward{
				{PlayerId: 101, TemporaryAddress: "0xaaa", TokenChange: 1},
				{PlayerId: 102, TemporaryAddress: "0xbbb", TokenChange: -1},
			},
		},
	}
	require.NoError(t, SaveGameResultTreeCommit(game.ID, gr))

	loaded3, err := LoadGameByGameID(game.ID)
	require.NoError(t, err)
	require.NotNil(t, loaded3.GameResult)
	require.NotNil(t, loaded3.GameResult.BattleReward)
	require.Len(t, loaded3.GameResult.BattleReward.PlayerRewards, 2)
}

func attachSampleTurnForPersistTest(g *dao.Game) {
	turn := &dao.Turn{
		RoundNumber: 1,
		TurnNumber:  1,
		TurnStatus:  uint32(proto.TurnStatus_TURN_WAITTING_BATTLE_CONFIRMATION),
		PlayerTurnInfos: []*dao.PlayerTurnInfo{
			{
				PlayerID:         g.Players[0].PlayerId,
				TemporaryAddress: g.Players[0].TemporaryAddress,
				TurnSubmittedCard: &dao.TurnSubmittedCard{
					HealthBefore:     3000,
					MultiplierBefore: 1,
				},
			},
			{
				PlayerID:         g.Players[1].PlayerId,
				TemporaryAddress: g.Players[1].TemporaryAddress,
				TurnSubmittedCard: &dao.TurnSubmittedCard{
					HealthBefore:     3000,
					MultiplierBefore: 1,
				},
			},
		},
	}
	g.Turns = []*dao.Turn{turn}
}

func TestPhase3_SaveFullGameGraphPreservesSnapshot(t *testing.T) {
	setupGamePersistMemDB(t)
	ga := seedSampleGameArgs(t)
	game := &dao.Game{
		GameArgs: ga,
		Type:     1,
		Status:   proto.GameStatus_GAME_INIT,
		Players: []*dao.GamePlayerInfo{
			{PlayerId: 201, TemporaryAddress: "0xc01"},
			{PlayerId: 202, TemporaryAddress: "0xc02"},
		},
	}
	attachSampleTurnForPersistTest(game)
	require.NoError(t, InsertNewGameGraphCommit(game))

	loaded, err := LoadGameByGameID(game.ID)
	require.NoError(t, err)
	before := CaptureGamePersistenceSnapshot(loaded)
	require.NoError(t, SaveFullGameGraph(loaded))

	again, err := LoadGameByGameID(game.ID)
	require.NoError(t, err)
	after := CaptureGamePersistenceSnapshot(again)
	require.Equal(t, before, after)
}
