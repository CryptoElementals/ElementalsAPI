package stress

import (
	"math/rand"
	"time"

	gameclient "github.com/CryptoElementals/common/game_client"
)

// RandomCardProvider provides random cards for stress testing
type RandomCardProvider struct {
	rng *rand.Rand
}

// NewRandomCardProvider creates a new random card provider
func NewRandomCardProvider() gameclient.CardProvider {
	return &RandomCardProvider{
		rng: rand.New(rand.NewSource(time.Now().UnixNano())),
	}
}

// GetCard returns a random card (1-5) for the given round and turn
func (p *RandomCardProvider) GetCard(ctx gameclient.CardPickContext) (uint32, error) {
	_ = ctx
	return uint32(p.rng.Intn(5) + 1), nil
}
