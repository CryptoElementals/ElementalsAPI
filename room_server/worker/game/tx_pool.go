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

const defaultPoolBatchSize = 10

// eventKey uniquely identifies an event by game ID, address, round number, and index
type eventKey struct {
	gameID      uint
	address     string
	roundNumber uint32
	index       uint32
}

// setTurnKey uniquely identifies a set-turn-ready event
type setTurnKey struct {
	gameID      uint
	roundNumber uint32
	turnNumber  uint32
}

type txPool struct {
	// Event pools for commitment and card submissions
	commitmentPool map[eventKey]*types.SubmitPlayerCommitment
	cardPool       map[eventKey]*types.SubmitPlayerCard
	// Pools for create room and set turn ready (same ticker as above)
	createRoomPool   map[uint]*types.RequireGameCreationEvent
	setTurnReadyPool map[setTurnKey]*types.RequireSetupNewTurnEvent
	poolLock         sync.RWMutex

	// Track max received turn indices per round per player per game
	// Structure: gameID -> playerAddress -> roundNumber -> maxTurnIndex
	gameTxInfos map[uint]*gameTxInfo
	txInfoLock  sync.RWMutex

	// Dependencies
	chainSvc  ContractClient
	batchSize int
}

type gameTxInfo struct {
	playerTxInfos map[types.PlayerAddress]*gameTxPlayerInfo
}

type gameTxPlayerInfo struct {
	commitmentTurnIndices map[uint32]int32
	cardTurnIndices       map[uint32]int32
}

// newTxPool creates a new transaction pool
func newTxPool(chainSvc ContractClient, batchSize int) *txPool {
	if batchSize <= 0 {
		batchSize = defaultPoolBatchSize
	}
	return &txPool{
		commitmentPool:   make(map[eventKey]*types.SubmitPlayerCommitment),
		cardPool:         make(map[eventKey]*types.SubmitPlayerCard),
		createRoomPool:   make(map[uint]*types.RequireGameCreationEvent),
		setTurnReadyPool: make(map[setTurnKey]*types.RequireSetupNewTurnEvent),
		gameTxInfos:      make(map[uint]*gameTxInfo),
		chainSvc:         chainSvc,
		batchSize:        batchSize,
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

// addCreateRoom enqueues a create-room event (one per game; overwrites if same game).
func (p *txPool) addCreateRoom(evt *types.RequireGameCreationEvent) {
	p.poolLock.Lock()
	defer p.poolLock.Unlock()
	p.createRoomPool[evt.GameID] = evt
}

// addSetTurnReady enqueues a set-turn-ready event (one per game/round/turn).
func (p *txPool) addSetTurnReady(evt *types.RequireSetupNewTurnEvent) {
	key := setTurnKey{gameID: evt.GameID, roundNumber: evt.RoundNumber, turnNumber: evt.TurnNumber}
	p.poolLock.Lock()
	defer p.poolLock.Unlock()
	p.setTurnReadyPool[key] = evt
}

// AddCommitment implements TxPoolEnqueuer; enqueues after validation (e.g. by Game).
func (p *txPool) AddCommitment(evt *types.SubmitPlayerCommitment) error {
	return p.addCommitment(evt)
}

// AddCard implements TxPoolEnqueuer; enqueues after validation (e.g. by Game).
func (p *txPool) AddCard(evt *types.SubmitPlayerCard) error {
	return p.addCard(evt)
}

// AddCreateRoom implements TxPoolEnqueuer (alias for addCreateRoom for interface).
func (p *txPool) AddCreateRoom(evt *types.RequireGameCreationEvent) {
	p.addCreateRoom(evt)
}

// AddSetTurnReady implements TxPoolEnqueuer (alias for addSetTurnReady for interface).
func (p *txPool) AddSetTurnReady(evt *types.RequireSetupNewTurnEvent) {
	p.addSetTurnReady(evt)
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
			p.processCreateRoomPool()
			p.processSetTurnReadyPool()
			p.processCommitmentPool()
			p.processCardPool()
		}
	}
}

