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
	// 初始化内存数据库
	err := Init(&Config{Development: true})
	require.NoError(t, err)

	// 关闭 SQL 日志输出（只显示错误）
	Get().Logger = Get().Logger.LogMode(gorm_logger.Error)

	// 迁移基础表
	migrates := []any{
		&dao.UserProfile{},
		&dao.UserStat{},
		&dao.PlayerReward{},
	}
	err = Get().AutoMigrate(migrates...)
	require.NoError(t, err)
}

func TestUpdateUserStatByPlayerIds_EmptyInput(t *testing.T) {
	setupTestDBForUserStat(t)

	// 测试空输入
	results, err := UpdateUserStatByPlayerIds([]int64{})
	require.NoError(t, err)
	require.Empty(t, results)
}

func TestUpdateUserStatByPlayerIds_NewPlayer(t *testing.T) {
	setupTestDBForUserStat(t)

	// 创建测试玩家
	playerID := int64(2001)
	profile := &dao.UserProfile{
		PlayerID: playerID,
		Address:  "0x1111111111111111111111111111111111111111",
		Name:     "test_player_1",
	}
	require.NoError(t, Get().Create(profile).Error)

	// 创建 PlayerReward 记录
	reward := &dao.PlayerReward{
		PlayerId:               playerID,
		TemporaryAddress:       "0xtemp111",
		TokenChange:            100,
		PointChange:            50,
		PlayerGameResultStatus: proto.PlayerGameResultStatus_PLAYER_WIN,
		IsOffline:              false,
		Surrendered:            false,
	}
	require.NoError(t, Get().Create(reward).Error)

	// 执行更新
	results, err := UpdateUserStatByPlayerIds([]int64{playerID})
	require.NoError(t, err)
	require.Len(t, results, 1)

	// 验证结果
	result := results[0]
	require.Equal(t, playerID, result.PlayerID)
	require.Equal(t, uint(1), result.TotalGameCount)
	require.Equal(t, uint(1), result.WinCount)
	require.Equal(t, uint(0), result.LoseCount)
	require.Equal(t, uint(0), result.TieCount)
	require.Equal(t, reward.ID, result.LastPlayerRewardID)

	// 验证数据库中的记录
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

	// 创建已有的用户统计记录
	existingStat := &dao.UserStat{
		PlayerID:           playerID,
		TotalGameCount:     5,
		WinCount:           3,
		LoseCount:          1,
		TieCount:           1,
		LastPlayerRewardID: 10, // 已处理到 reward 10
	}
	require.NoError(t, Get().Create(existingStat).Error)

	// 创建新的 PlayerReward 记录（id > 10）
	// 先创建一些 dummy rewards 确保 id > 10
	for i := 0; i < 12; i++ {
		dummyReward := &dao.PlayerReward{
			PlayerId:               int64(9999), // 其他玩家
			TemporaryAddress:       "0xdummy",
			PlayerGameResultStatus: proto.PlayerGameResultStatus_PLAYER_WIN,
		}
		Get().Create(dummyReward)
	}

	// 创建新的奖励记录
	newReward1 := &dao.PlayerReward{
		PlayerId:               playerID,
		TemporaryAddress:       "0xtemp222",
		TokenChange:            200,
		PointChange:            100,
		PlayerGameResultStatus: proto.PlayerGameResultStatus_PLAYER_LOSE,
	}
	require.NoError(t, Get().Create(newReward1).Error)

	newReward2 := &dao.PlayerReward{
		PlayerId:               playerID,
		TemporaryAddress:       "0xtemp222",
		TokenChange:            150,
		PointChange:            75,
		PlayerGameResultStatus: proto.PlayerGameResultStatus_PLAYER_TIE,
	}
	require.NoError(t, Get().Create(newReward2).Error)

	// 执行更新
	results, err := UpdateUserStatByPlayerIds([]int64{playerID})
	require.NoError(t, err)
	require.Len(t, results, 1)

	// 验证结果：应该累加数据
	result := results[0]
	require.Equal(t, playerID, result.PlayerID)
	require.Equal(t, uint(7), result.TotalGameCount)           // 5 + 2 = 7
	require.Equal(t, uint(3), result.WinCount)                 // 不变
	require.Equal(t, uint(2), result.LoseCount)                // 1 + 1 = 2
	require.Equal(t, uint(2), result.TieCount)                 // 1 + 1 = 2
	require.Equal(t, newReward2.ID, result.LastPlayerRewardID) // 应该是最大的 reward ID
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

	// 创建已有的用户统计记录
	existingStat := &dao.UserStat{
		PlayerID:           playerID,
		TotalGameCount:     10,
		WinCount:           6,
		LoseCount:          3,
		TieCount:           1,
		LastPlayerRewardID: 20,
	}
	require.NoError(t, Get().Create(existingStat).Error)

	// 不创建新的 PlayerReward 记录

	// 执行更新
	results, err := UpdateUserStatByPlayerIds([]int64{playerID})
	require.NoError(t, err)
	require.Len(t, results, 1)

	// 验证结果：应该保持不变
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

	// 创建两个玩家
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

	// 为玩家1创建已有的统计记录（record found 场景）
	existingStat1 := &dao.UserStat{
		PlayerID:           player1ID,
		TotalGameCount:     2,
		WinCount:           1,
		LoseCount:          1,
		TieCount:           0,
		LastPlayerRewardID: 5,
	}
	require.NoError(t, Get().Create(existingStat1).Error)

	// 为玩家2创建已有的统计记录（record found 场景）
	existingStat2 := &dao.UserStat{
		PlayerID:           player2ID,
		TotalGameCount:     3,
		WinCount:           2,
		LoseCount:          1,
		TieCount:           0,
		LastPlayerRewardID: 8,
	}
	require.NoError(t, Get().Create(existingStat2).Error)

	// 先创建一些 dummy rewards 确保 id > 8
	for i := 0; i < 10; i++ {
		dummyReward := &dao.PlayerReward{
			PlayerId:               int64(9999),
			TemporaryAddress:       "0xdummy",
			PlayerGameResultStatus: proto.PlayerGameResultStatus_PLAYER_WIN,
		}
		Get().Create(dummyReward)
	}

	// 为玩家1创建新的奖励记录（id > 5）
	reward1 := &dao.PlayerReward{
		PlayerId:               player1ID,
		TemporaryAddress:       "0xtemp444",
		PlayerGameResultStatus: proto.PlayerGameResultStatus_PLAYER_WIN,
	}
	require.NoError(t, Get().Create(reward1).Error)

	// 为玩家2创建新的奖励记录（id > 8）
	reward2 := &dao.PlayerReward{
		PlayerId:               player2ID,
		TemporaryAddress:       "0xtemp555",
		PlayerGameResultStatus: proto.PlayerGameResultStatus_PLAYER_LOSE,
	}
	require.NoError(t, Get().Create(reward2).Error)

	// 执行更新
	results, err := UpdateUserStatByPlayerIds([]int64{player1ID, player2ID})
	require.NoError(t, err)
	require.Len(t, results, 2)

	// 验证玩家1的结果（应该累加）
	var player1Result *dao.UserStat
	for _, r := range results {
		if r.PlayerID == player1ID {
			player1Result = r
			break
		}
	}
	require.NotNil(t, player1Result)
	require.Equal(t, uint(3), player1Result.TotalGameCount) // 2 + 1 = 3
	require.Equal(t, uint(2), player1Result.WinCount)       // 1 + 1 = 2
	require.Equal(t, uint(1), player1Result.LoseCount)      // 不变

	// 验证玩家2的结果（应该累加）
	var player2Result *dao.UserStat
	for _, r := range results {
		if r.PlayerID == player2ID {
			player2Result = r
			break
		}
	}
	require.NotNil(t, player2Result)
	require.Equal(t, uint(4), player2Result.TotalGameCount) // 3 + 1 = 4
	require.Equal(t, uint(2), player2Result.WinCount)       // 不变
	require.Equal(t, uint(2), player2Result.LoseCount)      // 1 + 1 = 2
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

	// 创建已有的统计记录（record found 场景）
	existingStat := &dao.UserStat{
		PlayerID:           playerID,
		TotalGameCount:     5,
		WinCount:           3,
		LoseCount:          2,
		TieCount:           0,
		LastPlayerRewardID: 10,
	}
	require.NoError(t, Get().Create(existingStat).Error)

	// 先创建一些 dummy rewards 确保 id > 10
	for i := 0; i < 12; i++ {
		dummyReward := &dao.PlayerReward{
			PlayerId:               int64(9999),
			TemporaryAddress:       "0xdummy",
			PlayerGameResultStatus: proto.PlayerGameResultStatus_PLAYER_WIN,
		}
		Get().Create(dummyReward)
	}

	// 创建新的奖励记录（id > 10）
	reward := &dao.PlayerReward{
		PlayerId:               playerID,
		TemporaryAddress:       "0xtemp666",
		PlayerGameResultStatus: proto.PlayerGameResultStatus_PLAYER_WIN,
	}
	require.NoError(t, Get().Create(reward).Error)

	// 执行更新，传入重复的玩家ID
	results, err := UpdateUserStatByPlayerIds([]int64{playerID, playerID, playerID})
	require.NoError(t, err)
	require.Len(t, results, 1) // 应该只返回一条记录

	// 验证结果（应该累加）
	result := results[0]
	require.Equal(t, playerID, result.PlayerID)
	require.Equal(t, uint(6), result.TotalGameCount) // 5 + 1 = 6
	require.Equal(t, uint(4), result.WinCount)       // 3 + 1 = 4
	require.Equal(t, uint(2), result.LoseCount)      // 不变
}

