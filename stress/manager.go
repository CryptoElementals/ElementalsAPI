package stress

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/CryptoElementals/common/log"
)

// Manager manages multiple stress test bots
type Manager struct {
	config         *Config
	botInfoManager *BotInfoManager
	bots           []*Bot
	ctx            context.Context
	cancel         context.CancelFunc
	mu             sync.RWMutex
}

// NewManager creates a new stress test manager
func NewManager(config *Config) (*Manager, error) {
	botInfoManager, err := NewBotInfoManager(config.BotInfoCSV)
	if err != nil {
		return nil, fmt.Errorf("failed to create bot info manager: %w", err)
	}

	ctx, cancel := context.WithCancel(context.Background())

	return &Manager{
		config:         config,
		botInfoManager: botInfoManager,
		bots:           make([]*Bot, 0, config.NumBots),
		ctx:            ctx,
		cancel:         cancel,
	}, nil
}

// Start starts all bots
func (m *Manager) Start() error {
	// Get or create wallets from CSV (this will create and save new wallets if needed)
	wallets, err := m.botInfoManager.GetOrCreateWallets(m.config.NumBots)
	if err != nil {
		return fmt.Errorf("failed to get/create wallets: %w", err)
	}

	// Create card provider (using random for variety)
	cardProvider := NewRandomCardProvider()

	// Create and start all bots
	var startedBots []*Bot
	for i := 0; i < m.config.NumBots; i++ {
		w := wallets[i]

		// Create bot (player ID will be fetched after login)
		bot, err := NewBot(i, w, cardProvider, m.config.BaseURL)
		if err != nil {
			return fmt.Errorf("failed to create bot %d: %w", i, err)
		}

		// Start bot (this will login and fetch player ID)
		if err := bot.Start(); err != nil {
			log.Errorw("failed to start bot", "bot_id", i, "error", err)
			bot.Stop()
			continue
		}

		// Update refresh token in CSV after login
		if err := m.botInfoManager.UpdateRefreshToken(i, bot.GetRefreshToken()); err != nil {
			log.Errorw("failed to update refresh token", "bot_id", i, "error", err)
		}

		startedBots = append(startedBots, bot)

		// Small delay between bot starts to avoid overwhelming the server
		if i < m.config.NumBots-1 {
			select {
			case <-m.ctx.Done():
				return nil
			case <-time.After(100 * time.Millisecond):
			}
		}
	}

	// Batch update bots slice
	m.mu.Lock()
	m.bots = append(m.bots, startedBots...)
	m.mu.Unlock()

	return nil
}

// Stop stops all bots
func (m *Manager) Stop() {
	m.cancel()

	m.mu.RLock()
	bots := make([]*Bot, len(m.bots))
	copy(bots, m.bots)
	m.mu.RUnlock()

	for _, bot := range bots {
		bot.Stop()
	}
}
