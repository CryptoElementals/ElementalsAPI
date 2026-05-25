package chain

import (
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"strings"

	"github.com/CryptoElementals/common/db"
	"github.com/CryptoElementals/common/log"
	dao "github.com/CryptoElementals/common/models"
	"github.com/CryptoElementals/common/room_server/worker/types"
	"github.com/CryptoElementals/common/rpc/proto"
	"github.com/ethereum/go-ethereum/common"
	goproto "google.golang.org/protobuf/proto"
)

const defaultPoolBatchSize = 10

// Room V3 batchSubmitTasks task indices (must match on-chain task router).
const (
	roomV3TaskIndexCreateRoom     uint8 = 1
	roomV3TaskIndexStartNewTurn   uint8 = 2
	roomV3TaskIndexSubmitCardHash uint8 = 3
	roomV3TaskIndexSubmitCard     uint8 = 4
)

// roomBatchSubmitter posts encoded tasks to the RoomV3 client (or a test double).
type roomBatchSubmitter interface {
	SubmitTasks([]types.RoomContractTask) error
}

type txPool struct {
	chainID int64
	sub     roomBatchSubmitter

	batchSize int
}

func newTxPool(rt *chainRuntime, batchSize int) *txPool {
	return newTxPoolWithSubmitter(rt, rt.chainID, batchSize)
}

func newTxPoolWithSubmitter(sub roomBatchSubmitter, chainID int64, batchSize int) *txPool {
	if batchSize <= 0 {
		batchSize = defaultPoolBatchSize
	}
	return &txPool{
		chainID:   chainID,
		sub:       sub,
		batchSize: batchSize,
	}
}

func (p *txPool) addCommitment(evt *proto.SubmitPlayerCommitmentRequest) error {
	if evt == nil || evt.Address == nil {
		return fmt.Errorf("invalid commitment request")
	}
	var addr types.PlayerAddress
	addr.FromProto(evt.Address)
	gameID := evt.GetGameID()
	taddr := strings.ToLower(addr.TemporaryAddress)

	payload, err := goproto.Marshal(evt)
	if err != nil {
		return err
	}
	row := &dao.ChainTxPoolItem{
		ChainID:             p.chainID,
		Kind:                dao.ChainTxPoolKindCommitment,
		GameID:              gameID,
		PlayerTemporaryAddr: taddr,
		RoundNumber:         evt.RoundNumber,
		TurnNumber:          evt.TurnNumber,
		Payload:             payload,
	}
	if err := db.InsertChainTxPoolItem(row); err != nil {
		if errors.Is(err, db.ErrChainTxPoolDuplicate) {
			return fmt.Errorf("commitment already in pool")
		}
		return err
	}

	log.Infow("request added to tx pool", "kind", "commitment", "gameID", gameID, "chain_id", p.chainID, "address", taddr, "round", evt.RoundNumber, "turn", evt.TurnNumber)
	return nil
}

func (p *txPool) addCard(evt *proto.SubmitPlayerCardRequest) error {
	if evt == nil || evt.Address == nil {
		return fmt.Errorf("invalid card request")
	}
	var addr types.PlayerAddress
	addr.FromProto(evt.Address)
	gameID := evt.GetGameID()
	taddr := strings.ToLower(addr.TemporaryAddress)

	payload, err := goproto.Marshal(evt)
	if err != nil {
		return err
	}
	row := &dao.ChainTxPoolItem{
		ChainID:             p.chainID,
		Kind:                dao.ChainTxPoolKindCard,
		GameID:              gameID,
		PlayerTemporaryAddr: taddr,
		RoundNumber:         evt.RoundNumber,
		TurnNumber:          evt.TurnNumber,
		Payload:             payload,
	}
	if err := db.InsertChainTxPoolItem(row); err != nil {
		if errors.Is(err, db.ErrChainTxPoolDuplicate) {
			return fmt.Errorf("card already in pool")
		}
		return err
	}

	log.Infow("request added to tx pool", "kind", "card", "gameID", gameID, "chain_id", p.chainID, "address", taddr, "round", evt.RoundNumber, "turn", evt.TurnNumber)
	return nil
}

