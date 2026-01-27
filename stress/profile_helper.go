package stress

import (
	"fmt"
	"math/rand"
	"time"
)

// RandomNameGenerator generates random names for bots
type RandomNameGenerator struct {
	rng *rand.Rand
}

// NewRandomNameGenerator creates a new random name generator
func NewRandomNameGenerator() *RandomNameGenerator {
	return &RandomNameGenerator{
		rng: rand.New(rand.NewSource(time.Now().UnixNano())),
	}
}

// GenerateName generates a random bot name
func (g *RandomNameGenerator) GenerateName(botID int) string {
	// Generate a random name like "Bot_12345" or "Player_67890"
	randomNum := g.rng.Intn(99999) + 10000
	return fmt.Sprintf("Bot_%d_%d", botID, randomNum)
}

// GenerateAvatar generates a random avatar filename
// Avatar filenames follow pattern: avatar_*.png
func (g *RandomNameGenerator) GenerateAvatar() string {
	// Common avatar patterns - you can adjust these based on actual avatar filenames
	avatars := []string{
		"avatar_1.png",
		"avatar_2.png",
		"avatar_3.png",
		"avatar_4.png",
		"avatar_5.png",
	}
	return avatars[g.rng.Intn(len(avatars))]
}
