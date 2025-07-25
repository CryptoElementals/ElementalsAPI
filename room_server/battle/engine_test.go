package battle

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"strings"
	"testing"

	"github.com/CryptoElementals/common/config"
	"github.com/CryptoElementals/common/db"
	"github.com/CryptoElementals/common/log"
	dao "github.com/CryptoElementals/common/models"
	pb "github.com/CryptoElementals/common/rpc/proto"
	"github.com/stretchr/testify/require"

	// 确保导入GetBattleInfoResponse和相关结构体
	"github.com/CryptoElementals/common/server/api/battle"
)

func TestExecuteRoundNormal(t *testing.T) {
	initTestEnv(t)

	prepareCards(t)

	engine := NewBattleEngine()

	input := &RoundInput{
		RoundNumber: 2,
		Players: []PlayerRoundInput{
			{
				WalletAddress:    "1_address",
				TemporaryAddress: "PLAYER1_TEMP_ADDRESS",
				HP:               1500,
				Cards:            []int{1, 2, 5},
				LostHP:           500,
			},
			{
				WalletAddress:    "2_address",
				TemporaryAddress: "PLAYER2_TEMP_ADDRESS",
				HP:               1500,
				Cards:            []int{1, 2, 3},
				LostHP:           2500,
			},
		},
	}

	result, err := engine.ExecuteRound(input)

	if err != nil {
		t.Errorf("ExecuteRound failed with error: %v", err)
		return
	}

	if result == nil {
		t.Error("ExecuteRound returned nil result")
		return
	}

	jsonData, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		t.Errorf("Failed to marshal result to JSON: %v", err)
		return
	}

	t.Logf("Round Result (JSON):\n%s", string(jsonData))
	t.Log("Test completed successfully - check the JSON output above for manual verification")
}

func TestExecuteRoundProto(t *testing.T) {
	initTestEnv(t)
	prepareCards(t)

	engine := NewBattleEngine()

	protoInput := &pb.RoundInput{
		RoundNumber: 3,
		Players: []*pb.PlayerRoundInput{
			{
				WalletAddress:    "player1_address",
				TemporaryAddress: "PLAYER1_TEMP_ADDRESS",
				Cards:            []int32{4, 1, 3},
				HP:               3000,
				LostHP:           2000,
			},
			{
				WalletAddress:    "player2_address",
				TemporaryAddress: "PLAYER2_TEMP_ADDRESS",
				Cards:            []int32{1, 3, 4},
				HP:               3000,
				LostHP:           2000,
			},
		},
	}

	roundResult, gameResult, err := engine.ExecuteRoundProto(protoInput)
	require.NoError(t, err)
	require.NotNil(t, roundResult)

	rrJSON, _ := json.MarshalIndent(roundResult, "", "  ")
	t.Logf("RoundResult (JSON):\n%s", string(rrJSON))

	if gameResult != nil {
		grJSON, _ := json.MarshalIndent(gameResult, "", "  ")
		t.Logf("GameResult (JSON):\n%s", string(grJSON))
	}
}

