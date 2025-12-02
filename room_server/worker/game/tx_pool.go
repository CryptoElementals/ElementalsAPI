package game

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/CryptoElementals/common/log"
	dao "github.com/CryptoElementals/common/models"
	"github.com/CryptoElementals/common/room_server/worker/types"
)

// eventKey uniquely identifies an event by game ID, address, round number, and index
type eventKey struct {
	gameID      uint
	address     string
	roundNumber uint32
	index       uint32
}

type txPool struct {
	// Event pools for commitment and card submissions
	commitmentPool map[eventKey]*types.SubmitPlayerCommitment
	cardPool       map[eventKey]*types.SubmitPlayerCard
	poolLock       sync.RWMutex

	// Track max received turn indices per round per player per game
	// Structure: gameID -> playerAddress -> roundNumber -> maxTurnIndex
	gameTxInfos map[uint]*gameTxInfo
	txInfoLock  sync.RWMutex

	// Dependencies
	chainSvc ContractClient
}

type gameTxInfo struct {
	playerTxInfos map[types.PlayerAddress]*gameTxPlayerInfo
}

type gameTxPlayerInfo struct {
	// Map of roundNumber -> max received commitment turn index for that round
	commitmentTurnIndices map[uint32]int32
	// Map of roundNumber -> max received card turn index for that round
	cardTurnIndices map[uint32]int32
}

// newTxPool creates a new transaction pool
func newTxPool(chainSvc ContractClient) *txPool {
	return &txPool{
		commitmentPool: make(map[eventKey]*types.SubmitPlayerCommitment),
		cardPool:       make(map[eventKey]*types.SubmitPlayerCard),
		gameTxInfos:    make(map[uint]*gameTxInfo),
		chainSvc:       chainSvc,
	}
}

// makeEventKey creates an eventKey from game ID, address, round number, and index
func makeEventKey(gameID uint, address types.PlayerAddress, roundNumber, index uint32) eventKey {
	return eventKey{
		gameID:      gameID,
		address:     address.TemporaryAddress,
		roundNumber: roundNumber,
		index:       index,
	}
}

// getOrCreatePlayerInfo gets or creates player transaction info
func (p *txPool) getOrCreatePlayerInfo(gameID uint, address types.PlayerAddress) *gameTxPlayerInfo {
	gameInfo, exists := p.gameTxInfos[gameID]
	if !exists {
		gameInfo = &gameTxInfo{
			playerTxInfos: make(map[types.PlayerAddress]*gameTxPlayerInfo),
		}
		p.gameTxInfos[gameID] = gameInfo
	}

	playerInfo, exists := gameInfo.playerTxInfos[address]
	if !exists {
		playerInfo = &gameTxPlayerInfo{
			commitmentTurnIndices: make(map[uint32]int32),
			cardTurnIndices:       make(map[uint32]int32),
		}
		gameInfo.playerTxInfos[address] = playerInfo
	}
	return playerInfo
}

// addCommitment adds a commitment to the pool after validating turn index
func (p *txPool) addCommitment(evt *types.SubmitPlayerCommitment) error {
	p.txInfoLock.Lock()
	playerInfo := p.getOrCreatePlayerInfo(evt.GameID, evt.Address)

	// Get max received turn index for this round
	maxTurnIndex := playerInfo.commitmentTurnIndices[evt.RoundNumber]

	// Reject if turn index is <= max received for this round
	if int32(evt.CommitmentIndex) <= maxTurnIndex {
		p.txInfoLock.Unlock()
		return fmt.Errorf("commitment with round %d, turn index %d rejected: already received index %d or higher for this round", evt.RoundNumber, evt.CommitmentIndex, maxTurnIndex)
	}

	// Update max received index for this round
	playerInfo.commitmentTurnIndices[evt.RoundNumber] = int32(evt.CommitmentIndex)
	p.txInfoLock.Unlock()

	// Add to pool
	key := makeEventKey(evt.GameID, evt.Address, evt.RoundNumber, evt.CommitmentIndex)
	p.poolLock.Lock()
	defer p.poolLock.Unlock()

	if _, exists := p.commitmentPool[key]; exists {
		return fmt.Errorf("commitment already in pool")
	}

	p.commitmentPool[key] = evt
	return nil
}

