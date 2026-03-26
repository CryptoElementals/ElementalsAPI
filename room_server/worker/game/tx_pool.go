package game

import (
	"context"
	"fmt"
	"math/big"
	"sync"
	"time"

	"github.com/CryptoElementals/common/log"
	"github.com/CryptoElementals/common/room_server/worker/chain"
	"github.com/CryptoElementals/common/room_server/worker/types"
	"github.com/CryptoElementals/common/rpc/proto"
	"github.com/ethereum/go-ethereum/common"
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
	commitmentPool map[eventKey]*proto.SubmitPlayerCommitmentRequest
	cardPool       map[eventKey]*proto.SubmitPlayerCardRequest
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
		commitmentPool:   make(map[eventKey]*proto.SubmitPlayerCommitmentRequest),
		cardPool:         make(map[eventKey]*proto.SubmitPlayerCardRequest),
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
func (p *txPool) addCommitment(evt *proto.SubmitPlayerCommitmentRequest) error {
	if evt == nil || evt.Address == nil {
		return fmt.Errorf("invalid commitment request")
	}
	var addr types.PlayerAddress
	addr.FromProto(evt.Address)
	gameID := uint(evt.GetGameID())

	p.txInfoLock.Lock()
	playerInfo := p.getOrCreatePlayerInfo(gameID, addr)

	// Get max received turn index for this round
	maxTurnIndex := playerInfo.commitmentTurnIndices[evt.RoundNumber]

	// Reject if turn index is <= max received for this round
	if int32(evt.TurnNumber) <= maxTurnIndex {
		p.txInfoLock.Unlock()
		return fmt.Errorf("commitment with round %d, turn index %d rejected: already received index %d or higher for this round", evt.RoundNumber, evt.TurnNumber, maxTurnIndex)
	}

	// Update max received index for this round
	playerInfo.commitmentTurnIndices[evt.RoundNumber] = int32(evt.TurnNumber)
	p.txInfoLock.Unlock()

	// Add to pool
	key := makeEventKey(gameID, addr, evt.RoundNumber, evt.TurnNumber)
	p.poolLock.Lock()
	defer p.poolLock.Unlock()

	if _, exists := p.commitmentPool[key]; exists {
		return fmt.Errorf("commitment already in pool")
	}

	p.commitmentPool[key] = evt
	return nil
}

// addCard adds a card to the pool after validating turn index
func (p *txPool) addCard(evt *proto.SubmitPlayerCardRequest) error {
	if evt == nil || evt.Address == nil {
		return fmt.Errorf("invalid card request")
	}
	var addr types.PlayerAddress
	addr.FromProto(evt.Address)
	gameID := uint(evt.GetGameID())

	p.txInfoLock.Lock()
	playerInfo := p.getOrCreatePlayerInfo(gameID, addr)

	// Get max received turn index for this round
	maxTurnIndex := playerInfo.cardTurnIndices[evt.RoundNumber]

	// Reject if turn index is <= max received for this round
	if int32(evt.TurnNumber) <= maxTurnIndex {
		p.txInfoLock.Unlock()
		return fmt.Errorf("card with round %d, turn index %d rejected: already received index %d or higher for this round", evt.RoundNumber, evt.TurnNumber, maxTurnIndex)
	}

	// Update max received index for this round
	playerInfo.cardTurnIndices[evt.RoundNumber] = int32(evt.TurnNumber)
	p.txInfoLock.Unlock()

	// Add to pool
	key := makeEventKey(gameID, addr, evt.RoundNumber, evt.TurnNumber)
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
func (p *txPool) AddCommitment(evt *proto.SubmitPlayerCommitmentRequest) error {
	return p.addCommitment(evt)
}

// AddCard implements TxPoolEnqueuer; enqueues after validation (e.g. by Game).
func (p *txPool) AddCard(evt *proto.SubmitPlayerCardRequest) error {
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

// processPools periodically processes events in the pools and sends them to chain manager.
// Tick interval is fixed at 1s (previously configurable via GameArgs).
func (p *txPool) processPools(ctx context.Context) {
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			// Collect tasks from all pools and submit them in batches.
			var flatTasks []types.RoomContractTask
			if tasks := p.processCreateRoomPool(); len(tasks) > 0 {
				flatTasks = append(flatTasks, tasks...)
			}
			if tasks := p.processSetTurnReadyPool(); len(tasks) > 0 {
				flatTasks = append(flatTasks, tasks...)
			}
			if tasks := p.processCommitmentPool(); len(tasks) > 0 {
				flatTasks = append(flatTasks, tasks...)
			}
			if tasks := p.processCardPool(); len(tasks) > 0 {
				flatTasks = append(flatTasks, tasks...)
			}

			if len(flatTasks) == 0 {
				continue
			}

			// Respect pool batch size when sending tasks to chain.
			for start := 0; start < len(flatTasks); start += p.batchSize {
				end := start + p.batchSize
				if end > len(flatTasks) {
					end = len(flatTasks)
				}
				batch := flatTasks[start:end]
				if err := p.chainSvc.SubmitTasks(batch); err != nil {
					log.Errorw("failed to submit tasks batch to chain", "error", err, "count", len(batch))
					break
				}
				log.Infow("submitted tasks batch to chain", "count", len(batch))
			}
		}
	}
}

// processCreateRoomPool drains create-room pool and returns encoded tasks.
func (p *txPool) processCreateRoomPool() []types.RoomContractTask {
	p.poolLock.Lock()
	events := make([]*types.RequireGameCreationEvent, 0, len(p.createRoomPool))
	for _, evt := range p.createRoomPool {
		events = append(events, evt)
	}
	for k := range p.createRoomPool {
		delete(p.createRoomPool, k)
	}
	p.poolLock.Unlock()
	if len(events) == 0 {
		return nil
	}

	return encodeCreateRoomEventsToTasks(events)
}

// processSetTurnReadyPool drains set-turn-ready pool and returns encoded tasks.
func (p *txPool) processSetTurnReadyPool() []types.RoomContractTask {
	p.poolLock.Lock()
	events := make([]*types.RequireSetupNewTurnEvent, 0, len(p.setTurnReadyPool))
	for _, evt := range p.setTurnReadyPool {
		events = append(events, evt)
	}
	for k := range p.setTurnReadyPool {
		delete(p.setTurnReadyPool, k)
	}
	p.poolLock.Unlock()
	if len(events) == 0 {
		return nil
	}

	return encodeSetTurnReadyEventsToTasks(events)
}

// processCommitmentPool drains commitment pool and returns encoded tasks.
func (p *txPool) processCommitmentPool() []types.RoomContractTask {
	// Collect events to process while holding lock
	p.poolLock.Lock()
	batchEvents := make([]*proto.SubmitPlayerCommitmentRequest, 0, len(p.commitmentPool))
	for key, evt := range p.commitmentPool {
		if evt.GetGameID() != 0 {
			batchEvents = append(batchEvents, evt)
		} else {
			addr := ""
			if evt.Address != nil {
				addr = evt.Address.TemporaryAddress
			}
			log.Errorw("commitment event missing GameID", "address", addr)
		}
		delete(p.commitmentPool, key)
	}
	p.poolLock.Unlock()

	if len(batchEvents) == 0 {
		return nil
	}

	return encodeCommitmentEventsToTasks(batchEvents)
}

// processCardPool drains card pool and returns encoded tasks.
func (p *txPool) processCardPool() []types.RoomContractTask {
	// Collect events to process while holding lock
	p.poolLock.Lock()
	batchEvents := make([]*proto.SubmitPlayerCardRequest, 0, len(p.cardPool))
	for key, evt := range p.cardPool {
		if evt.GetGameID() != 0 {
			batchEvents = append(batchEvents, evt)
		} else {
			addr := ""
			if evt.Address != nil {
				addr = evt.Address.TemporaryAddress
			}
			log.Errorw("card event missing GameID", "address", addr)
		}
		delete(p.cardPool, key)
	}
	p.poolLock.Unlock()

	if len(batchEvents) == 0 {
		return nil
	}

	return encodeCardEventsToTasks(batchEvents)
}

// clearGameInfo clears transaction info for a completed game
func (p *txPool) clearGameInfo(gameID uint) {
	p.txInfoLock.Lock()
	defer p.txInfoLock.Unlock()
	delete(p.gameTxInfos, gameID)
	log.Infow("cleared transaction info for game", "gameID", gameID)
}

// ClearGameInfo implements TxPoolEnqueuer.
func (p *txPool) ClearGameInfo(gameID uint) {
	p.clearGameInfo(gameID)
}

// encodeCreateRoomEventsToTasks converts create-room events into encoded RoomV3 tasks.
func encodeCreateRoomEventsToTasks(events []*types.RequireGameCreationEvent) []types.RoomContractTask {
	tasks := make([]types.RoomContractTask, 0, len(events))
	for _, evt := range events {
		if len(evt.Players) < 2 {
			log.Errorw("failed to encode create room task: need 2 players", "game_id", evt.GameID)
			continue
		}
		player1 := evt.Players[0]
		player2 := evt.Players[1]

		player1ID := big.NewInt(player1.Id)
		player2ID := big.NewInt(player2.Id)
		player1Addr := common.HexToAddress(player1.TemporaryAddress)
		player2Addr := common.HexToAddress(player2.TemporaryAddress)
		roundTimeout := big.NewInt(evt.RoundTimeout)
		totalRound := big.NewInt(evt.MaxRoundNumber)
		totalCardIndex := big.NewInt(3) // 3 cards per round
		initialHP := big.NewInt(evt.InitialHP)
		gameID := big.NewInt(int64(evt.GameID))

		payload, err := chain.EncodeCreateRoomTask(
			player1ID,
			player2ID,
			player1Addr,
			player2Addr,
			roundTimeout,
			totalRound,
			totalCardIndex,
			initialHP,
			gameID,
		)
		if err != nil {
			log.Errorw("failed to encode create room task", "error", err, "game_id", evt.GameID)
			continue
		}
		tasks = append(tasks, types.RoomContractTask{Index: 1, Task: payload})
	}
	return tasks
}

// encodeSetTurnReadyEventsToTasks converts set-turn-ready events into encoded RoomV3 tasks.
func encodeSetTurnReadyEventsToTasks(events []*types.RequireSetupNewTurnEvent) []types.RoomContractTask {
	tasks := make([]types.RoomContractTask, 0, len(events))
	for _, evt := range events {
		gameID := big.NewInt(int64(evt.GameID))
		payload, err := chain.EncodeStartNewTurnTask(gameID)
		if err != nil {
			log.Errorw("failed to encode set turn ready task", "error", err, "game_id", evt.GameID)
			continue
		}
		tasks = append(tasks, types.RoomContractTask{Index: 2, Task: payload})
	}
	return tasks
}

// encodeCommitmentEventsToTasks converts a batch of commitment events into encoded RoomV3 tasks.
func encodeCommitmentEventsToTasks(events []*proto.SubmitPlayerCommitmentRequest) []types.RoomContractTask {
	tasks := make([]types.RoomContractTask, 0, len(events))
	for _, evt := range events {
		if len(evt.Commitment) != 32 {
			log.Errorw("commitment must be 32 bytes", "len", len(evt.Commitment), "game_id", evt.GetGameID())
			continue
		}
		var commitmentHash [32]byte
		copy(commitmentHash[:], evt.Commitment)

		gameID := big.NewInt(int64(evt.GetGameID()))
		cardIndex := big.NewInt(int64(evt.TurnNumber))
		round := big.NewInt(int64(evt.RoundNumber))

		payload, err := chain.EncodeSubmitCardHashTask(
			gameID,
			commitmentHash,
			cardIndex,
			round,
			evt.Signature,
		)
		if err != nil {
			log.Errorw("failed to encode commitment task", "error", err, "game_id", evt.GetGameID())
			continue
		}
		tasks = append(tasks, types.RoomContractTask{Index: 3, Task: payload})
	}
	return tasks
}

// encodeCardEventsToTasks converts a batch of card events into encoded RoomV3 tasks.
func encodeCardEventsToTasks(events []*proto.SubmitPlayerCardRequest) []types.RoomContractTask {
	tasks := make([]types.RoomContractTask, 0, len(events))
	for _, evt := range events {
		gameID := big.NewInt(int64(evt.GetGameID()))
		card := big.NewInt(int64(evt.Card))
		cardIndex := big.NewInt(int64(evt.TurnNumber))
		round := big.NewInt(int64(evt.RoundNumber))

		payload, err := chain.EncodeSubmitCardTask(
			gameID,
			card,
			evt.Salt,
			cardIndex,
			round,
			evt.Signature,
		)
		if err != nil {
			log.Errorw("failed to encode card task", "error", err, "game_id", evt.GetGameID())
			continue
		}
		tasks = append(tasks, types.RoomContractTask{Index: 4, Task: payload})
	}
	return tasks
}