func TestExecuteRoundProtoFromFile(t *testing.T) {
	initTestEnv(t)
	prepareCards(t)
	// 读取测试输入文件
	data, err := ioutil.ReadFile("/data/ws_tj/BeastRoyaleBackend/test/api/battle/test_inputs.json")
	require.NoError(t, err)

	var testInputs []pb.RoundInput
	err = json.Unmarshal(data, &testInputs)
	require.NoError(t, err)

	engine := NewBattleEngine()

	for i, input := range testInputs {
		t.Run(fmt.Sprintf("TestCase%d", i+1), func(t *testing.T) {
			roundResult, gameResult, err := engine.ExecuteRoundProto(&input)
			require.NoError(t, err)
			require.NotNil(t, roundResult)

			// 确保GetBattleInfoResponse和相关结构体在当前文件中定义或导入
			// 确保类型转换正确，例如将int32转换为int或uint32
			// 确保使用正确的字段名称

			// 将结果转换为GetBattleInfoResponse格式
			response := &battle.GetBattleInfoResponse{
				RoundResult: &battle.RoundResult{
					Round:      roundResult.RoundNumber,
					IsGameOver: roundResult.IsGameOver,
					Players:    make([]battle.PlayerRoundStat, len(roundResult.Players)),
				},
			}

			for i, player := range roundResult.Players {
				response.RoundResult.Players[i] = battle.PlayerRoundStat{
					PlayerAddress: player.WalletAddress,
					IsSelf:        false,
					CardStats:     make([]battle.PlayerCardStat, len(player.CardStats)),
				}

				for j, cardStat := range player.CardStats {
					response.RoundResult.Players[i].CardStats[j] = battle.PlayerCardStat{
						CardNumber:       cardStat.CardNumber,
						CardID:           cardStat.CardID,
						HPBefore:         cardStat.HPBefore,
						HPAfter:          cardStat.HPAfter,
						MultiplierBefore: cardStat.MultiplierBefore,
						MultiplierAfter:  cardStat.MultiplierAfter,
						Description:      cardStat.Description,
						ElementRelation:  int32(cardStat.ElementRelation),
					}
				}
			}

			if gameResult != nil {
				response.GameResult = &battle.GameResult{
					Winner:              gameResult.WinnerWalletAddress,
					GameResultType:      uint32(gameResult.GameResultType),
					GameFinalMultiplier: uint32(gameResult.Multiplier),
					Reward: &battle.BattleReward{
						PlayerRewards: make([]battle.PlayerReward, len(gameResult.Reward.PlayerRewards)),
						SystemFee:     int32(gameResult.Reward.SystemFee),
					},
				}

				for i, reward := range gameResult.Reward.PlayerRewards {
					response.GameResult.Reward.PlayerRewards[i] = battle.PlayerReward{
						PlayerAddress: reward.WalletAddress,
						TokenChange:   int32(reward.TokenChange),
						PointChange:   int32(reward.PointChange),
					}
				}
			}

			// 打印转换后的结果
			responseJSON, err := json.MarshalIndent(response, "", "  ")
			require.NoError(t, err)
			t.Logf("GetBattleInfoResponse (JSON):\n%s", string(responseJSON))

			// 保存到文件
			ioutil.WriteFile(fmt.Sprintf("/data/ws_tj/BeastRoyaleBackend/test/api/battle/mock/response_case_%d.json", i+1), responseJSON, 0644)
		})
	}
}

