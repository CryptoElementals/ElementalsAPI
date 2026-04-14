package db

import (
	"testing"

	dao "github.com/CryptoElementals/common/models"
	"github.com/CryptoElementals/common/rpc/proto"
	"github.com/stretchr/testify/require"
	gorm_logger "gorm.io/gorm/logger"
)

func setupTestDBForUserStat(t *testing.T) {
	t.Helper()
	require.NoError(t, Init(&Config{Development: true}))
	Get().Logger = Get().Logger.LogMode(gorm_logger.Error)
	require.NoError(t, MigrateMemDb())
}

func sampleGameArgsStat() *dao.GameArgs {
	return &dao.GameArgs{
		MaxNormalRounds:                       3,
		MaxExtraRounds:                        0,
		MaxTurnsPerNormalRound:                3,
		MaxTurnsPerExtraRound:                 1,
		InitialHP:                             3000,
		MaxHP:                                 3000,
		BaseStake:                             1000,
		ConfirmationTimeout:                   60,
		CommitmentSubmissionTimeout:           60,
		CardSubmissionTimeout:                 60,
		GameContinueTimeout:                   120,
		ConfirmationTimeoutRedundancy:         10,
		CommitmentSubmissionTimeoutRedundancy: 10,
		CardSubmissionTimeoutRedundancy:       10,
		GameContinueTimeoutRedundancy:         10,
	}
}

// createPlayerRewardWithOutcome inserts one match-worth of rows: game → game_result → battle_reward_pvp → player_result_info → player_reward.
func createPlayerRewardWithOutcome(t *testing.T, playerID int64, temp string, st proto.PlayerGameResultStatus, tok, pt int32) *dao.PlayerReward {
	t.Helper()
	ga := sampleGameArgsStat()
	require.NoError(t, Get().Create(ga).Error)
	g := &dao.Game{GameArgsID: ga.ID, Type: 1, Status: proto.GameStatus_GAME_END}
	require.NoError(t, Get().Create(g).Error)
	gr := &dao.GameResult{GameID: g.ID, GameResultType: proto.GameResultType_GAME_NORMAL}
	require.NoError(t, Get().Create(gr).Error)
	br := &dao.BattleRewardPVP{GameID: g.ID}
	require.NoError(t, Get().Create(br).Error)
	pri := &dao.PlayerResultInfo{
		GameResultID:           gr.ID,
		PlayerId:               playerID,
		TemporaryAddress:       temp,
		PlayerGameResultStatus: st,
	}
	require.NoError(t, Get().Create(pri).Error)
	pr := &dao.PlayerReward{
		BattleRewardID: br.ID,
		PlayerId:       playerID,
		TokenChange:    tok,
		PointChange:    pt,
	}
	require.NoError(t, Get().Create(pr).Error)
	return pr
}

func TestUpdateUserStatByPlayerIds_EmptyInput(t *testing.T) {
	setupTestDBForUserStat(t)
	results, err := UpdateUserStatByPlayerIds([]int64{})
	require.NoError(t, err)
	require.Empty(t, results)
}

func TestUpdateUserStatByPlayerIds_NewPlayer(t *testing.T) {
	setupTestDBForUserStat(t)
	playerID := int64(2001)
	profile := &dao.UserProfile{
		PlayerID: playerID,
		Address:  "0x1111111111111111111111111111111111111111",
		Name:     "test_player_1",
	}
	require.NoError(t, Get().Create(profile).Error)

	reward := createPlayerRewardWithOutcome(t, playerID, "0xtemp111", proto.PlayerGameResultStatus_PLAYER_WIN, 100, 50)

	results, err := UpdateUserStatByPlayerIds([]int64{playerID})
	require.NoError(t, err)
	require.Len(t, results, 1)

	result := results[0]
	require.Equal(t, playerID, result.PlayerID)
	require.Equal(t, uint(1), result.TotalGameCount)
	require.Equal(t, uint(1), result.WinCount)
	require.Equal(t, uint(0), result.LoseCount)
	require.Equal(t, uint(0), result.TieCount)
	require.Equal(t, reward.ID, result.LastPlayerRewardID)

	var userStat dao.UserStat
	require.NoError(t, Get().Where("player_id = ?", playerID).First(&userStat).Error)
	require.Equal(t, uint(1), userStat.TotalGameCount)
	require.Equal(t, uint(1), userStat.WinCount)
}

