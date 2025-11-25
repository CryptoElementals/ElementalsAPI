package db

import (
	"encoding/json"
	"testing"

	dao "github.com/CryptoElementals/common/models"
	"github.com/CryptoElementals/common/rpc/proto"
	"github.com/stretchr/testify/require"
	gorm_logger "gorm.io/gorm/logger"
)

func setupTestDB(t *testing.T) {
	t.Helper()
	// 初始化内存数据库
	err := Init(&Config{Development: true})
	require.NoError(t, err)

	// 关闭 SQL 日志输出（只显示错误）
	Get().Logger = Get().Logger.LogMode(gorm_logger.Error)

	// 迁移基础表（不包含 Round 和 Turn，避免 GORM 解析嵌套结构）
	migrates := []any{
		&dao.UserProfile{},
		&dao.CardStat{},
		&dao.CardEffect{},
	}
	err = Get().AutoMigrate(migrates...)
	require.NoError(t, err)

	// 手动创建 rounds 表
	err = Get().Exec(`
		CREATE TABLE IF NOT EXISTS rounds (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			created_at DATETIME,
			updated_at DATETIME,
			deleted_at DATETIME,
			game_id INTEGER,
			round_number INTEGER,
			is_last_round INTEGER,
			complete_reason INTEGER
		)
	`).Error
	require.NoError(t, err)

	// 手动创建 turns 表，避免 GORM 解析 PlayerTurnInfos 嵌套结构
	err = Get().Exec(`
		CREATE TABLE IF NOT EXISTS turns (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			created_at DATETIME,
			updated_at DATETIME,
			deleted_at DATETIME,
			round_id INTEGER,
			turn_number INTEGER,
			turn_start_at INTEGER
		)
	`).Error
	require.NoError(t, err)

	// 手动创建 player_turn_infos 表，turn_submitted_card 作为 JSON 文本存储
	err = Get().Exec(`
		CREATE TABLE IF NOT EXISTS player_turn_infos (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			created_at DATETIME,
			updated_at DATETIME,
			deleted_at DATETIME,
			turn_id INTEGER,
			player_id INTEGER,
			player_status INTEGER,
			temporary_address TEXT,
			turn_submitted_card TEXT
		)
	`).Error
	require.NoError(t, err)
}

// createRound 创建 Round 记录（使用原始 SQL 避免 GORM 解析嵌套结构）
func createRound(t *testing.T, gameID uint, roundNumber uint32) uint {
	t.Helper()
	var roundID uint
	result := Get().Exec(`
		INSERT INTO rounds (game_id, round_number, created_at, updated_at)
		VALUES (?, ?, datetime('now'), datetime('now'))
	`, gameID, roundNumber)
	require.NoError(t, result.Error)
	require.NoError(t, Get().Raw("SELECT last_insert_rowid()").Scan(&roundID).Error)
	return roundID
}

// createTurn 创建 Turn 记录（使用原始 SQL 避免 GORM 解析嵌套结构）
func createTurn(t *testing.T, roundID uint, turnNumber uint32) uint {
	t.Helper()
	var turnID uint
	result := Get().Exec(`
		INSERT INTO turns (round_id, turn_number, created_at, updated_at)
		VALUES (?, ?, datetime('now'), datetime('now'))
	`, roundID, turnNumber)
	require.NoError(t, result.Error)
	require.NoError(t, Get().Raw("SELECT last_insert_rowid()").Scan(&turnID).Error)
	return turnID
}

// createPlayerTurnInfoWithCard 创建 PlayerTurnInfo 并手动序列化 TurnSubmittedCard 为 JSON
func createPlayerTurnInfoWithCard(t *testing.T, turnID uint, playerID int64, tempAddr string, card *dao.TurnSubmittedCard) {
	t.Helper()
	// 序列化 TurnSubmittedCard 为 JSON（不包含 CardEffects，因为它是关联表）
	cardJSON, err := json.Marshal(card)
	require.NoError(t, err)

	// 使用原始 SQL 插入，避免 GORM 解析嵌套结构
	result := Get().Exec(`
		INSERT INTO player_turn_infos (turn_id, player_id, player_status, temporary_address, turn_submitted_card, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, datetime('now'), datetime('now'))
	`, turnID, playerID, proto.PlayerTurnStatus_PLAYER_TURN_CARD_SUBMITTED, tempAddr, string(cardJSON))
	require.NoError(t, result.Error)
}

func TestUpdateCardStatByPlayerIDs_EmptyInput(t *testing.T) {
	setupTestDB(t)

	// 测试空输入
	results, err := UpdateCardStatByPlayerIDs([]int64{})
	require.NoError(t, err)
	require.Empty(t, results)
}

