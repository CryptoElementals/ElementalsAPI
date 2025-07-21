package battle

import (
	"encoding/json"
	"testing"

	"github.com/CryptoElementals/common/db"
	"github.com/CryptoElementals/common/log"
)

func TestExecuteRound(t *testing.T) {
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

	input := &RoundInput{
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
				HP:         2000,
				Multiplier: 1.0,
				Cards:      []int{1, 2, 4},
				LostHP:     0,
			},
		},
	}

	round := uint(3)
	result, err := engine.ExecuteRound(input, round)

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

//cd /data/ws_tj/BeastRoyaleBackend && go test -v ./server/services/battle/ -run TestExecuteRound