func (p *txPool) addCreateRoom(evt *types.RequireGameCreationEvent) {
	if evt == nil {
		return
	}
	payload, err := json.Marshal(evt)
	if err != nil {
		log.Errorw("addCreateRoom: marshal", "err", err, "gameID", evt.GameID)
		return
	}
	row := &dao.ChainTxPoolItem{
		ChainID:             p.chainID,
		Kind:                dao.ChainTxPoolKindCreateRoom,
		GameID:              evt.GameID,
		PlayerTemporaryAddr: "",
		RoundNumber:         0,
		TurnNumber:          0,
		Payload:             payload,
	}
	if err := db.InsertChainTxPoolItem(row); err != nil {
		if errors.Is(err, db.ErrChainTxPoolDuplicate) {
			log.Errorw("addCreateRoom: duplicate in pool", "err", err, "gameID", evt.GameID, "chain_id", p.chainID)
			return
		}
		log.Errorw("addCreateRoom: insert", "err", err, "gameID", evt.GameID, "chain_id", p.chainID)
		return
	}
	log.Infow("request added to tx pool", "kind", "create_room", "gameID", evt.GameID, "chain_id", p.chainID)
}

func (p *txPool) addSetTurnReady(evt *types.RequireSetupNewTurnEvent) {
	if evt == nil {
		return
	}
	payload, err := json.Marshal(evt)
	if err != nil {
		log.Errorw("addSetTurnReady: marshal", "err", err, "gameID", evt.GameID)
		return
	}
	row := &dao.ChainTxPoolItem{
		ChainID:             p.chainID,
		Kind:                dao.ChainTxPoolKindSetTurnReady,
		GameID:              evt.GameID,
		PlayerTemporaryAddr: "",
		RoundNumber:         evt.RoundNumber,
		TurnNumber:          evt.TurnNumber,
		Payload:             payload,
	}
	if err := db.InsertChainTxPoolItem(row); err != nil {
		if errors.Is(err, db.ErrChainTxPoolDuplicate) {
			log.Errorw("addSetTurnReady: duplicate in pool", "err", err, "gameID", evt.GameID, "chain_id", p.chainID)
			return
		}
		log.Errorw("addSetTurnReady: insert", "err", err, "gameID", evt.GameID, "chain_id", p.chainID)
		return
	}
	log.Infow("request added to tx pool", "kind", "set_turn_ready", "gameID", evt.GameID, "chain_id", p.chainID, "round", evt.RoundNumber, "turn", evt.TurnNumber)
}

// runPoolTickForOrderedRows submits pending work for this chain. Rows must already be in flush order
// (see db.chainItemsToPendingRowsInFlushOrder).
func (p *txPool) runPoolTickForOrderedRows(ordered []db.ChainTxPoolPendingRow) {
	flatTasks, rowIDs, dropIDs, err := p.loadPendingTasksFromRows(ordered)
	if err != nil {
		log.Errorw("loadPendingTasksFromRows", "err", err, "chain_id", p.chainID)
		return
	}
	if len(dropIDs) > 0 {
		if derr := db.DeleteChainTxPoolItemsByIDs(dropIDs); derr != nil {
			log.Errorw("delete dropped pool rows", "err", derr, "chain_id", p.chainID)
		}
	}
	if len(flatTasks) == 0 {
		return
	}
	for start := 0; start < len(flatTasks); start += p.batchSize {
		end := start + p.batchSize
		if end > len(flatTasks) {
			end = len(flatTasks)
		}
		batch := flatTasks[start:end]
		ids := rowIDs[start:end]
		if err := p.sub.SubmitTasks(batch); err != nil {
			log.Errorw("failed to submit tasks batch to chain", "error", err, "count", len(batch), "chain_id", p.chainID)
			break
		}
		if derr := db.DeleteChainTxPoolItemsByIDs(ids); derr != nil {
			log.Errorw("delete after submit: chain tx pool rows", "err", derr, "chain_id", p.chainID, "ids", ids)
		}
		log.Infow("submitted tasks batch to chain", "count", len(batch), "chain_id", p.chainID)
	}
}

func (p *txPool) loadPendingTasksFromRows(rows []db.ChainTxPoolPendingRow) (tasks []types.RoomContractTask, rowIDs []uint, dropIDs []uint, err error) {
	for _, r := range rows {
		t, ok, drop := p.pendingRowToTask(r)
		if drop {
			dropIDs = append(dropIDs, r.ID)
			continue
		}
		if !ok {
			continue
		}
		tasks = append(tasks, t)
		rowIDs = append(rowIDs, r.ID)
	}
	return tasks, rowIDs, dropIDs, nil
}