func TestUpdateCardStatByPlayerIDs_NewPlayer(t *testing.T) {
	setupTestDB(t)

	// 创建测试玩家
	playerID := int64(1001)
	profile := &dao.UserProfile{
		PlayerID: playerID,
		Address:  "0x1234567890123456789012345678901234567890",
		Name:     "test_player",
	}
	require.NoError(t, Get().Create(profile).Error)

	// 创建 Round 和 Turn
	roundID := createRound(t, 1, 1)
	turnID := createTurn(t, roundID, 1)

	// 创建 PlayerTurnInfo，包含卡牌提交
	card := &dao.TurnSubmittedCard{
		CardID:          1,
		ElementRelation: proto.ElementRelation_OVER_POWER, // 胜利
	}
	createPlayerTurnInfoWithCard(t, turnID, playerID, "0xtemp123", card)

	// 执行更新
	results, err := UpdateCardStatByPlayerIDs([]int64{playerID})
	require.NoError(t, err)
	require.Len(t, results, 1)

	// 验证结果
	result := results[0]
	require.Equal(t, playerID, result.PlayerID)
	require.Equal(t, uint(1), result.CardID)
	require.Equal(t, uint(1), result.RoundCount)
	require.Equal(t, uint(1), result.UsageCount)
	require.Equal(t, uint(1), result.WinCount)
	require.Equal(t, uint(0), result.LoseCount)
	require.Equal(t, uint(0), result.TieCount)
	require.Equal(t, roundID, result.LastPlayerRoundInfoID)

	// 验证数据库中的记录
	var cardStat dao.CardStat
	require.NoError(t, Get().Where("player_id = ? AND card_id = ?", playerID, 1).First(&cardStat).Error)
	require.Equal(t, uint(1), cardStat.RoundCount)
	require.Equal(t, uint(1), cardStat.UsageCount)
	require.Equal(t, uint(1), cardStat.WinCount)
}

func TestUpdateCardStatByPlayerIDs_ExistingPlayerWithNewData(t *testing.T) {
	setupTestDB(t)

	playerID := int64(1002)
	profile := &dao.UserProfile{
		PlayerID: playerID,
		Address:  "0x2222222222222222222222222222222222222222",
		Name:     "test_player_2",
	}
	require.NoError(t, Get().Create(profile).Error)

	// 创建已有的卡牌统计记录
	existingStat := &dao.CardStat{
		PlayerID:              playerID,
		CardID:                1,
		RoundCount:            2,
		UsageCount:            3,
		WinCount:              2,
		LoseCount:             1,
		TieCount:              0,
		LastPlayerRoundInfoID: 10, // 已处理到 round 10
	}
	require.NoError(t, Get().Create(existingStat).Error)

	// 创建新的 Round（round_id > 10）
	// 先创建一些 dummy rounds 确保 round_id > 10
	for i := 0; i < 12; i++ {
		createRound(t, 2, uint32(i+1))
	}
	roundID := createRound(t, 2, 13)
	require.Greater(t, roundID, uint(10)) // 确保是新 round

	// 创建新的 Turn
	turnID := createTurn(t, roundID, 1)

	// 创建新的 PlayerTurnInfo
	card := &dao.TurnSubmittedCard{
		CardID:          1,
		ElementRelation: proto.ElementRelation_TIE, // 平局
	}
	createPlayerTurnInfoWithCard(t, turnID, playerID, "0xtemp456", card)

	// 执行更新
	results, err := UpdateCardStatByPlayerIDs([]int64{playerID})
	require.NoError(t, err)
	require.Len(t, results, 1)

	// 验证结果：应该累加数据
	result := results[0]
	require.Equal(t, playerID, result.PlayerID)
	require.Equal(t, uint(1), result.CardID)
	require.Equal(t, uint(3), result.RoundCount) // 2 + 1 = 3
	require.Equal(t, uint(4), result.UsageCount) // 3 + 1 = 4
	require.Equal(t, uint(2), result.WinCount)   // 不变
	require.Equal(t, uint(1), result.LoseCount)  // 不变
	require.Equal(t, uint(1), result.TieCount)   // 0 + 1 = 1
	require.Equal(t, roundID, result.LastPlayerRoundInfoID)
}