func TestExecuteRoundThreePlayers(t *testing.T) {
	initTestEnv(t)
	prepareCards(t)

	engine := NewBattleEngine()

	// 测试三人游戏 - NORMAL情况（没人血量为0，第3轮比较血量）
	input := &RoundInput{
		RoundNumber: 3,
		Players: []PlayerRoundInput{
			{
				WalletAddress:    "1_address",
				TemporaryAddress: "PLAYER1_TEMP_ADDRESS",
				HP:               2800,           // 最高血量，应该获胜
				Cards:            []int{4, 1, 3}, // Fire, Metal, Water
				LostHP:           1000,           // 倍率2
			},
			{
				WalletAddress:    "2_address",
				TemporaryAddress: "PLAYER2_TEMP_ADDRESS",
				HP:               2500,           // 中等血量，输家
				Cards:            []int{1, 3, 4}, // Metal, Water, Fire
				LostHP:           2500,           // 倍率5
			},
			{
				WalletAddress:    "3_address",
				TemporaryAddress: "PLAYER3_TEMP_ADDRESS",
				HP:               2200,           // 最低血量，输家
				Cards:            []int{3, 4, 1}, // Water, Fire, Metal
				LostHP:           3500,           // 会在战斗中增加到5500，倍率8
			},
		},
	}

	result, err := engine.ExecuteRound(input)

	if err != nil {
		t.Errorf("ExecuteRound failed with error: %v", err)
		return
	}

	if result == nil {
		t.Error("ExecuteRound returned nil result")
		return
	}

	// 验证游戏结果
	if !result.IsGameOver {
		t.Error("Game should be over after round 3")
	}

	if result.GameResult == nil {
		t.Error("GameResult should not be nil when game is over")
	} else {
		// 验证游戏类型为NORMAL（没人血量为0）
		if result.GameResult.GameResultType != GAME_NORMAL {
			t.Errorf("Expected GAME_NORMAL, got %v", result.GameResult.GameResultType)
		}

		// 验证赢家是player1
		if result.GameResult.WinnerWalletAddress != "player1_address" {
			t.Errorf("Expected winner to be player1_address, got %s", result.GameResult.WinnerWalletAddress)
		}

		// 验证最终倍率是输家中最大的（player3的倍率8）
		if result.GameResult.Multiplier != 8 {
			t.Errorf("Expected multiplier to be 8, got %d", result.GameResult.Multiplier)
		}

		// 验证奖励分配
		if len(result.GameResult.Reward.PlayerRewards) != 3 {
			t.Errorf("Expected 3 player rewards, got %d", len(result.GameResult.Reward.PlayerRewards))
		}

		// 找到每个玩家的奖励
		rewardMap := make(map[string]*PlayerReward)
		for i := range result.GameResult.Reward.PlayerRewards {
			reward := &result.GameResult.Reward.PlayerRewards[i]
			rewardMap[reward.WalletAddress] = reward
		}

		// 验证赢家奖励（获得token和积分）
		player1Reward := rewardMap["player1_address"]
		if player1Reward == nil {
			t.Error("Player1 reward not found")
		} else if player1Reward.TokenChange <= 0 {
			t.Errorf("Player1 should gain tokens, got %d", player1Reward.TokenChange)
		}

		// 验证输家奖励（损失token但获得少量积分）
		player2Reward := rewardMap["player2_address"]
		if player2Reward == nil {
			t.Error("Player2 reward not found")
		} else if player2Reward.TokenChange >= 0 {
			t.Errorf("Player2 should lose tokens, got %d", player2Reward.TokenChange)
		}

		player3Reward := rewardMap["player3_address"]
		if player3Reward == nil {
			t.Error("Player3 reward not found")
		} else if player3Reward.TokenChange >= 0 {
			t.Errorf("Player3 should lose tokens, got %d", player3Reward.TokenChange)
		}
	}

	jsonData, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		t.Errorf("Failed to marshal result to JSON: %v", err)
		return
	}

	t.Logf("Three Players Round Result (JSON):\n%s", string(jsonData))
	t.Log("Three players test completed successfully")
}

func TestExecuteRoundThreePlayersKO(t *testing.T) {
	initTestEnv(t)
	prepareCards(t)

	engine := NewBattleEngine()

	// 测试三人游戏 - KO情况（有人血量为0）
	input := &RoundInput{
		RoundNumber: 3, // 第2轮就有人血量为0
		Players: []PlayerRoundInput{
			{
				WalletAddress:    "1_address",
				TemporaryAddress: "PLAYER1_TEMP_ADDRESS",
				HP:               1500,           // 会被打死
				Cards:            []int{1, 3, 4}, // Metal, Water, Fire
				LostHP:           4000,           // 会在战斗中增加到6000，倍率9
			},
			{
				WalletAddress:    "2_address",
				TemporaryAddress: "PLAYER2_TEMP_ADDRESS",
				HP:               2500,           // 存活，赢家
				Cards:            []int{4, 1, 3}, // Fire, Metal, Water
				LostHP:           1500,           // 倍率3
			},
			{
				WalletAddress:    "3_address",
				TemporaryAddress: "PLAYER3_TEMP_ADDRESS",
				HP:               2000,           // 存活，赢家
				Cards:            []int{3, 4, 2}, // Water, Fire, Wood
				LostHP:           2000,           // 倍率4
			},
		},
	}

	result, err := engine.ExecuteRound(input)

	if err != nil {
		t.Errorf("ExecuteRound failed with error: %v", err)
		return
	}

	if result == nil {
		t.Error("ExecuteRound returned nil result")
		return
	}

	// 验证游戏结果
	if !result.IsGameOver {
		t.Error("Game should be over when someone has 0 HP")
	}

	if result.GameResult == nil {
		t.Error("GameResult should not be nil when game is over")
	} else {
		// 验证游戏类型为KO
		if result.GameResult.GameResultType != GAME_KO {
			t.Errorf("Expected GAME_KO, got %v", result.GameResult.GameResultType)
		}

		// 验证有多个赢家（用|分割）
		winners := result.GameResult.WinnerWalletAddress
		if !strings.Contains(winners, "player2_address") || !strings.Contains(winners, "player3_address") {
			t.Errorf("Expected both player2 and player3 as winners, got %s", winners)
		}

		// 验证最终倍率是输家的倍率（player1的倍率9）
		if result.GameResult.Multiplier != 9 {
			t.Errorf("Expected multiplier to be 9, got %d", result.GameResult.Multiplier)
		}
	}

	jsonData, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		t.Errorf("Failed to marshal result to JSON: %v", err)
		return
	}

	t.Logf("Three Players KO Round Result (JSON):\n%s", string(jsonData))
	t.Log("Three players KO test completed successfully")
}