func (p *txPool) pendingRowToTask(r db.ChainTxPoolPendingRow) (t types.RoomContractTask, ok bool, drop bool) {
	switch r.Kind {
	case dao.ChainTxPoolKindCreateRoom:
		var evt types.RequireGameCreationEvent
		if err := json.Unmarshal(r.Payload, &evt); err != nil {
			log.Errorw("create_room pool row: bad json", "id", r.ID, "err", err)
			return t, false, true
		}
		t, ok = encodeCreateRoomEventToTask(&evt)
		return t, ok, !ok
	case dao.ChainTxPoolKindSetTurnReady:
		var evt types.RequireSetupNewTurnEvent
		if err := json.Unmarshal(r.Payload, &evt); err != nil {
			log.Errorw("set_turn pool row: bad json", "id", r.ID, "err", err)
			return t, false, true
		}
		t, ok = encodeSetTurnReadyEventToTask(&evt)
		return t, ok, !ok
	case dao.ChainTxPoolKindCommitment:
		var msg proto.SubmitPlayerCommitmentRequest
		if err := goproto.Unmarshal(r.Payload, &msg); err != nil {
			log.Errorw("commitment pool row: bad proto", "id", r.ID, "err", err)
			return t, false, true
		}
		if msg.GetGameID() == 0 {
			log.Errorw("commitment event missing GameID", "id", r.ID)
			return t, false, true
		}
		t, ok = encodeCommitmentEventToTask(&msg)
		return t, ok, !ok
	case dao.ChainTxPoolKindCard:
		var msg proto.SubmitPlayerCardRequest
		if err := goproto.Unmarshal(r.Payload, &msg); err != nil {
			log.Errorw("card pool row: bad proto", "id", r.ID, "err", err)
			return t, false, true
		}
		if msg.GetGameID() == 0 {
			log.Errorw("card event missing GameID", "id", r.ID)
			return t, false, true
		}
		t, ok = encodeCardEventToTask(&msg)
		return t, ok, !ok
	default:
		log.Errorw("unknown chain tx pool kind", "id", r.ID, "kind", r.Kind)
		return t, false, true
	}
}

func encodeCreateRoomEventToTask(evt *types.RequireGameCreationEvent) (types.RoomContractTask, bool) {
	if evt == nil || len(evt.Players) < 2 {
		gameID := int64(0)
		if evt != nil {
			gameID = evt.GameID
		}
		log.Errorw("failed to encode create room task: need 2 players", "game_id", gameID)
		return types.RoomContractTask{}, false
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
	tournamentID := new(big.Int).SetInt64(evt.TournamentID)
	tierNo := new(big.Int).SetInt64(evt.TierNo)
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
		tournamentID,
		tierNo,
	)
	if err != nil {
		log.Errorw("failed to encode create room task", "error", err, "game_id", evt.GameID)
		return types.RoomContractTask{}, false
	}
	return types.RoomContractTask{Index: roomV3TaskIndexCreateRoom, Task: payload}, true
}

func encodeSetTurnReadyEventToTask(evt *types.RequireSetupNewTurnEvent) (types.RoomContractTask, bool) {
	if evt == nil {
		return types.RoomContractTask{}, false
	}
	gameID := new(big.Int).SetInt64(evt.GameID)
	payload, err := EncodeStartNewTurnTask(gameID)
	if err != nil {
		log.Errorw("failed to encode set turn ready task", "error", err, "game_id", evt.GameID)
		return types.RoomContractTask{}, false
	}
	return types.RoomContractTask{Index: roomV3TaskIndexStartNewTurn, Task: payload}, true
}

func encodeCommitmentEventToTask(evt *proto.SubmitPlayerCommitmentRequest) (types.RoomContractTask, bool) {
	if evt == nil {
		return types.RoomContractTask{}, false
	}
	if len(evt.Commitment) != 32 {
		log.Errorw("commitment must be 32 bytes", "len", len(evt.Commitment), "game_id", evt.GetGameID())
		return types.RoomContractTask{}, false
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
		return types.RoomContractTask{}, false
	}
	return types.RoomContractTask{Index: roomV3TaskIndexSubmitCardHash, Task: payload}, true
}

func encodeCardEventToTask(evt *proto.SubmitPlayerCardRequest) (types.RoomContractTask, bool) {
	if evt == nil {
		return types.RoomContractTask{}, false
	}
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
		return types.RoomContractTask{}, false
	}
	return types.RoomContractTask{Index: roomV3TaskIndexSubmitCard, Task: payload}, true
}

// --- Chain implements game.TxPoolEnqueuer (method set below) ---

func (h *Chain) poolForGame(gameID int64) (*txPool, error) {
	cid, err := db.GetChainIDForGame(gameID)
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
	if err := db.DeleteChainTxPoolItemsForGame(gameID); err != nil {
		log.Errorw("ClearGameInfo: delete chain tx pool rows", "gameID", gameID, "err", err)
	}
}