func TestUpdateCardStatByPlayerIDs_MultipleCards(t *testing.T) {
	setupTestDB(t)

	playerID := int64(1003)
	profile := &dao.UserProfile{
		PlayerID: playerID,
		Address:  "0x3333333333333333333333333333333333333333",
		Name:     "test_player_3",
	}
	require.NoError(t, Get().Create(profile).Error)

	// 创建 Round
	roundID := createRound(t, 3, 1)

	// 创建多个 Turn，每个 Turn 使用不同的卡牌
	turn1ID := createTurn(t, roundID, 1)
	turn2ID := createTurn(t, roundID, 2)

	// 卡牌1：胜利
	card1 := &dao.TurnSubmittedCard{
		CardID:          1,
		ElementRelation: proto.ElementRelation_OVER_POWER,
	}
	createPlayerTurnInfoWithCard(t, turn1ID, playerID, "0xtemp789", card1)

	// 卡牌2：失败
	card2 := &dao.TurnSubmittedCard{
		CardID:          2,
		ElementRelation: proto.ElementRelation_OVER_POWERED,
	}
	createPlayerTurnInfoWithCard(t, turn2ID, playerID, "0xtemp789", card2)

	// 执行更新
	results, err := UpdateCardStatByPlayerIDs([]int64{playerID})
	require.NoError(t, err)
	require.Len(t, results, 2) // 两张卡牌

	// 验证卡牌1
	var card1Stat *dao.CardStat
	for _, r := range results {
		if r.CardID == 1 {
			card1Stat = r
			break
		}
	}
	require.NotNil(t, card1Stat)
	require.Equal(t, uint(1), card1Stat.WinCount)
	require.Equal(t, uint(0), card1Stat.LoseCount)

	// 验证卡牌2
	var card2Stat *dao.CardStat
	for _, r := range results {
		if r.CardID == 2 {
			card2Stat = r
			break
		}
	}
	require.NotNil(t, card2Stat)
	require.Equal(t, uint(0), card2Stat.WinCount)
	require.Equal(t, uint(1), card2Stat.LoseCount)
}

func TestUpdateCardStatByPlayerIDs_NoNewData(t *testing.T) {
	setupTestDB(t)

	playerID := int64(1004)
	profile := &dao.UserProfile{
		PlayerID: playerID,
		Address:  "0x4444444444444444444444444444444444444444",
		Name:     "test_player_4",
	}
	require.NoError(t, Get().Create(profile).Error)

	// 创建已有的卡牌统计记录
	existingStat := &dao.CardStat{
		PlayerID:              playerID,
		CardID:                1,
		RoundCount:            5,
		UsageCount:            10,
		WinCount:              7,
		LoseCount:             3,
		TieCount:              0,
		LastPlayerRoundInfoID: 100, // 已处理到 round 100
	}
	require.NoError(t, Get().Create(existingStat).Error)

	// 不创建新的 Round 和 Turn（没有新数据）

	// 执行更新
	results, err := UpdateCardStatByPlayerIDs([]int64{playerID})
	require.NoError(t, err)
	require.Len(t, results, 1)

	// 验证结果：应该返回现有记录，数据不变
	result := results[0]
	require.Equal(t, playerID, result.PlayerID)
	require.Equal(t, uint(1), result.CardID)
	require.Equal(t, uint(5), result.RoundCount)
	require.Equal(t, uint(10), result.UsageCount)
	require.Equal(t, uint(7), result.WinCount)
}

func TestUpdateCardStatByPlayerIDs_DeduplicatePlayerIDs(t *testing.T) {
	setupTestDB(t)

	playerID := int64(1005)
	profile := &dao.UserProfile{
		PlayerID: playerID,
		Address:  "0x5555555555555555555555555555555555555555",
		Name:     "test_player_5",
	}
	require.NoError(t, Get().Create(profile).Error)

	// 创建 Round 和 Turn
	roundID := createRound(t, 5, 1)
	turnID := createTurn(t, roundID, 1)

	card := &dao.TurnSubmittedCard{
		CardID:          1,
		ElementRelation: proto.ElementRelation_OVER_POWER,
	}
	createPlayerTurnInfoWithCard(t, turnID, playerID, "0xtemp999", card)

	// 测试去重：传入重复的 playerID
	results, err := UpdateCardStatByPlayerIDs([]int64{playerID, playerID, playerID})
	require.NoError(t, err)
	require.Len(t, results, 1) // 应该只处理一次

	// 验证数据库中只有一条记录
	var count int64
	require.NoError(t, Get().Model(&dao.CardStat{}).Where("player_id = ?", playerID).Count(&count).Error)
	require.Equal(t, int64(1), count)
}