func TestExecuteRoundThreePlayersTie(t *testing.T) {
	initTestEnv(t)
	prepareCards(t)

	engine := NewBattleEngine()

	// 测试三人游戏 - 平局情况（所有人血量相同）
	input := &RoundInput{
		RoundNumber: 3,
		Players: []PlayerRoundInput{
			{
				WalletAddress:    "player1_address",
				TemporaryAddress: "PLAYER1_TEMP_ADDRESS",
				HP:               2500, // 相同血量
				Cards:            []int{4, 1, 3},
				LostHP:           1000, // 倍率2
			},
			{
				WalletAddress:    "player2_address",
				TemporaryAddress: "PLAYER2_TEMP_ADDRESS",
				HP:               2500, // 相同血量
				Cards:            []int{1, 3, 4},
				LostHP:           1500, // 倍率3
			},
			{
				WalletAddress:    "player3_address",
				TemporaryAddress: "PLAYER3_TEMP_ADDRESS",
				HP:               2500, // 相同血量
				Cards:            []int{3, 4, 1},
				LostHP:           2000, // 倍率4
			},
		},
	}

	result, err := engine.ExecuteRound(input)

	if err != nil {
		t.Errorf("ExecuteRound failed with error: %v", err)
		return
	}

	if result == nil {
		t.Error("ExecuteRound returned nil result")
		return
	}

	// 验证游戏结果
	if !result.IsGameOver {
		t.Error("Game should be over after round 3")
	}

	if result.GameResult == nil {
		t.Error("GameResult should not be nil when game is over")
	} else {
		// 验证游戏类型为平局
		if result.GameResult.GameResultType != GAME_TIE {
			t.Errorf("Expected GAME_TIE, got %v", result.GameResult.GameResultType)
		}

		// 验证没有赢家
		if result.GameResult.WinnerWalletAddress != "" {
			t.Errorf("Expected no winner in tie game, got %s", result.GameResult.WinnerWalletAddress)
		}

		// 验证倍率为1（平局情况）
		if result.GameResult.Multiplier != 1 {
			t.Errorf("Expected multiplier to be 1 for tie game, got %d", result.GameResult.Multiplier)
		}

		// 验证所有玩家都扣除相同的token
		if len(result.GameResult.Reward.PlayerRewards) != 3 {
			t.Errorf("Expected 3 player rewards, got %d", len(result.GameResult.Reward.PlayerRewards))
		}

		for _, reward := range result.GameResult.Reward.PlayerRewards {
			if reward.TokenChange >= 0 {
				t.Errorf("All players should lose tokens in tie game, player %s got %d", reward.WalletAddress, reward.TokenChange)
			}
			if reward.PointChange <= 0 {
				t.Errorf("All players should gain points in tie game, player %s got %d", reward.WalletAddress, reward.PointChange)
			}
		}
	}

	jsonData, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		t.Errorf("Failed to marshal result to JSON: %v", err)
		return
	}

	t.Logf("Three Players Tie Round Result (JSON):\n%s", string(jsonData))
	t.Log("Three players tie test completed successfully")
}

// ------------------ 公共辅助函数 ------------------

func initTestEnv(t *testing.T) {
	t.Helper()

	// 只初始化一次（并发测试时保证安全）
	if err := log.InitGlobalLogger(&log.Config{Development: true, Level: "debug"}); err != nil {
		// 如果已初始化会返回错误，忽略
	}

	if db.Get() == nil {
		require.NoError(t, db.Init(&db.Config{Development: true}))
		require.NoError(t, db.MigrateMemDb())
	}

	// 加载配置文件（只加载一次）
	if config.GameParams.MaxHP == 0 {
		cfgPath := "../../config.yaml" // 相对路径：从 room_server/battle 到项目根目录
		if err := config.InitRSConfig(cfgPath); err != nil {
			t.Fatalf("failed to load config: %v", err)
		}
	}
}

