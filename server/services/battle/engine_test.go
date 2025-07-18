package battle

import (
	"encoding/json"
	"testing"

	"github.com/CryptoElementals/common/db"
	"github.com/CryptoElementals/common/log"
)

func TestExecuteBattle(t *testing.T) {
	err := log.InitGlobalLogger(&log.Config{
		Development: true,
		Level:       "debug",
	})
	if err != nil {
		t.Fatalf("Failed to initialize logger: %v", err)
	}

	dbConfig := &db.Config{
		Endpoint: "localhost:3306",
		User:     "root",
		Password: "beastroyale123",
		DbName:   "beast_royale",
	}

	err = db.Init(dbConfig)
	if err != nil {
		t.Fatalf("Failed to initialize database: %v", err)
	}

	engine := NewBattleEngine()

	input := &BattleInput{
		Player1Address:    "player1_address",
		Player2Address:    "player2_address",
		Player1HP:         3000,
		Player2HP:         3000,
		Player1Multiplier: 1.0,
		Player2Multiplier: 1.0,
		Player1Cards:      []int{1, 2, 3},
		Player2Cards:      []int{4, 5, 1},
		Player1LostHP:     0,
		Player2LostHP:     0,
	}

	round := uint(3)
	result, err := engine.ExecuteBattle(input, round)

	if err != nil {
		t.Errorf("ExecuteBattle failed with error: %v", err)
		return
	}

	if result == nil {
		t.Error("ExecuteBattle returned nil result")
		return
	}

	jsonData, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		t.Errorf("Failed to marshal result to JSON: %v", err)
		return
	}

	t.Logf("Battle Result (JSON):\n%s", string(jsonData))
	t.Log("Test completed successfully - check the JSON output above for manual verification")
}

//cd /data/ws_tj/BeastRoyaleBackend && go test -v ./server/services/battle/ -run TestExecuteBattle