func TestUpdateCardStatByPlayerIDs_MultiplePlayers(t *testing.T) {
	setupTestDB(t)

	// 创建两个玩家
	player1ID := int64(1006)
	player2ID := int64(1007)

	profile1 := &dao.UserProfile{
		PlayerID: player1ID,
		Address:  "0x6666666666666666666666666666666666666666",
		Name:     "test_player_6",
	}
	require.NoError(t, Get().Create(profile1).Error)

	profile2 := &dao.UserProfile{
		PlayerID: player2ID,
		Address:  "0x7777777777777777777777777777777777777777",
		Name:     "test_player_7",
	}
	require.NoError(t, Get().Create(profile2).Error)

	// 为玩家1创建数据
	round1ID := createRound(t, 6, 1)
	turn1ID := createTurn(t, round1ID, 1)

	card1 := &dao.TurnSubmittedCard{
		CardID:          1,
		ElementRelation: proto.ElementRelation_OVER_POWER,
	}
	createPlayerTurnInfoWithCard(t, turn1ID, player1ID, "0xtemp111", card1)

	// 为玩家2创建数据
	round2ID := createRound(t, 7, 1)
	turn2ID := createTurn(t, round2ID, 1)

	card2 := &dao.TurnSubmittedCard{
		CardID:          2,
		ElementRelation: proto.ElementRelation_OVER_POWERED,
	}
	createPlayerTurnInfoWithCard(t, turn2ID, player2ID, "0xtemp222", card2)

	// 执行更新
	results, err := UpdateCardStatByPlayerIDs([]int64{player1ID, player2ID})
	require.NoError(t, err)
	require.Len(t, results, 2) // 两个玩家各一条记录

	// 验证玩家1
	var player1Stat *dao.CardStat
	for _, r := range results {
		if r.PlayerID == player1ID {
			player1Stat = r
			break
		}
	}
	require.NotNil(t, player1Stat)
	require.Equal(t, uint(1), player1Stat.CardID)
	require.Equal(t, uint(1), player1Stat.WinCount)

	// 验证玩家2
	var player2Stat *dao.CardStat
	for _, r := range results {
		if r.PlayerID == player2ID {
			player2Stat = r
			break
		}
	}
	require.NotNil(t, player2Stat)
	require.Equal(t, uint(2), player2Stat.CardID)
	require.Equal(t, uint(1), player2Stat.LoseCount)
}

func TestUpdateCardStatByPlayerIDs_SyncRoundCountForOtherCards(t *testing.T) {
	setupTestDB(t)

	playerID := int64(1008)
	profile := &dao.UserProfile{
		PlayerID: playerID,
		Address:  "0x8888888888888888888888888888888888888888",
		Name:     "test_player_8",
	}
	require.NoError(t, Get().Create(profile).Error)

	// 创建已有的卡牌统计记录（卡牌1）
	existingStat1 := &dao.CardStat{
		PlayerID:              playerID,
		CardID:                1,
		RoundCount:            3,
		UsageCount:            5,
		WinCount:              3,
		LoseCount:             2,
		TieCount:              0,
		LastPlayerRoundInfoID: 20,
	}
	require.NoError(t, Get().Create(existingStat1).Error)

	// 创建已有的卡牌统计记录（卡牌2，但没有新数据）
	existingStat2 := &dao.CardStat{
		PlayerID:              playerID,
		CardID:                2,
		RoundCount:            3,
		UsageCount:            2,
		WinCount:              1,
		LoseCount:             1,
		TieCount:              0,
		LastPlayerRoundInfoID: 20,
	}
	require.NoError(t, Get().Create(existingStat2).Error)

	// 创建新的 Round（round_id > 20）
	// 先创建一些 dummy rounds 确保 round_id > 20
	for i := 0; i < 22; i++ {
		createRound(t, 8, uint32(i+1))
	}
	roundID := createRound(t, 8, 23)
	require.Greater(t, roundID, uint(20))

	// 创建新的 Turn，只使用卡牌1
	turnID := createTurn(t, roundID, 1)

	card := &dao.TurnSubmittedCard{
		CardID:          1, // 只使用卡牌1
		ElementRelation: proto.ElementRelation_OVER_POWER,
	}
	createPlayerTurnInfoWithCard(t, turnID, playerID, "0xtemp333", card)

	// 执行更新
	results, err := UpdateCardStatByPlayerIDs([]int64{playerID})
	require.NoError(t, err)
	require.Len(t, results, 2) // 两张卡牌都应该返回

	// 验证卡牌1：有新数据，应该更新
	var card1Stat *dao.CardStat
	for _, r := range results {
		if r.CardID == 1 {
			card1Stat = r
			break
		}
	}
	require.NotNil(t, card1Stat)
	require.Equal(t, uint(4), card1Stat.RoundCount) // 3 + 1 = 4
	require.Equal(t, uint(6), card1Stat.UsageCount) // 5 + 1 = 6

	// 验证卡牌2：没有新数据，但 round_count 应该同步更新
	var card2Stat *dao.CardStat
	for _, r := range results {
		if r.CardID == 2 {
			card2Stat = r
			break
		}
	}
	require.NotNil(t, card2Stat)
	require.Equal(t, uint(4), card2Stat.RoundCount) // 3 + 1 = 4（同步更新）
	require.Equal(t, uint(2), card2Stat.UsageCount) // 不变
}