func prepareCards(t *testing.T) {
	t.Helper()
	cards := []dao.Card{
		{CardID: 1, ElementType: "Metal", Level: "normal", LifeForce: 500, Attack: 1000, Defense: 500, Name: "Kylin", Description: "Kylin clad in armor, representing strength and protection"},
		{CardID: 2, ElementType: "Wood", Level: "normal", LifeForce: 500, Attack: 1000, Defense: 500, Name: "Forest Spirit", Description: "Forest Spirit controlling the cycle of life and death"},
		{CardID: 3, ElementType: "Water", Level: "normal", LifeForce: 500, Attack: 1000, Defense: 500, Name: "Siren", Description: "Siren, half-human half-beast, possessing enchanting charm"},
		{CardID: 4, ElementType: "Fire", Level: "normal", LifeForce: 500, Attack: 1000, Defense: 500, Name: "Phoenix", Description: "Phoenix with flames and rebirth, symbolizing eternal life"},
		{CardID: 5, ElementType: "Earth", Level: "normal", LifeForce: 500, Attack: 1000, Defense: 500, Name: "World Turtle", Description: "World Turtle, steady and powerful with immense strength"},
	}
	require.NoError(t, db.Get().Save(&cards).Error)
}

func TestOfflinePlayerWithLessThan3Cards(t *testing.T) {
	// 初始化测试环境
	initTestEnv(t)
	prepareCards(t)

	engine := NewBattleEngine()

	// 测试案例1：一个玩家卡牌数量少于3（应该被判为离线并直接失败）
	input := &RoundInput{
		RoundNumber: 1,
		Players: []PlayerRoundInput{
			{
				WalletAddress:    "player1_address",
				TemporaryAddress: "PLAYER1_TEMP_ADDRESS",
				Cards:            []int{1, 2, 3}, // 3张卡，正常
				HP:               3000,
				LostHP:           0,
				Status:           PLAYER_ONLINE,
			},
			{
				WalletAddress:    "player2_address",
				TemporaryAddress: "PLAYER2_TEMP_ADDRESS",
				Cards:            []int{1}, // 只有1张卡，少于3张
				HP:               3000,
				LostHP:           0,
				Status:           PLAYER_ONLINE,
			},
		},
	}

	result, err := engine.ExecuteRound(input)
	if err != nil {
		t.Fatalf("执行回合失败: %v", err)
	}

	// 验证游戏应该结束
	if !result.IsGameOver {
		t.Errorf("期望游戏结束，但游戏仍在继续")
	}

	// 验证结果类型应该是KO
	if result.GameResult.GameResultType != GAME_KO {
		t.Errorf("期望游戏结果类型是KO，实际得到: %v", result.GameResult.GameResultType)
	}

	// 验证player1应该获胜
	if result.GameResult.WinnerWalletAddress != "player1_address" {
		t.Errorf("期望player1获胜，实际获胜者: %s", result.GameResult.WinnerWalletAddress)
	}

	t.Logf("测试案例1通过：玩家卡牌不足被判为离线并失败")
}

func TestAllPlayersOffline(t *testing.T) {
	// 初始化测试环境
	initTestEnv(t)
	prepareCards(t)

	engine := NewBattleEngine()

	// 测试案例2：所有玩家卡牌数量都少于3（应该返回错误）
	input := &RoundInput{
		RoundNumber: 1,
		Players: []PlayerRoundInput{
			{
				WalletAddress:    "player1_address",
				TemporaryAddress: "PLAYER1_TEMP_ADDRESS",
				Cards:            []int{1}, // 只有1张卡
				HP:               3000,
				LostHP:           0,
				Status:           PLAYER_ONLINE,
			},
			{
				WalletAddress:    "player2_address",
				TemporaryAddress: "PLAYER2_TEMP_ADDRESS",
				Cards:            []int{2}, // 只有1张卡
				HP:               3000,
				LostHP:           0,
				Status:           PLAYER_ONLINE,
			},
		},
	}

	_, err := engine.ExecuteRound(input)
	if err == nil {
		t.Errorf("期望返回错误（所有玩家都离线），但没有错误")
	}

	expectedError := "at least one player must be online"
	if err.Error() != expectedError {
		t.Errorf("期望错误信息: %s, 实际得到: %s", expectedError, err.Error())
	}

	t.Logf("测试案例2通过：所有玩家卡牌不足时正确返回错误")
}

