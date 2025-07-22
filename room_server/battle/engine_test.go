package battle

import (
	"encoding/json"
	"testing"

	"github.com/CryptoElementals/common/db"
	"github.com/CryptoElementals/common/log"
	dao "github.com/CryptoElementals/common/models"
	"github.com/stretchr/testify/require"
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
		Development: true,
	}

	err = db.Init(dbConfig)
	if err != nil {
		t.Fatalf("Failed to initialize database: %v", err)
	}
	require.NoError(t, db.MigrateMemDb())

	cards := []dao.Card{
		{
			CardID:             1,
			ElementType:        "Metal",
			Level:              "normal",
			LifeForce:          500,
			Attack:             1000,
			Defense:            500,
			NormalImageURL:     "https://example.com/normal_image.png",
			ActiveImageURL:     "https://example.com/active_image.png",
			BackgroundImageURL: "https://example.com/background_image.png",
			IconURL:            "https://example.com/icon.png",
			Name:               "Metal Element",
			Description:        "",
		},
		{
			CardID:             2,
			ElementType:        "Wood",
			Level:              "normal",
			LifeForce:          500,
			Attack:             1000,
			Defense:            500,
			NormalImageURL:     "https://example.com/normal_image.png",
			ActiveImageURL:     "https://example.com/active_image.png",
			BackgroundImageURL: "https://example.com/background_image.png",
			IconURL:            "https://example.com/icon.png",
			Name:               "Metal Element",
			Description:        "",
		},
		{
			CardID:             3,
			ElementType:        "Water",
			Level:              "normal",
			LifeForce:          500,
			Attack:             1000,
			Defense:            500,
			NormalImageURL:     "https://example.com/normal_image.png",
			ActiveImageURL:     "https://example.com/active_image.png",
			BackgroundImageURL: "https://example.com/background_image.png",
			IconURL:            "https://example.com/icon.png",
			Name:               "Metal Element",
			Description:        "",
		},
		{
			CardID:             4,
			ElementType:        "Fire",
			Level:              "normal",
			LifeForce:          500,
			Attack:             1000,
			Defense:            500,
			NormalImageURL:     "https://example.com/normal_image.png",
			ActiveImageURL:     "https://example.com/active_image.png",
			BackgroundImageURL: "https://example.com/background_image.png",
			IconURL:            "https://example.com/icon.png",
			Name:               "Metal Element",
			Description:        "",
		},
		{
			CardID:             5,
			ElementType:        "Earth",
			Level:              "normal",
			LifeForce:          500,
			Attack:             1000,
			Defense:            500,
			NormalImageURL:     "https://example.com/normal_image.png",
			ActiveImageURL:     "https://example.com/active_image.png",
			BackgroundImageURL: "https://example.com/background_image.png",
			IconURL:            "https://example.com/icon.png",
			Name:               "Metal Element",
			Description:        "",
		},
	}
	require.NoError(t, db.Get().Save(&cards).Error)

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
				HP:         3000,
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

// go test -v ./room_server/battle/ -run TestExecuteRound