// processCreateRoomPool drains create-room pool and calls chain CreateRoomContract for each.
func (p *txPool) processCreateRoomPool() {
	p.poolLock.Lock()
	events := make([]*types.RequireGameCreationEvent, 0, len(p.createRoomPool))
	for _, evt := range p.createRoomPool {
		events = append(events, evt)
	}
	for k := range p.createRoomPool {
		delete(p.createRoomPool, k)
	}
	p.poolLock.Unlock()
	for _, evt := range events {
		if err := p.chainSvc.CreateRoomContract(evt); err != nil {
			log.Errorw("failed to create room contract", "error", err, "game_id", evt.GameID)
			return
		}
	}
}

// processSetTurnReadyPool drains set-turn-ready pool and calls chain SetTurnReady for each.
func (p *txPool) processSetTurnReadyPool() {
	p.poolLock.Lock()
	events := make([]*types.RequireSetupNewTurnEvent, 0, len(p.setTurnReadyPool))
	for _, evt := range p.setTurnReadyPool {
		events = append(events, evt)
	}
	for k := range p.setTurnReadyPool {
		delete(p.setTurnReadyPool, k)
	}
	p.poolLock.Unlock()
	for _, evt := range events {
		if err := p.chainSvc.SetTurnReady(evt); err != nil {
			log.Errorw("failed to set turn ready", "error", err, "game_id", evt.GameID)
			return
		}
	}
}

// processCommitmentPool processes commitment pool and sends events to chain manager in batch
func (p *txPool) processCommitmentPool() {
	// Collect events to process while holding lock
	p.poolLock.Lock()
	batchEvents := make([]*types.SubmitPlayerCommitment, 0, len(p.commitmentPool))
	for key, evt := range p.commitmentPool {
		if evt.GameID != 0 {
			batchEvents = append(batchEvents, evt)
		} else {
			log.Errorw("commitment event missing GameID", "address", evt.Address.TemporaryAddress)
		}
		delete(p.commitmentPool, key)
	}
	p.poolLock.Unlock()

	if len(batchEvents) == 0 {
		return
	}

	// Submit in batches
	for start := 0; start < len(batchEvents); start += p.batchSize {
		end := start + p.batchSize
		if end > len(batchEvents) {
			end = len(batchEvents)
		}

		batch := batchEvents[start:end]

		if err := p.chainSvc.SubmitPlayerCommitmentsBatch(batch); err != nil {
			log.Errorw("failed to submit commitments batch to chain", "error", err, "count", len(batch))
			return
		}

		log.Infow("submitted commitments batch to chain", "count", len(batch))
	}
}

// processCardPool processes card pool and sends events to chain manager in batch
func (p *txPool) processCardPool() {
	// Collect events to process while holding lock
	p.poolLock.Lock()
	batchEvents := make([]*types.SubmitPlayerCard, 0, len(p.cardPool))
	for key, evt := range p.cardPool {
		if evt.GameID != 0 {
			batchEvents = append(batchEvents, evt)
		} else {
			log.Errorw("card event missing GameID", "address", evt.Address.TemporaryAddress)
		}
		delete(p.cardPool, key)
	}
	p.poolLock.Unlock()

	if len(batchEvents) == 0 {
		return
	}

	// Submit in batches
	for start := 0; start < len(batchEvents); start += p.batchSize {
		end := start + p.batchSize
		if end > len(batchEvents) {
			end = len(batchEvents)
		}

		batch := batchEvents[start:end]

		if err := p.chainSvc.SubmitPlayerCardsBatch(batch); err != nil {
			log.Errorw("failed to submit cards batch to chain", "error", err, "count", len(batch))
			return
		}

		log.Infow("submitted cards batch to chain", "count", len(batch))
	}
}

// clearGameInfo clears transaction info for a completed game
func (p *txPool) clearGameInfo(gameID uint) {
	p.txInfoLock.Lock()
	defer p.txInfoLock.Unlock()
	delete(p.gameTxInfos, gameID)
	log.Infow("cleared transaction info for game", "gameID", gameID)
}
