package battle

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"testing"

	"github.com/CryptoElementals/common/config"
	"github.com/CryptoElementals/common/db"
	"github.com/CryptoElementals/common/log"
	dao "github.com/CryptoElementals/common/models"
	pb "github.com/CryptoElementals/common/rpc/proto"
	"github.com/CryptoElementals/common/server/api"
	"github.com/stretchr/testify/require"

	// 确保导入GetBattleInfoResponse和相关结构体

	"google.golang.org/protobuf/encoding/protojson"
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
				Commitment:       []byte{1},
			},
			{
				WalletAddress:    "2_address",
				TemporaryAddress: "PLAYER2_TEMP_ADDRESS",
				HP:               1500,
				Cards:            []int{1, 2, 3},
				LostHP:           2500,
				Commitment:       []byte{1},
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
				Cards:            []int32{},
				HP:               500,
				LostHP:           4000,
				Commitment:       []byte(""),
			},
			{
				WalletAddress:    "player2_address",
				TemporaryAddress: "PLAYER2_TEMP_ADDRESS",
				Cards:            []int32{},
				HP:               1500,
				LostHP:           8000,
				Commitment:       []byte("dummy"),
			},
		},
	}

	roundResult, gameResult, err := engine.ExecuteRoundProto(protoInput)
	require.NoError(t, err)
	require.NotNil(t, roundResult)

	protoJSON := protojson.MarshalOptions{EmitUnpopulated: true, UseProtoNames: true, Indent: "  "}
	rrJSON, _ := protoJSON.Marshal(roundResult)
	t.Logf("RoundResult (JSON):\n%s", string(rrJSON))

	if gameResult != nil {
		grJSON, _ := protoJSON.Marshal(gameResult)
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
			response := &api.GetBattleInfoResponse{
				RoundResult: &api.RoundResult{
					Round:      roundResult.RoundNumber,
					IsGameOver: roundResult.IsGameOver,
					Players:    make([]api.PlayerRoundStat, len(roundResult.Players)),
				},
			}

			for i, player := range roundResult.Players {
				response.RoundResult.Players[i] = api.PlayerRoundStat{
					PlayerAddress: player.WalletAddress,
					IsSelf:        false,
					CardStats:     make([]api.PlayerCardStat, len(player.CardStats)),
				}

				for j, cardStat := range player.CardStats {
					response.RoundResult.Players[i].CardStats[j] = api.PlayerCardStat{
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
				response.GameResult = &api.GameResult{
					Winner:              gameResult.WinnerWalletAddress,
					GameResultType:      uint32(gameResult.GameResultType),
					GameFinalMultiplier: uint32(gameResult.Multiplier),
					Reward: &api.BattleReward{
						PlayerRewards: make([]api.PlayerReward, len(gameResult.Reward.PlayerRewards)),
						SystemFee:     int32(gameResult.Reward.SystemFee),
					},
				}

				for i, reward := range gameResult.Reward.PlayerRewards {
					response.GameResult.Reward.PlayerRewards[i] = api.PlayerReward{
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
				Commitment:       []byte{1},
			},
			{
				WalletAddress:    "2_address",
				TemporaryAddress: "PLAYER2_TEMP_ADDRESS",
				HP:               2500,           // 中等血量，输家
				Cards:            []int{1, 3, 4}, // Metal, Water, Fire
				LostHP:           2500,           // 倍率5
				Commitment:       []byte{1},
			},
			{
				WalletAddress:    "3_address",
				TemporaryAddress: "PLAYER3_TEMP_ADDRESS",
				HP:               2200,           // 最低血量，输家
				Cards:            []int{3, 4, 1}, // Water, Fire, Metal
				LostHP:           3500,           // 会在战斗中增加到5500，倍率8
				Commitment:       []byte{1},
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
				Commitment:       []byte{1},
			},
			{
				WalletAddress:    "2_address",
				TemporaryAddress: "PLAYER2_TEMP_ADDRESS",
				HP:               2500,           // 存活，赢家
				Cards:            []int{4, 1, 3}, // Fire, Metal, Water
				LostHP:           1500,           // 倍率3
				Commitment:       []byte{1},
			},
			{
				WalletAddress:    "3_address",
				TemporaryAddress: "PLAYER3_TEMP_ADDRESS",
				HP:               2000,           // 存活，赢家
				Cards:            []int{3, 4, 2}, // Water, Fire, Wood
				LostHP:           2000,           // 倍率4
				Commitment:       []byte{1},
			},
		},
	}

	result, err := engine.ExecuteRound(input)

	if err != nil {
		t.Errorf("ExecuteRound failed with error: %v", err)
		return
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
				WalletAddress:    "1_address",
				TemporaryAddress: "PLAYER1_TEMP_ADDRESS",
				HP:               2500, // 相同血量
				Cards:            []int{4, 1, 3},
				LostHP:           1000, // 倍率2
				Commitment:       []byte{1},
			},
			{
				WalletAddress:    "2_address",
				TemporaryAddress: "PLAYER2_TEMP_ADDRESS",
				HP:               2500, // 相同血量
				Cards:            []int{1, 3, 4},
				LostHP:           1500, // 倍率3
				Commitment:       []byte{1},
			},
			{
				WalletAddress:    "3_address",
				TemporaryAddress: "PLAYER3_TEMP_ADDRESS",
				HP:               2500, // 相同血量
				Cards:            []int{3, 4, 1},
				LostHP:           2000, // 倍率4
				Commitment:       []byte{1},
			},
		},
	}

	result, _ := engine.ExecuteRound(input)

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
	if config.GameParams.InitialHP == 0 {
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
				Commitment:       []byte{1},
			},
			{
				WalletAddress:    "player2_address",
				TemporaryAddress: "PLAYER2_TEMP_ADDRESS",
				Cards:            []int{1, 2}, // 只有1张卡，离线
				HP:               3000,
				LostHP:           0,
				Commitment:       []byte{1},
			},
			{
				WalletAddress:    "player3_address",
				TemporaryAddress: "PLAYER3_TEMP_ADDRESS",
				Cards:            []int{2}, // 只有1张卡，离线
				HP:               3000,
				LostHP:           4000,
				Commitment:       []byte{1},
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

func TestAllOfflinePlayers(t *testing.T) {
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
				Cards:            []int{1, 3},
				HP:               3000,
				LostHP:           0,
				Commitment:       []byte{1},
			},
			{
				WalletAddress:    "player2_address",
				TemporaryAddress: "PLAYER2_TEMP_ADDRESS",
				Cards:            []int{1}, // 只有1张卡，离线
				HP:               3000,
				LostHP:           0,
				Commitment:       []byte{1},
			},
			{
				WalletAddress:    "player3_address",
				TemporaryAddress: "PLAYER3_TEMP_ADDRESS",
				Cards:            []int{2}, // 只有1张卡，离线
				HP:               3000,
				LostHP:           4000,
				Commitment:       []byte{1},
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

func TestNewOfflinePlayerLogic(t *testing.T) {
	initTestEnv(t)
	prepareCards(t)

	engine := NewBattleEngine()

	// 测试案例1：部分玩家卡牌不足（离线），在线玩家获胜
	input := &RoundInput{
		RoundNumber: 1,
		Players: []PlayerRoundInput{
			{
				WalletAddress:    "1_address",
				TemporaryAddress: "PLAYER1_TEMP_ADDRESS",
				Cards:            []int{1, 2, 3}, // 3张卡，正常
				HP:               3000,
				LostHP:           500, // 有一些失血
				Commitment:       []byte{1},
			},
			{
				WalletAddress:    "2_address",
				TemporaryAddress: "PLAYER2_TEMP_ADDRESS",
				Cards:            []int{1}, // 只有1张卡，会被设为离线
				HP:               2500,
				LostHP:           4000, // 更多失血，倍率更高
				Commitment:       []byte{1},
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

func TestAllPlayersOfflineLogic(t *testing.T) {
	initTestEnv(t)
	prepareCards(t)

	engine := NewBattleEngine()

	// 测试案例2：所有玩家都卡牌不足（离线），按血量判断胜负
	input := &RoundInput{
		RoundNumber: 1,
		Players: []PlayerRoundInput{
			{
				WalletAddress:    "1_address",
				TemporaryAddress: "PLAYER1_TEMP_ADDRESS",
				Cards:            []int{1}, // 只有1张卡
				HP:               3000,     // 血量更高
				LostHP:           500,
				Commitment:       []byte{1},
			},
			{
				WalletAddress:    "2_address",
				TemporaryAddress: "PLAYER2_TEMP_ADDRESS",
				Cards:            []int{}, // 只有1张卡
				HP:               2000,    // 血量更低
				LostHP:           3500,    // 失血更多，倍率更高
				Commitment:       []byte{1},
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
				Commitment:       []byte{1},
			},
			{
				WalletAddress:    "2_address",
				TemporaryAddress: "PLAYER2_TEMP_ADDRESS",
				Cards:            []int{2}, // 只有1张卡
				HP:               3000,     // 血量相同
				LostHP:           1000,     // 失血更多，倍率更高
				Commitment:       []byte{1},
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

// 投降功能测试用例
func TestSurrenderLogic(t *testing.T) {
	initTestEnv(t)
	prepareCards(t)

	engine := NewBattleEngine()

	// 测试案例：一个玩家投降
	input := &RoundInput{
		RoundNumber: 1,
		Players: []PlayerRoundInput{
			{
				WalletAddress:    "1_address",
				TemporaryAddress: "PLAYER1_TEMP_ADDRESS",
				Cards:            []int{},
				HP:               3000,
				LostHP:           500,
				Commitment:       []byte{},
				Surrendered:      true, // 玩家1投降
			},
			{
				WalletAddress:    "1_address",
				TemporaryAddress: "PLAYER2_TEMP_ADDRESS",
				Cards:            []int{},
				HP:               2500,
				LostHP:           4000,
				Commitment:       []byte{},
				Surrendered:      false, // 玩家2不投降
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

	t.Logf("投降测试结果 (JSON):\n%s", string(jsonData))
}

func TestAllPlayersSurrender(t *testing.T) {
	initTestEnv(t)
	prepareCards(t)

	engine := NewBattleEngine()

	// 测试案例：所有玩家都投降
	input := &RoundInput{
		RoundNumber: 1,
		Players: []PlayerRoundInput{
			{
				WalletAddress:    "1_address",
				TemporaryAddress: "PLAYER1_TEMP_ADDRESS",
				Cards:            []int{1, 2, 3},
				HP:               3000,
				LostHP:           500,
				Commitment:       []byte{1},
				Surrendered:      true, // 玩家1投降
			},
			{
				WalletAddress:    "2_address",
				TemporaryAddress: "PLAYER2_TEMP_ADDRESS",
				Cards:            []int{1, 2, 3},
				HP:               2500,
				LostHP:           4000,
				Commitment:       []byte{1},
				Surrendered:      true, // 玩家2也投降
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

	t.Logf("全员投降测试结果 (JSON):\n%s", string(jsonData))
}

func TestSurrenderPriorityOverOffline(t *testing.T) {
	initTestEnv(t)
	prepareCards(t)

	engine := NewBattleEngine()

	// 测试案例：投降优先级高于离线
	input := &RoundInput{
		RoundNumber: 1,
		Players: []PlayerRoundInput{
			{
				WalletAddress:    "1_address",
				TemporaryAddress: "PLAYER1_TEMP_ADDRESS",
				Cards:            []int{1}, // 卡牌不足，会被设为离线
				HP:               3000,
				LostHP:           500,
				Commitment:       []byte{1},
				Surrendered:      true, // 但玩家1投降，优先级更高
			},
			{
				WalletAddress:    "2_address",
				TemporaryAddress: "PLAYER2_TEMP_ADDRESS",
				Cards:            []int{1, 2, 3},
				HP:               2500,
				LostHP:           4000,
				Commitment:       []byte{1},
				Surrendered:      false,
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

	t.Logf("投降优先级测试结果 (JSON):\n%s", string(jsonData))
}

// 三人游戏投降测试用例

func TestThreePlayersOneSurrender(t *testing.T) {
	initTestEnv(t)
	prepareCards(t)

	engine := NewBattleEngine()

	// 测试案例：三人游戏中1人投降
	input := &RoundInput{
		RoundNumber: 1,
		Players: []PlayerRoundInput{
			{
				WalletAddress:    "1_address",
				TemporaryAddress: "PLAYER1_TEMP_ADDRESS",
				Cards:            []int{1, 2, 3},
				HP:               3000,
				LostHP:           2500,
				Commitment:       []byte{1},
				Surrendered:      true, // 玩家1投降
			},
			{
				WalletAddress:    "2_address",
				TemporaryAddress: "PLAYER2_TEMP_ADDRESS",
				Cards:            []int{1, 2, 3},
				HP:               2500,
				LostHP:           1000,
				Commitment:       []byte{1},
				Surrendered:      false, // 玩家2不投降
			},
			{
				WalletAddress:    "3_address",
				TemporaryAddress: "PLAYER3_TEMP_ADDRESS",
				Cards:            []int{1, 2, 3},
				HP:               2000,
				LostHP:           1500,
				Commitment:       []byte{1},
				Surrendered:      false, // 玩家3不投降
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

	t.Logf("三人游戏1人投降测试结果 (JSON):\n%s", string(jsonData))
}

func TestThreePlayersTwoSurrender(t *testing.T) {
	initTestEnv(t)
	prepareCards(t)

	engine := NewBattleEngine()

	// 测试案例：三人游戏中2人投降
	input := &RoundInput{
		RoundNumber: 1,
		Players: []PlayerRoundInput{
			{
				WalletAddress:    "1_address",
				TemporaryAddress: "PLAYER1_TEMP_ADDRESS",
				Cards:            []int{1, 2, 3},
				HP:               3000,
				LostHP:           500,
				Commitment:       []byte{1},
				Surrendered:      true, // 玩家1投降
			},
			{
				WalletAddress:    "2_address",
				TemporaryAddress: "PLAYER2_TEMP_ADDRESS",
				Cards:            []int{1, 2, 3},
				HP:               2500,
				LostHP:           3000,
				Commitment:       []byte{1},
				Surrendered:      true, // 玩家2也投降
			},
			{
				WalletAddress:    "3_address",
				TemporaryAddress: "PLAYER3_TEMP_ADDRESS",
				Cards:            []int{1, 2, 3},
				HP:               2000,
				LostHP:           1500,
				Commitment:       []byte{1},
				Surrendered:      false, // 玩家3不投降，应该获胜
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

	t.Logf("三人游戏2人投降测试结果 (JSON):\n%s", string(jsonData))
}

func TestThreePlayersAllSurrender(t *testing.T) {
	initTestEnv(t)
	prepareCards(t)

	engine := NewBattleEngine()

	// 测试案例：三人游戏中所有人都投降
	input := &RoundInput{
		RoundNumber: 1,
		Players: []PlayerRoundInput{
			{
				WalletAddress:    "1_address",
				TemporaryAddress: "PLAYER1_TEMP_ADDRESS",
				Cards:            []int{1, 2, 3},
				HP:               3000,
				LostHP:           2500,
				Commitment:       []byte{1},
				Surrendered:      true, // 玩家1投降
			},
			{
				WalletAddress:    "2_address",
				TemporaryAddress: "PLAYER2_TEMP_ADDRESS",
				Cards:            []int{1, 2, 3},
				HP:               2500,
				LostHP:           3000,
				Commitment:       []byte{1},
				Surrendered:      true, // 玩家2投降
			},
			{
				WalletAddress:    "3_address",
				TemporaryAddress: "PLAYER3_TEMP_ADDRESS",
				Cards:            []int{1, 2, 3},
				HP:               2000,
				LostHP:           1500,
				Commitment:       []byte{1},
				Surrendered:      true, // 玩家3也投降
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

	t.Logf("三人游戏全员投降测试结果 (JSON):\n%s", string(jsonData))
}

func TestExecuteRoundProtoSurrender(t *testing.T) {
	initTestEnv(t)
	prepareCards(t)

	engine := NewBattleEngine()

	// 测试案例：使用Proto格式测试2人游戏投降
	protoInput := &pb.RoundInput{
		RoundNumber: 1,
		Players: []*pb.PlayerRoundInput{
			{
				WalletAddress:    "player1_address",
				TemporaryAddress: "PLAYER1_TEMP_ADDRESS",
				Cards:            []int32{1, 2, 3},
				HP:               3000,
				LostHP:           8000,
				Commitment:       []byte("dummy"),
				Surrendered:      true, // 玩家1投降
			},
			{
				WalletAddress:    "player2_address",
				TemporaryAddress: "PLAYER2_TEMP_ADDRESS",
				Cards:            []int32{3, 4, 2},
				HP:               2500,
				LostHP:           1500,
				Commitment:       []byte("dummy"),
				Surrendered:      true, // 玩家2不投降，应该获胜
			},
		},
	}

	roundResult, gameResult, err := engine.ExecuteRoundProto(protoInput)
	require.NoError(t, err)
	require.NotNil(t, roundResult)

	// 打印回合结果
	protoJSON := protojson.MarshalOptions{EmitUnpopulated: true, UseProtoNames: true, Indent: "  "}
	rrJSON, _ := protoJSON.Marshal(roundResult)
	t.Logf("RoundResult (JSON):\n%s", string(rrJSON))

	// 打印游戏结果（如果有的话）
	if gameResult != nil {
		grJSON, _ := protoJSON.Marshal(gameResult)
		t.Logf("GameResult (JSON):\n%s", string(grJSON))
	}
}

func TestExecuteRoundProtoThreePlayersSurrender(t *testing.T) {
	initTestEnv(t)
	prepareCards(t)

	engine := NewBattleEngine()

	// 测试案例：使用Proto格式测试3人游戏投降
	protoInput := &pb.RoundInput{
		RoundNumber: 1,
		Players: []*pb.PlayerRoundInput{
			{
				WalletAddress:    "player1_address",
				TemporaryAddress: "PLAYER1_TEMP_ADDRESS",
				Cards:            []int32{1, 2, 3},
				HP:               3000,
				LostHP:           2000,
				Commitment:       []byte("dummy"),
				Surrendered:      true, // 玩家1投降
			},
			{
				WalletAddress:    "player2_address",
				TemporaryAddress: "PLAYER2_TEMP_ADDRESS",
				Cards:            []int32{3, 4, 2},
				HP:               2500,
				LostHP:           1500,
				Commitment:       []byte("dummy"),
				Surrendered:      false, // 玩家2不投降
			},
			{
				WalletAddress:    "player3_address",
				TemporaryAddress: "PLAYER3_TEMP_ADDRESS",
				Cards:            []int32{2, 1, 4},
				HP:               2000,
				LostHP:           1000,
				Commitment:       []byte("dummy"),
				Surrendered:      false, // 玩家3不投降
			},
		},
	}

	roundResult, gameResult, err := engine.ExecuteRoundProto(protoInput)
	require.NoError(t, err)
	require.NotNil(t, roundResult)

	// 打印回合结果
	protoJSON := protojson.MarshalOptions{EmitUnpopulated: true, UseProtoNames: true, Indent: "  "}
	rrJSON, _ := protoJSON.Marshal(roundResult)
	t.Logf("RoundResult (JSON):\n%s", string(rrJSON))

	// 打印游戏结果（如果有的话）
	if gameResult != nil {
		grJSON, _ := protoJSON.Marshal(gameResult)
		t.Logf("GameResult (JSON):\n%s", string(grJSON))
	}
}

// 测试服务器超时导致的回合结束 - ROUND_COMPLETE_SERVER_CHAIN_TIMEOUT
func TestExecuteRoundServerChainTimeout(t *testing.T) {
	initTestEnv(t)
	prepareCards(t)

	engine := NewBattleEngine()

	// 测试案例：服务器链上超时
	input := &RoundInput{
		RoundNumber: 2,
		Reason:      pb.RoundCompleteReason_ROUND_COMPLETE_SERVER_CHAIN_TIMEOUT,
		Players: []PlayerRoundInput{
			{
				WalletAddress:    "player1_address",
				TemporaryAddress: "PLAYER1_TEMP_ADDRESS",
				Cards:            []int{1, 2, 3},
				HP:               3000,
				LostHP:           500,
				Commitment:       []byte{1},
				Surrendered:      false,
			},
			{
				WalletAddress:    "player2_address",
				TemporaryAddress: "PLAYER2_TEMP_ADDRESS",
				Cards:            []int{3, 4, 2},
				HP:               2500,
				LostHP:           1000,
				Commitment:       []byte{1},
				Surrendered:      false,
			},
		},
	}

	result, err := engine.ExecuteRound(input)
	require.NoError(t, err)
	require.NotNil(t, result)

	// 打印结果
	jsonData, err := json.MarshalIndent(result, "", "  ")
	require.NoError(t, err)
	t.Logf("服务器链上超时测试结果 (JSON):\n%s", string(jsonData))
}

// 测试服务器内部超时导致的回合结束 - ROUND_COMPLETE_SERVER_INTERNAL_TIMEOUT
func TestExecuteRoundServerInternalTimeout(t *testing.T) {
	initTestEnv(t)
	prepareCards(t)

	engine := NewBattleEngine()

	// 测试案例：服务器内部超时
	input := &RoundInput{
		RoundNumber: 1,
		Reason:      pb.RoundCompleteReason_ROUND_COMPLETE_SERVER_INTERNAL_TIMEOUT,
		Players: []PlayerRoundInput{
			{
				WalletAddress:    "player1_address",
				TemporaryAddress: "PLAYER1_TEMP_ADDRESS",
				Cards:            []int{1, 2, 3},
				HP:               3000,
				LostHP:           500,
				Commitment:       []byte{1},
				Surrendered:      false,
			},
			{
				WalletAddress:    "player2_address",
				TemporaryAddress: "PLAYER2_TEMP_ADDRESS",
				Cards:            []int{3, 4, 2},
				HP:               2500,
				LostHP:           1000,
				Commitment:       []byte{1},
				Surrendered:      false,
			},
			{
				WalletAddress:    "player3_address",
				TemporaryAddress: "PLAYER3_TEMP_ADDRESS",
				Cards:            []int{2, 1, 4},
				HP:               2000,
				LostHP:           1500,
				Commitment:       []byte{1},
				Surrendered:      false,
			},
		},
	}

	result, err := engine.ExecuteRound(input)
	require.NoError(t, err)
	require.NotNil(t, result)

	// 打印结果
	jsonData, err := json.MarshalIndent(result, "", "  ")
	require.NoError(t, err)
	t.Logf("服务器内部超时测试结果 (JSON):\n%s", string(jsonData))
}

// 测试使用Proto格式的服务器超时
func TestExecuteRoundProtoServerTimeout(t *testing.T) {
	initTestEnv(t)
	prepareCards(t)

	engine := NewBattleEngine()

	// 测试案例：使用Proto格式测试服务器链上超时
	protoInput := &pb.RoundInput{
		RoundNumber: 2,
		Reason:      pb.RoundCompleteReason_ROUND_COMPLETE_SERVER_CHAIN_TIMEOUT,
		Players: []*pb.PlayerRoundInput{
			{
				WalletAddress:    "player1_address",
				TemporaryAddress: "PLAYER1_TEMP_ADDRESS",
				Cards:            []int32{1, 2, 3},
				HP:               3000,
				LostHP:           500,
				Commitment:       []byte("dummy"),
				Surrendered:      false,
			},
			{
				WalletAddress:    "player2_address",
				TemporaryAddress: "PLAYER2_TEMP_ADDRESS",
				Cards:            []int32{3, 4, 2},
				HP:               2500,
				LostHP:           1000,
				Commitment:       []byte("dummy"),
				Surrendered:      false,
			},
		},
	}

	roundResult, gameResult, err := engine.ExecuteRoundProto(protoInput)
	require.NoError(t, err)

	// 打印结果
	protoJSON := protojson.MarshalOptions{EmitUnpopulated: true, UseProtoNames: true, Indent: "  "}
	rrJSON, _ := protoJSON.Marshal(roundResult)
	t.Logf("Proto服务器超时测试结果 - RoundResult (JSON):\n%s", string(rrJSON))

	grJSON, _ := protoJSON.Marshal(gameResult)
	t.Logf("Proto服务器超时测试结果 - GameResult (JSON):\n%s", string(grJSON))
}

// go test -v ./room_server/battle/ -run TestExecuteRoundNormal
// go test -v ./room_server/battle/ -run "^TestExecuteRoundProto$"
//go test -v ./room_server/battle/ -run "^TestExecuteRoundProtoFromFile$" | tee test/api/battle/test.log

//go test -v ./room_server/battle/ -run TestExecuteRoundServerChainTimeout