func TestMultipleOfflinePlayers(t *testing.T) {
	// 初始化测试环境
	initTestEnv(t)
	prepareCards(t)

	engine := NewBattleEngine()

	// 测试案例3：多个玩家中有些离线
	input := &RoundInput{
		RoundNumber: 1,
		Players: []PlayerRoundInput{
			{
				WalletAddress:    "player1_address",
				TemporaryAddress: "PLAYER1_TEMP_ADDRESS",
				Cards:            []int{1, 2, 3}, // 3张卡，正常
				HP:               3000,
				LostHP:           0,
				Status:           PLAYER_ONLINE,
			},
			{
				WalletAddress:    "player2_address",
				TemporaryAddress: "PLAYER2_TEMP_ADDRESS",
				Cards:            []int{1}, // 只有1张卡，离线
				HP:               3000,
				LostHP:           0,
				Status:           PLAYER_ONLINE,
			},
			{
				WalletAddress:    "player3_address",
				TemporaryAddress: "PLAYER3_TEMP_ADDRESS",
				Cards:            []int{2}, // 只有1张卡，离线
				HP:               3000,
				LostHP:           0,
				Status:           PLAYER_ONLINE,
			},
		},
	}

	result, err := engine.ExecuteRound(input)
	if err != nil {
		t.Fatalf("执行回合失败: %v", err)
	}

	// 验证游戏应该结束
	if !result.IsGameOver {
		t.Errorf("期望游戏结束，但游戏仍在继续")
	}

	// 验证结果类型应该是KO
	if result.GameResult.GameResultType != GAME_KO {
		t.Errorf("期望游戏结果类型是KO，实际得到: %v", result.GameResult.GameResultType)
	}

	// 验证player1应该获胜
	if result.GameResult.WinnerWalletAddress != "player1_address" {
		t.Errorf("期望player1获胜，实际获胜者: %s", result.GameResult.WinnerWalletAddress)
	}

	t.Logf("测试案例3通过：多玩家中部分离线的情况处理正确")
}

func TestNewOfflinePlayerLogic(t *testing.T) {
	initTestEnv(t)
	prepareCards(t)

	engine := NewBattleEngine()

	// 测试案例1：部分玩家卡牌不足（离线），在线玩家获胜
	input := &RoundInput{
		RoundNumber: 1,
		Players: []PlayerRoundInput{
			{
				WalletAddress:    "player1_address",
				TemporaryAddress: "PLAYER1_TEMP_ADDRESS",
				Cards:            []int{1, 2, 3}, // 3张卡，正常
				HP:               3000,
				LostHP:           500, // 有一些失血
			},
			{
				WalletAddress:    "player2_address",
				TemporaryAddress: "PLAYER2_TEMP_ADDRESS",
				Cards:            []int{1}, // 只有1张卡，会被设为离线
				HP:               2500,
				LostHP:           1000, // 更多失血，倍率更高
			},
		},
	}

	result, err := engine.ExecuteRound(input)
	if err != nil {
		t.Fatalf("执行回合失败: %v", err)
	}

	// 验证游戏应该结束
	if !result.IsGameOver {
		t.Errorf("期望游戏结束，但游戏仍在继续")
	}

	// 验证结果类型应该是KO
	if result.GameResult.GameResultType != GAME_KO {
		t.Errorf("期望游戏结果类型是KO，实际得到: %v", result.GameResult.GameResultType)
	}

	// 验证player1应该获胜
	if result.GameResult.WinnerWalletAddress != "player1_address" {
		t.Errorf("期望player1获胜，实际获胜者: %s", result.GameResult.WinnerWalletAddress)
	}

	// 验证player1的血量保持不变（没有卡牌对战）
	if input.Players[0].HP != 3000 {
		t.Errorf("期望player1血量保持3000，实际: %d", input.Players[0].HP)
	}

	// 验证player2的血量保持不变（没有被额外扣血）
	if input.Players[1].HP != 2500 {
		t.Errorf("期望player2血量保持2500，实际: %d", input.Players[1].HP)
	}

	// 验证使用的是输家（player2）的倍率
	expectedMultiplier := uint32(4) // LostHP=1000 对应的倍率
	if result.GameResult.Multiplier != expectedMultiplier {
		t.Errorf("期望倍率为%d，实际得到: %d", expectedMultiplier, result.GameResult.Multiplier)
	}

	t.Logf("测试案例1通过：部分玩家离线时在线玩家获胜")
}