// addCard adds a card to the pool after validating turn index
func (p *txPool) addCard(evt *types.SubmitPlayerCard) error {
	p.txInfoLock.Lock()
	playerInfo := p.getOrCreatePlayerInfo(evt.GameID, evt.Address)

	// Get max received turn index for this round
	maxTurnIndex := playerInfo.cardTurnIndices[evt.RoundNumber]

	// Reject if turn index is <= max received for this round
	if int32(evt.CardIndex) <= maxTurnIndex {
		p.txInfoLock.Unlock()
		return fmt.Errorf("card with round %d, turn index %d rejected: already received index %d or higher for this round", evt.RoundNumber, evt.CardIndex, maxTurnIndex)
	}

	// Update max received index for this round
	playerInfo.cardTurnIndices[evt.RoundNumber] = int32(evt.CardIndex)
	p.txInfoLock.Unlock()

	// Add to pool
	key := makeEventKey(evt.GameID, evt.Address, evt.RoundNumber, evt.CardIndex)
	p.poolLock.Lock()
	defer p.poolLock.Unlock()

	if _, exists := p.cardPool[key]; exists {
		return fmt.Errorf("card already in pool")
	}

	p.cardPool[key] = evt
	return nil
}

// processPools periodically processes events in the pools and sends them to chain manager
func (p *txPool) processPools(ctx context.Context, wg *sync.WaitGroup, args dao.GameArgs) {
	defer wg.Done()
	ticker := time.NewTicker(time.Duration(args.PoolProcessingInterval) * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			p.processCommitmentPool()
			p.processCardPool()
		}
	}
}

// processCommitmentPool processes commitment pool and sends events to chain manager in batch
func (p *txPool) processCommitmentPool() {
	// Collect events to process while holding lock
	p.poolLock.Lock()
	batchEvents := make([]*types.SubmitPlayerCommitment, 0, len(p.commitmentPool))
	for _, evt := range p.commitmentPool {
		if evt.GameID != 0 {
			batchEvents = append(batchEvents, evt)
		} else {
			log.Errorw("commitment event missing GameID", "address", evt.Address.TemporaryAddress)
		}
	}
	p.poolLock.Unlock()

	if len(batchEvents) == 0 {
		return
	}

	// Submit batch
	if err := p.chainSvc.SubmitPlayerCommitmentsBatch(batchEvents); err != nil {
		log.Errorw("failed to submit commitments batch to chain", "error", err, "count", len(batchEvents))
		return
	}

	// Remove successfully submitted events from pool to avoid duplicate submissions
	p.poolLock.Lock()
	for _, evt := range batchEvents {
		key := makeEventKey(evt.GameID, evt.Address, evt.RoundNumber, evt.CommitmentIndex)
		delete(p.commitmentPool, key)
	}
	p.poolLock.Unlock()

	log.Infow("submitted commitments batch to chain", "count", len(batchEvents))
}

// processCardPool processes card pool and sends events to chain manager in batch
func (p *txPool) processCardPool() {
	// Collect events to process while holding lock
	p.poolLock.Lock()
	batchEvents := make([]*types.SubmitPlayerCard, 0, len(p.cardPool))
	for _, evt := range p.cardPool {
		if evt.GameID != 0 {
			batchEvents = append(batchEvents, evt)
		} else {
			log.Errorw("card event missing GameID", "address", evt.Address.TemporaryAddress)
		}
	}
	p.poolLock.Unlock()

	if len(batchEvents) == 0 {
		return
	}

	// Submit batch
	if err := p.chainSvc.SubmitPlayerCardsBatch(batchEvents); err != nil {
		log.Errorw("failed to submit cards batch to chain", "error", err, "count", len(batchEvents))
		return
	}

	// Remove successfully submitted events from pool to avoid duplicate submissions
	p.poolLock.Lock()
	for _, evt := range batchEvents {
		key := makeEventKey(evt.GameID, evt.Address, evt.RoundNumber, evt.CardIndex)
		delete(p.cardPool, key)
	}
	p.poolLock.Unlock()

	log.Infow("submitted cards batch to chain", "count", len(batchEvents))
}

// clearGameInfo clears transaction info for a completed game
func (p *txPool) clearGameInfo(gameID uint) {
	p.txInfoLock.Lock()
	defer p.txInfoLock.Unlock()
	delete(p.gameTxInfos, gameID)
	log.Infow("cleared transaction info for game", "gameID", gameID)
}
