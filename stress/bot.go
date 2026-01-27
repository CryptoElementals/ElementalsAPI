package stress

import (
	"context"
	"fmt"
	"time"

	"github.com/CryptoElementals/common/log"
	gameclient "github.com/CryptoElementals/common/game_client"
	"github.com/CryptoElementals/common/wallet"
)

// Bot represents a single stress test bot
type Bot struct {
	id           int
	playerID     string // Player ID fetched from API
	wallet       *wallet.Wallet
	gameContext  *gameclient.GameContextHTTP
	ctx          context.Context
	cancel       context.CancelFunc
	refreshToken string
}

// NewBot creates a new bot instance
func NewBot(
	id int,
	w *wallet.Wallet,
	cardProvider gameclient.CardProvider,
	baseURL string,
) (*Bot, error) {
	ctx, cancel := context.WithCancel(context.Background())

	// Use 0 as placeholder player ID, will be fetched from API after login
	gameCtx, err := gameclient.NewGameContextHTTP(
		ctx,
		baseURL,
		0,
		w,
		cardProvider,
	)
	if err != nil {
		cancel()
		return nil, fmt.Errorf("failed to create game context: %w", err)
	}

	return &Bot{
		id:          id,
		playerID:    "", // Will be set after login
		wallet:      w,
		gameContext: gameCtx,
		ctx:         ctx,
		cancel:      cancel,
	}, nil
}

// Start starts the bot's game loop
func (b *Bot) Start() error {
	// Step 1: Sign in (this will fetch player ID from API)
	if err := b.gameContext.SignIn(); err != nil {
		return fmt.Errorf("bot %d sign in failed: %w", b.id, err)
	}

	// Get player ID and refresh token from game context
	b.playerID = b.gameContext.GetPlayerID()
	b.refreshToken = b.gameContext.GetRefreshToken()

	// Step 2: Get or set user profile
	if err := b.setupUserProfile(); err != nil {
		log.Errorw("failed to setup user profile", "bot_id", b.id, "player_id", b.playerID, "error", err)
		// Continue even if profile setup fails
	}

	// Step 3: Check and collect daily reward
	if err := b.handleDailyReward(); err != nil {
		log.Errorw("failed to handle daily reward", "bot_id", b.id, "player_id", b.playerID, "error", err)
		// Continue even if daily reward fails
	}

	// Step 4: Subscribe to game events
	if err := b.gameContext.Subscribe(); err != nil {
		return fmt.Errorf("bot %d subscribe failed: %w", b.id, err)
	}

	// Step 5: Start game loop
	go b.gameLoop()

	return nil
}

// setupUserProfile gets user profile or sets it with random values if it doesn't exist
func (b *Bot) setupUserProfile() error {
	apiClient := b.gameContext.GetApiClient()

	// Try to get user profile
	_, err := apiClient.GetUserProfile(b.playerID)
	if err == nil {
		// Profile exists, no need to set
		return nil
	}

	// Profile doesn't exist or error occurred, set it with random values
	nameGen := NewRandomNameGenerator()
	name := nameGen.GenerateName(b.id)
	avatar := nameGen.GenerateAvatar()

	if err := apiClient.SetUserProfile(b.playerID, name, avatar); err != nil {
		return fmt.Errorf("failed to set user profile: %w", err)
	}

	log.Infow("set user profile", "bot_id", b.id, "player_id", b.playerID, "name", name, "avatar", avatar)
	return nil
}

// handleDailyReward checks if daily reward is collected and collects it if not
func (b *Bot) handleDailyReward() error {
	apiClient := b.gameContext.GetApiClient()

	// Check if daily reward has been collected
	collected, err := apiClient.HasCollectedDailyReward(b.playerID)
	if err != nil {
		return fmt.Errorf("failed to check daily reward: %w", err)
	}

	if collected {
		// Already collected, skip
		return nil
	}

	// Collect daily reward
	if err := apiClient.CollectDailyReward(b.playerID); err != nil {
		return fmt.Errorf("failed to collect daily reward: %w", err)
	}

	log.Infow("collected daily reward", "bot_id", b.id, "player_id", b.playerID)
	return nil
}

// GetRefreshToken returns the refresh token
func (b *Bot) GetRefreshToken() string {
	return b.refreshToken
}

// gameLoop runs the bot's game loop: join queue -> play game -> repeat
func (b *Bot) gameLoop() {
	for {
		select {
		case <-b.ctx.Done():
			return
		default:
			// Join queue
			if err := b.gameContext.JoinQueue(); err != nil {
				log.Errorw("bot failed to join queue", "bot_id", b.id, "player_id", b.playerID, "error", err)
				time.Sleep(5 * time.Second)
				continue
			}

			// Play the game (Run will block until game completes)
			if err := b.gameContext.Run(); err != nil {
				log.Errorw("bot game run failed", "bot_id", b.id, "player_id", b.playerID, "error", err)
			}

			// Wait a bit before joining queue again
			time.Sleep(2 * time.Second)
		}
	}
}

// Stop stops the bot
func (b *Bot) Stop() {
	b.cancel()
	b.gameContext.Close()
}

// GetID returns the bot ID
func (b *Bot) GetID() int {
	return b.id
}

// GetPlayerID returns the player ID
func (b *Bot) GetPlayerID() string {
	return b.playerID
}

// GetAddress returns the wallet address
func (b *Bot) GetAddress() string {
	return b.wallet.GetAddrHex()
}
