package battle

import (
	"encoding/json"
	"testing"

	"github.com/CryptoElementals/common/db"
	"github.com/CryptoElementals/common/log"
	dao "github.com/CryptoElementals/common/models"
	pb "github.com/CryptoElementals/common/rpc/proto"
	"github.com/stretchr/testify/require"
)

func TestExecuteRound(t *testing.T) {
	initTestEnv(t)

	prepareCards(t)

	engine := NewBattleEngine()

	input := &RoundInput{
		Round: 3,
		Players: []PlayerRoundInput{
			{
				Address:    "player1_address",
				HP:         3000,
				Multiplier: 1.0,
				Cards:      []int{4, 5, 3},
				LostHP:     0,
			},
			{
				Address:    "player2_address",
				HP:         3000,
				Multiplier: 1.0,
				Cards:      []int{1, 2, 4},
				LostHP:     0,
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

// TestExecuteRoundProto 使用 proto 消息调用并验证结果
func TestExecuteRoundProto(t *testing.T) {
	initTestEnv(t)

	// 准备卡牌数据
	prepareCards(t)

	engine := NewBattleEngine()

	protoInput := &pb.RoundInput{
		RoundNumber: 3,
		Players: []*pb.PlayerRoundInput{
			{
				WalletAddress:    "player1_address",
				TemporaryAddress: "",
				Cards:            []int32{4, 5, 3},
				HP:               3000,
				LostHP:           0,
			},
			{
				WalletAddress:    "player2_address",
				TemporaryAddress: "",
				Cards:            []int32{1, 2, 4},
				HP:               3000,
				LostHP:           0,
			},
		},
	}

	roundResult, gameResult, err := engine.ExecuteRoundProto(protoInput)
	require.NoError(t, err)
	require.NotNil(t, roundResult)

	// 打印结果，方便调试
	if data, _ := json.MarshalIndent(roundResult, "", "  "); len(data) > 0 {
		t.Logf("RoundResult:\n%s", string(data))
	}
	if gameResult != nil {
		if data, _ := json.MarshalIndent(gameResult, "", "  "); len(data) > 0 {
			t.Logf("GameResult:\n%s", string(data))
		}
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
}

func prepareCards(t *testing.T) {
	t.Helper()
	cards := []dao.Card{
		{CardID: 1, ElementType: "Metal", Level: "normal", LifeForce: 500, Attack: 1000, Defense: 500},
		{CardID: 2, ElementType: "Wood", Level: "normal", LifeForce: 500, Attack: 1000, Defense: 500},
		{CardID: 3, ElementType: "Water", Level: "normal", LifeForce: 500, Attack: 1000, Defense: 500},
		{CardID: 4, ElementType: "Fire", Level: "normal", LifeForce: 500, Attack: 1000, Defense: 500},
		{CardID: 5, ElementType: "Earth", Level: "normal", LifeForce: 500, Attack: 1000, Defense: 500},
	}
	require.NoError(t, db.Get().Save(&cards).Error)
}

// go test -v ./room_server/battle/ -run TestExecuteRound
// go test -v ./room_server/battle/ -run TestExecuteRoundProto