func TestAllPlayersOfflineLogic(t *testing.T) {
	initTestEnv(t)
	prepareCards(t)

	engine := NewBattleEngine()

	// 测试案例2：所有玩家都卡牌不足（离线），按血量判断胜负
	input := &RoundInput{
		RoundNumber: 1,
		Players: []PlayerRoundInput{
			{
				WalletAddress:    "player1_address",
				TemporaryAddress: "PLAYER1_TEMP_ADDRESS",
				Cards:            []int{1}, // 只有1张卡
				HP:               3000,     // 血量更高
				LostHP:           500,
			},
			{
				WalletAddress:    "player2_address",
				TemporaryAddress: "PLAYER2_TEMP_ADDRESS",
				Cards:            []int{2}, // 只有1张卡
				HP:               2000,     // 血量更低
				LostHP:           1500,     // 失血更多，倍率更高
			},
		},
	}

	result, err := engine.ExecuteRound(input)
	if err != nil {
		t.Fatalf("执行回合失败: %v", err)
	}

	// 验证游戏应该结束
	if !result.IsGameOver {
		t.Errorf("期望游戏结束，但游戏仍在继续")
	}

	// 验证结果类型应该是NORMAL（按血量判断）
	if result.GameResult.GameResultType != GAME_NORMAL {
		t.Errorf("期望游戏结果类型是NORMAL，实际得到: %v", result.GameResult.GameResultType)
	}

	// 验证血量高的player1应该获胜
	if result.GameResult.WinnerWalletAddress != "player1_address" {
		t.Errorf("期望player1获胜，实际获胜者: %s", result.GameResult.WinnerWalletAddress)
	}

	// 验证使用的是输家（player2）的倍率
	expectedMultiplier := uint32(5) // LostHP=1500 对应的倍率
	if result.GameResult.Multiplier != expectedMultiplier {
		t.Errorf("期望倍率为%d，实际得到: %d", expectedMultiplier, result.GameResult.Multiplier)
	}

	t.Logf("测试案例2通过：所有玩家离线时按血量判断胜负")
}

func TestAllPlayersOfflineTie(t *testing.T) {
	initTestEnv(t)
	prepareCards(t)

	engine := NewBattleEngine()

	// 测试案例3：所有玩家都离线且血量相同，应该是平局
	input := &RoundInput{
		RoundNumber: 1,
		Players: []PlayerRoundInput{
			{
				WalletAddress:    "1_address",
				TemporaryAddress: "PLAYER1_TEMP_ADDRESS",
				Cards:            []int{1}, // 只有1张卡
				HP:               3000,     // 血量相同
				LostHP:           500,
			},
			{
				WalletAddress:    "2_address",
				TemporaryAddress: "PLAYER2_TEMP_ADDRESS",
				Cards:            []int{2}, // 只有1张卡
				HP:               3000,     // 血量相同
				LostHP:           1000,     // 失血更多，倍率更高
			},
		},
	}

	result, err := engine.ExecuteRound(input)
	if err != nil {
		t.Fatalf("执行回合失败: %v", err)
	}

	// 完整打印结果
	jsonData, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		t.Errorf("Failed to marshal result to JSON: %v", err)
		return
	}

	t.Logf("测试结果 (JSON):\n%s", string(jsonData))
}

// go test -v ./room_server/battle/ -run TestExecuteRoundNormal
// go test -v ./room_server/battle/ -run TestExecuteRoundProto
//go test -v ./room_server/battle/ -run TestExecuteRoundProtoFromFile | tee test/api/battle/test_output.log

//go test -v ./room_server/battle/ -run TestExecuteRoundThreePlayers