func TestUpdateUserStatByPlayerIds_ExistingPlayerWithNewData(t *testing.T) {
	setupTestDBForUserStat(t)
	playerID := int64(2002)
	profile := &dao.UserProfile{
		PlayerID: playerID,
		Address:  "0x2222222222222222222222222222222222222222",
		Name:     "test_player_2",
	}
	require.NoError(t, Get().Create(profile).Error)

	existingStat := &dao.UserStat{
		PlayerID:           playerID,
		TotalGameCount:     5,
		WinCount:           3,
		LoseCount:          1,
		TieCount:           1,
		LastPlayerRewardID: 0,
	}
	require.NoError(t, Get().Create(existingStat).Error)

	newReward1 := createPlayerRewardWithOutcome(t, playerID, "0xtemp222", proto.PlayerGameResultStatus_PLAYER_LOSE, 200, 100)
	newReward2 := createPlayerRewardWithOutcome(t, playerID, "0xtemp222", proto.PlayerGameResultStatus_PLAYER_TIE, 150, 75)

	results, err := UpdateUserStatByPlayerIds([]int64{playerID})
	require.NoError(t, err)
	require.Len(t, results, 1)

	result := results[0]
	require.Equal(t, playerID, result.PlayerID)
	require.Equal(t, uint(7), result.TotalGameCount)
	require.Equal(t, uint(3), result.WinCount)
	require.Equal(t, uint(2), result.LoseCount)
	require.Equal(t, uint(2), result.TieCount)
	require.Equal(t, newReward2.ID, result.LastPlayerRewardID)
	_ = newReward1
}

func TestUpdateUserStatByPlayerIds_NoNewData(t *testing.T) {
	setupTestDBForUserStat(t)
	playerID := int64(2003)
	profile := &dao.UserProfile{
		PlayerID: playerID,
		Address:  "0x3333333333333333333333333333333333333333",
		Name:     "test_player_3",
	}
	require.NoError(t, Get().Create(profile).Error)

	existingStat := &dao.UserStat{
		PlayerID:           playerID,
		TotalGameCount:     10,
		WinCount:           6,
		LoseCount:          3,
		TieCount:           1,
		LastPlayerRewardID: 20,
	}
	require.NoError(t, Get().Create(existingStat).Error)

	results, err := UpdateUserStatByPlayerIds([]int64{playerID})
	require.NoError(t, err)
	require.Len(t, results, 1)

	result := results[0]
	require.Equal(t, playerID, result.PlayerID)
	require.Equal(t, uint(10), result.TotalGameCount)
	require.Equal(t, uint(6), result.WinCount)
	require.Equal(t, uint(3), result.LoseCount)
	require.Equal(t, uint(1), result.TieCount)
	require.Equal(t, uint(20), result.LastPlayerRewardID)
}

func TestUpdateUserStatByPlayerIds_MultiplePlayers(t *testing.T) {
	setupTestDBForUserStat(t)
	player1ID := int64(2004)
	profile1 := &dao.UserProfile{
		PlayerID: player1ID,
		Address:  "0x4444444444444444444444444444444444444444",
		Name:     "test_player_4",
	}
	require.NoError(t, Get().Create(profile1).Error)

	player2ID := int64(2005)
	profile2 := &dao.UserProfile{
		PlayerID: player2ID,
		Address:  "0x5555555555555555555555555555555555555555",
		Name:     "test_player_5",
	}
	require.NoError(t, Get().Create(profile2).Error)

	require.NoError(t, Get().Create(&dao.UserStat{
		PlayerID: player1ID, TotalGameCount: 2, WinCount: 1, LoseCount: 1, TieCount: 0, LastPlayerRewardID: 0,
	}).Error)
	require.NoError(t, Get().Create(&dao.UserStat{
		PlayerID: player2ID, TotalGameCount: 3, WinCount: 2, LoseCount: 1, TieCount: 0, LastPlayerRewardID: 0,
	}).Error)

	reward1 := createPlayerRewardWithOutcome(t, player1ID, "0xtemp444", proto.PlayerGameResultStatus_PLAYER_WIN, 0, 0)
	reward2 := createPlayerRewardWithOutcome(t, player2ID, "0xtemp555", proto.PlayerGameResultStatus_PLAYER_LOSE, 0, 0)

	results, err := UpdateUserStatByPlayerIds([]int64{player1ID, player2ID})
	require.NoError(t, err)
	require.Len(t, results, 2)
	_ = reward1
	_ = reward2

	var player1Result, player2Result *dao.UserStat
	for _, r := range results {
		switch r.PlayerID {
		case player1ID:
			player1Result = r
		case player2ID:
			player2Result = r
		}
	}
	require.NotNil(t, player1Result)
	require.Equal(t, uint(3), player1Result.TotalGameCount)
	require.Equal(t, uint(2), player1Result.WinCount)
	require.Equal(t, uint(1), player1Result.LoseCount)

	require.NotNil(t, player2Result)
	require.Equal(t, uint(4), player2Result.TotalGameCount)
	require.Equal(t, uint(2), player2Result.WinCount)
	require.Equal(t, uint(2), player2Result.LoseCount)
}

