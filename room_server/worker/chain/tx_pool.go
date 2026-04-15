package chain

import (
	"context"
	"fmt"
	"math/big"
	"sync"
	"time"

	"github.com/CryptoElementals/common/log"
	"github.com/CryptoElementals/common/room_server/worker/types"
	"github.com/CryptoElementals/common/rpc/proto"
	"github.com/ethereum/go-ethereum/common"
)

const defaultPoolBatchSize = 10
const defaultPoolProcessingInterval = time.Second

type eventKey struct {
	gameID      int64
	address     string
	roundNumber uint32
	index       uint32
}

type setTurnKey struct {
	gameID      int64
	roundNumber uint32
	turnNumber  uint32
}

type txPool struct {
	rt *chainRuntime

	commitmentPool   map[eventKey]*proto.SubmitPlayerCommitmentRequest
	cardPool         map[eventKey]*proto.SubmitPlayerCardRequest
	createRoomPool   map[int64]*types.RequireGameCreationEvent
	setTurnReadyPool map[setTurnKey]*types.RequireSetupNewTurnEvent
	poolLock         sync.RWMutex

	gameTxInfos map[int64]*gameTxInfo
	txInfoLock  sync.RWMutex

	batchSize int
	tickerDur time.Duration
}

type gameTxInfo struct {
	playerTxInfos map[types.PlayerAddress]*gameTxPlayerInfo
}

type gameTxPlayerInfo struct {
	commitmentTurnIndices map[uint32]int32
	cardTurnIndices       map[uint32]int32
}

func newTxPool(rt *chainRuntime, batchSize int, processingIntervalSec int) *txPool {
	if batchSize <= 0 {
		batchSize = defaultPoolBatchSize
	}
	tickerDur := time.Duration(processingIntervalSec) * time.Second
	if tickerDur <= 0 {
		tickerDur = defaultPoolProcessingInterval
	}
	return &txPool{
		rt:               rt,
		commitmentPool:   make(map[eventKey]*proto.SubmitPlayerCommitmentRequest),
		cardPool:         make(map[eventKey]*proto.SubmitPlayerCardRequest),
		createRoomPool:   make(map[int64]*types.RequireGameCreationEvent),
		setTurnReadyPool: make(map[setTurnKey]*types.RequireSetupNewTurnEvent),
		gameTxInfos:      make(map[int64]*gameTxInfo),
		batchSize:        batchSize,
		tickerDur:        tickerDur,
	}
}

func makeEventKey(gameID int64, address types.PlayerAddress, roundNumber, index uint32) eventKey {
	return eventKey{
		gameID:      gameID,
		address:     address.TemporaryAddress,
		roundNumber: roundNumber,
		index:       index,
	}
}