func TestUpdateUserStatByPlayerIds_NonExistentPlayer(t *testing.T) {
	setupTestDBForUserStat(t)

	// 不创建玩家，直接使用不存在的玩家ID
	nonExistentPlayerID := int64(9999)

	// 执行更新
	results, err := UpdateUserStatByPlayerIds([]int64{nonExistentPlayerID})
	require.NoError(t, err)
	require.Empty(t, results) // 应该返回空结果，跳过不存在的玩家
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

	// 创建已有的统计记录（record found 场景）
	existingStat := &dao.UserStat{
		PlayerID:           playerID,
		TotalGameCount:     4,
		WinCount:           2,
		LoseCount:          1,
		TieCount:           1,
		LastPlayerRewardID: 15,
	}
	require.NoError(t, Get().Create(existingStat).Error)

	// 先创建一些 dummy rewards 确保 id > 15
	for i := 0; i < 17; i++ {
		dummyReward := &dao.PlayerReward{
			PlayerId:               int64(9999),
			TemporaryAddress:       "0xdummy",
			PlayerGameResultStatus: proto.PlayerGameResultStatus_PLAYER_WIN,
		}
		Get().Create(dummyReward)
	}

	// 创建多种游戏结果的奖励记录（id > 15）
	winReward := &dao.PlayerReward{
		PlayerId:               playerID,
		TemporaryAddress:       "0xtemp777",
		PlayerGameResultStatus: proto.PlayerGameResultStatus_PLAYER_WIN,
	}
	require.NoError(t, Get().Create(winReward).Error)

	loseReward := &dao.PlayerReward{
		PlayerId:               playerID,
		TemporaryAddress:       "0xtemp777",
		PlayerGameResultStatus: proto.PlayerGameResultStatus_PLAYER_LOSE,
	}
	require.NoError(t, Get().Create(loseReward).Error)

	tieReward := &dao.PlayerReward{
		PlayerId:               playerID,
		TemporaryAddress:       "0xtemp777",
		PlayerGameResultStatus: proto.PlayerGameResultStatus_PLAYER_TIE,
	}
	require.NoError(t, Get().Create(tieReward).Error)

	// 执行更新
	results, err := UpdateUserStatByPlayerIds([]int64{playerID})
	require.NoError(t, err)
	require.Len(t, results, 1)

	// 验证结果（应该累加）
	result := results[0]
	require.Equal(t, playerID, result.PlayerID)
	require.Equal(t, uint(7), result.TotalGameCount) // 4 + 3 = 7
	require.Equal(t, uint(3), result.WinCount)       // 2 + 1 = 3
	require.Equal(t, uint(2), result.LoseCount)      // 1 + 1 = 2
	require.Equal(t, uint(2), result.TieCount)       // 1 + 1 = 2
	require.Equal(t, tieReward.ID, result.LastPlayerRewardID)
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

	// 创建已有的用户统计记录
	existingStat := &dao.UserStat{
		PlayerID:           playerID,
		TotalGameCount:     3,
		WinCount:           2,
		LoseCount:          1,
		TieCount:           0,
		LastPlayerRewardID: 5,
	}
	require.NoError(t, Get().Create(existingStat).Error)

	// 创建多个新的奖励记录
	rewards := []*dao.PlayerReward{
		{
			PlayerId:               playerID,
			TemporaryAddress:       "0xtemp888",
			PlayerGameResultStatus: proto.PlayerGameResultStatus_PLAYER_WIN,
		},
		{
			PlayerId:               playerID,
			TemporaryAddress:       "0xtemp888",
			PlayerGameResultStatus: proto.PlayerGameResultStatus_PLAYER_WIN,
		},
		{
			PlayerId:               playerID,
			TemporaryAddress:       "0xtemp888",
			PlayerGameResultStatus: proto.PlayerGameResultStatus_PLAYER_LOSE,
		},
	}

	// 先创建一些 dummy rewards 确保 id > 5
	for i := 0; i < 7; i++ {
		dummyReward := &dao.PlayerReward{
			PlayerId:               int64(9999),
			TemporaryAddress:       "0xdummy",
			PlayerGameResultStatus: proto.PlayerGameResultStatus_PLAYER_WIN,
		}
		Get().Create(dummyReward)
	}

	for _, reward := range rewards {
		require.NoError(t, Get().Create(reward).Error)
	}

	// 执行更新
	results, err := UpdateUserStatByPlayerIds([]int64{playerID})
	require.NoError(t, err)
	require.Len(t, results, 1)

	// 验证结果
	result := results[0]
	require.Equal(t, playerID, result.PlayerID)
	require.Equal(t, uint(6), result.TotalGameCount)       // 3 + 3 = 6
	require.Equal(t, uint(4), result.WinCount)             // 2 + 2 = 4
	require.Equal(t, uint(2), result.LoseCount)            // 1 + 1 = 2
	require.Equal(t, uint(0), result.TieCount)             // 不变
	require.Greater(t, result.LastPlayerRewardID, uint(5)) // 应该是最大的 reward ID
}