func TestUpdateUserStatByPlayerIds_DeduplicatePlayerIDs(t *testing.T) {
	setupTestDBForUserStat(t)
	playerID := int64(2006)
	profile := &dao.UserProfile{
		PlayerID: playerID,
		Address:  "0x6666666666666666666666666666666666666666",
		Name:     "test_player_6",
	}
	require.NoError(t, Get().Create(profile).Error)

	require.NoError(t, Get().Create(&dao.UserStat{
		PlayerID: playerID, TotalGameCount: 5, WinCount: 3, LoseCount: 2, TieCount: 0, LastPlayerRewardID: 0,
	}).Error)

	createPlayerRewardWithOutcome(t, playerID, "0xtemp666", proto.PlayerGameResultStatus_PLAYER_WIN, 0, 0)

	results, err := UpdateUserStatByPlayerIds([]int64{playerID, playerID, playerID})
	require.NoError(t, err)
	require.Len(t, results, 1)

	result := results[0]
	require.Equal(t, playerID, result.PlayerID)
	require.Equal(t, uint(6), result.TotalGameCount)
	require.Equal(t, uint(4), result.WinCount)
	require.Equal(t, uint(2), result.LoseCount)
}

func TestUpdateUserStatByPlayerIds_NonExistentPlayer(t *testing.T) {
	setupTestDBForUserStat(t)
	results, err := UpdateUserStatByPlayerIds([]int64{999999})
	require.NoError(t, err)
	require.Empty(t, results)
}

func TestUpdateUserStatByPlayerIds_MixedGameResults(t *testing.T) {
	setupTestDBForUserStat(t)
	playerID := int64(2007)
	profile := &dao.UserProfile{
		PlayerID: playerID,
		Address:  "0x7777777777777777777777777777777777777777",
		Name:     "test_player_7",
	}
	require.NoError(t, Get().Create(profile).Error)

	require.NoError(t, Get().Create(&dao.UserStat{
		PlayerID: playerID, TotalGameCount: 4, WinCount: 2, LoseCount: 1, TieCount: 1, LastPlayerRewardID: 0,
	}).Error)

	winReward := createPlayerRewardWithOutcome(t, playerID, "0xtemp777", proto.PlayerGameResultStatus_PLAYER_WIN, 0, 0)
	loseReward := createPlayerRewardWithOutcome(t, playerID, "0xtemp777", proto.PlayerGameResultStatus_PLAYER_LOSE, 0, 0)
	tieReward := createPlayerRewardWithOutcome(t, playerID, "0xtemp777", proto.PlayerGameResultStatus_PLAYER_TIE, 0, 0)

	results, err := UpdateUserStatByPlayerIds([]int64{playerID})
	require.NoError(t, err)
	require.Len(t, results, 1)

	result := results[0]
	require.Equal(t, playerID, result.PlayerID)
	require.Equal(t, uint(7), result.TotalGameCount)
	require.Equal(t, uint(3), result.WinCount)
	require.Equal(t, uint(2), result.LoseCount)
	require.Equal(t, uint(2), result.TieCount)
	require.Equal(t, tieReward.ID, result.LastPlayerRewardID)
	_ = winReward
	_ = loseReward
}

func TestUpdateUserStatByPlayerIds_ExistingPlayerWithMultipleNewRewards(t *testing.T) {
	setupTestDBForUserStat(t)
	playerID := int64(2008)
	profile := &dao.UserProfile{
		PlayerID: playerID,
		Address:  "0x8888888888888888888888888888888888888888",
		Name:     "test_player_8",
	}
	require.NoError(t, Get().Create(profile).Error)

	require.NoError(t, Get().Create(&dao.UserStat{
		PlayerID: playerID, TotalGameCount: 3, WinCount: 2, LoseCount: 1, TieCount: 0, LastPlayerRewardID: 0,
	}).Error)

	createPlayerRewardWithOutcome(t, playerID, "0xtemp888", proto.PlayerGameResultStatus_PLAYER_WIN, 0, 0)
	createPlayerRewardWithOutcome(t, playerID, "0xtemp888", proto.PlayerGameResultStatus_PLAYER_WIN, 0, 0)
	last := createPlayerRewardWithOutcome(t, playerID, "0xtemp888", proto.PlayerGameResultStatus_PLAYER_LOSE, 0, 0)

	results, err := UpdateUserStatByPlayerIds([]int64{playerID})
	require.NoError(t, err)
	require.Len(t, results, 1)

	result := results[0]
	require.Equal(t, playerID, result.PlayerID)
	require.Equal(t, uint(6), result.TotalGameCount)
	require.Equal(t, uint(4), result.WinCount)
	require.Equal(t, uint(2), result.LoseCount)
	require.Equal(t, uint(0), result.TieCount)
	require.Equal(t, last.ID, result.LastPlayerRewardID)
}