func (p *txPool) getOrCreatePlayerInfo(gameID int64, address types.PlayerAddress) *gameTxPlayerInfo {
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

func (p *txPool) addCommitment(evt *proto.SubmitPlayerCommitmentRequest) error {
	if evt == nil || evt.Address == nil {
		return fmt.Errorf("invalid commitment request")
	}
	var addr types.PlayerAddress
	addr.FromProto(evt.Address)
	gameID := evt.GetGameID()

	p.txInfoLock.Lock()
	playerInfo := p.getOrCreatePlayerInfo(gameID, addr)
	maxTurnIndex := playerInfo.commitmentTurnIndices[evt.RoundNumber]
	if int32(evt.TurnNumber) <= maxTurnIndex {
		p.txInfoLock.Unlock()
		return fmt.Errorf("commitment with round %d, turn index %d rejected: already received index %d or higher for this round", evt.RoundNumber, evt.TurnNumber, maxTurnIndex)
	}
	playerInfo.commitmentTurnIndices[evt.RoundNumber] = int32(evt.TurnNumber)
	p.txInfoLock.Unlock()

	key := makeEventKey(gameID, addr, evt.RoundNumber, evt.TurnNumber)
	p.poolLock.Lock()
	defer p.poolLock.Unlock()
	if _, exists := p.commitmentPool[key]; exists {
		return fmt.Errorf("commitment already in pool")
	}
	p.commitmentPool[key] = evt
	log.Infow("request added to tx pool", "kind", "commitment", "gameID", gameID, "chain_id", p.rt.chainID, "address", addr.TemporaryAddress, "round", evt.RoundNumber, "turn", evt.TurnNumber)
	return nil
}

func (p *txPool) addCard(evt *proto.SubmitPlayerCardRequest) error {
	if evt == nil || evt.Address == nil {
		return fmt.Errorf("invalid card request")
	}
	var addr types.PlayerAddress
	addr.FromProto(evt.Address)
	gameID := evt.GetGameID()

	p.txInfoLock.Lock()
	playerInfo := p.getOrCreatePlayerInfo(gameID, addr)
	maxTurnIndex := playerInfo.cardTurnIndices[evt.RoundNumber]
	if int32(evt.TurnNumber) <= maxTurnIndex {
		p.txInfoLock.Unlock()
		return fmt.Errorf("card with round %d, turn index %d rejected: already received index %d or higher for this round", evt.RoundNumber, evt.TurnNumber, maxTurnIndex)
	}
	playerInfo.cardTurnIndices[evt.RoundNumber] = int32(evt.TurnNumber)
	p.txInfoLock.Unlock()

	key := makeEventKey(gameID, addr, evt.RoundNumber, evt.TurnNumber)
	p.poolLock.Lock()
	defer p.poolLock.Unlock()
	if _, exists := p.cardPool[key]; exists {
		return fmt.Errorf("card already in pool")
	}
	p.cardPool[key] = evt
	log.Infow("request added to tx pool", "kind", "card", "gameID", gameID, "chain_id", p.rt.chainID, "address", addr.TemporaryAddress, "round", evt.RoundNumber, "turn", evt.TurnNumber)
	return nil
}

func (p *txPool) addCreateRoom(evt *types.RequireGameCreationEvent) {
	p.poolLock.Lock()
	defer p.poolLock.Unlock()
	p.createRoomPool[evt.GameID] = evt
	log.Infow("request added to tx pool", "kind", "create_room", "gameID", evt.GameID, "chain_id", p.rt.chainID)
}

func (p *txPool) addSetTurnReady(evt *types.RequireSetupNewTurnEvent) {
	key := setTurnKey{gameID: evt.GameID, roundNumber: evt.RoundNumber, turnNumber: evt.TurnNumber}
	p.poolLock.Lock()
	defer p.poolLock.Unlock()
	p.setTurnReadyPool[key] = evt
	log.Infow("request added to tx pool", "kind", "set_turn_ready", "gameID", evt.GameID, "chain_id", p.rt.chainID, "round", evt.RoundNumber, "turn", evt.TurnNumber)
}

func (p *txPool) processPools(ctx context.Context) {
	ticker := time.NewTicker(p.tickerDur)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
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
			for start := 0; start < len(flatTasks); start += p.batchSize {
				end := start + p.batchSize
				if end > len(flatTasks) {
					end = len(flatTasks)
				}
				batch := flatTasks[start:end]
				if err := p.rt.SubmitTasks(batch); err != nil {
					log.Errorw("failed to submit tasks batch to chain", "error", err, "count", len(batch), "chain_id", p.rt.chainID)
					break
				}
				log.Infow("submitted tasks batch to chain", "count", len(batch), "chain_id", p.rt.chainID)
			}
		}
	}
}

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

