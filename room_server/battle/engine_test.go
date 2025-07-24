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
	"github.com/stretchr/testify/require"

	// 确保导入GetBattleInfoResponse和相关结构体
	"github.com/CryptoElementals/common/server/api/battle"
)

func TestExecuteRoundNormal(t *testing.T) {
	initTestEnv(t)

	prepareCards(t)

	engine := NewBattleEngine()

	input := &RoundInput{
		RoundNumber: 3,
		Players: []PlayerRoundInput{
			{
				WalletAddress:    "player1_address",
				TemporaryAddress: "PLAYER1_TEMP_ADDRESS",
				HP:               3000,
				Cards:            []int{4, 1, 3},
				LostHP:           500,
			},
			{
				WalletAddress:    "player2_address",
				TemporaryAddress: "PLAYER2_TEMP_ADDRESS",
				HP:               3000,
				Cards:            []int{1, 3, 4},
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

// go test -v ./room_server/battle/ -run TestExecuteRoundNormal
// go test -v ./room_server/battle/ -run TestExecuteRoundProto
//go test -v ./room_server/battle/ -run TestExecuteRoundProtoFromFile | tee test/api/battle/test_output.log