func (p *txPool) processCommitmentPool() []types.RoomContractTask {
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

func (p *txPool) processCardPool() []types.RoomContractTask {
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

func (p *txPool) clearGameInfo(gameID int64) {
	p.txInfoLock.Lock()
	defer p.txInfoLock.Unlock()
	delete(p.gameTxInfos, gameID)
	log.Infow("cleared transaction info for game", "gameID", gameID, "chain_id", p.rt.chainID)
}

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
		totalCardIndex := big.NewInt(3)
		initialHP := big.NewInt(evt.InitialHP)
		gameID := new(big.Int).SetInt64(evt.GameID)
		payload, err := EncodeCreateRoomTask(
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

func encodeSetTurnReadyEventsToTasks(events []*types.RequireSetupNewTurnEvent) []types.RoomContractTask {
	tasks := make([]types.RoomContractTask, 0, len(events))
	for _, evt := range events {
		gameID := new(big.Int).SetInt64(evt.GameID)
		payload, err := EncodeStartNewTurnTask(gameID)
		if err != nil {
			log.Errorw("failed to encode set turn ready task", "error", err, "game_id", evt.GameID)
			continue
		}
		tasks = append(tasks, types.RoomContractTask{Index: 2, Task: payload})
	}
	return tasks
}

func encodeCommitmentEventsToTasks(events []*proto.SubmitPlayerCommitmentRequest) []types.RoomContractTask {
	tasks := make([]types.RoomContractTask, 0, len(events))
	for _, evt := range events {
		if len(evt.Commitment) != 32 {
			log.Errorw("commitment must be 32 bytes", "len", len(evt.Commitment), "game_id", evt.GetGameID())
			continue
		}
		var commitmentHash [32]byte
		copy(commitmentHash[:], evt.Commitment)
		gameID := new(big.Int).SetInt64(evt.GetGameID())
		cardIndex := big.NewInt(int64(evt.TurnNumber))
		round := big.NewInt(int64(evt.RoundNumber))
		payload, err := EncodeSubmitCardHashTask(
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

func encodeCardEventsToTasks(events []*proto.SubmitPlayerCardRequest) []types.RoomContractTask {
	tasks := make([]types.RoomContractTask, 0, len(events))
	for _, evt := range events {
		gameID := new(big.Int).SetInt64(evt.GetGameID())
		card := big.NewInt(int64(evt.Card))
		cardIndex := big.NewInt(int64(evt.TurnNumber))
		round := big.NewInt(int64(evt.RoundNumber))
		payload, err := EncodeSubmitCardTask(
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

// --- Chain implements game.TxPoolEnqueuer (method set below) ---

func (h *Chain) poolForGame(gameID int64) (*txPool, error) {
	cid, err := h.chainIDForGame(gameID)
	if err != nil {
		return nil, err
	}
	p, ok := h.pools[cid]
	if !ok {
		return nil, fmt.Errorf("no tx pool for chain_id %d", cid)
	}
	return p, nil
}

// AddSetTurnReady implements the game TxPoolEnqueuer contract.
func (h *Chain) AddSetTurnReady(evt *types.RequireSetupNewTurnEvent) {
	if evt == nil {
		return
	}
	p, err := h.poolForGame(evt.GameID)
	if err != nil {
		log.Errorw("AddSetTurnReady: resolve pool", "gameID", evt.GameID, "err", err)
		return
	}
	p.addSetTurnReady(evt)
}

// AddCommitment implements the game TxPoolEnqueuer contract.
func (h *Chain) AddCommitment(evt *proto.SubmitPlayerCommitmentRequest) error {
	if evt == nil {
		return fmt.Errorf("nil request")
	}
	p, err := h.poolForGame(evt.GetGameID())
	if err != nil {
		return err
	}
	return p.addCommitment(evt)
}

// AddCard implements the game TxPoolEnqueuer contract.
func (h *Chain) AddCard(evt *proto.SubmitPlayerCardRequest) error {
	if evt == nil {
		return fmt.Errorf("nil request")
	}
	p, err := h.poolForGame(evt.GetGameID())
	if err != nil {
		return err
	}
	return p.addCard(evt)
}

// ClearGameInfo implements the game TxPoolEnqueuer contract.
func (h *Chain) ClearGameInfo(gameID int64) {
	for _, p := range h.pools {
		p.clearGameInfo(gameID)
	}
	h.gameToChainMu.Lock()
	delete(h.gameToChainID, gameID)
	h.gameToChainMu.Unlock()
}
