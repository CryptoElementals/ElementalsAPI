package worker

import (
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"strings"
	"time"

	"github.com/CryptoElementals/common/db"
	"github.com/CryptoElementals/common/log"
	dao "github.com/CryptoElementals/common/models"
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

var (
	claimChainTxPoolBatchForChain = func(chainID int64, limit int, claimTimeout time.Duration) ([]db.ChainTxPoolPendingRow, error) {
		return db.ClaimChainTxPoolBatchForChain(chainID, limit, claimTimeout)
	}
	deleteChainTxPoolItemsByIDs = db.DeleteChainTxPoolItemsByIDs
)

// roomBatchSubmitter posts encoded tasks to the RoomV3 client (or a test double).
type roomBatchSubmitter interface {
	SubmitTasks([]RoomContractTask) error
}

type txPool struct {
	chainID      int64
	sub          roomBatchSubmitter
	batchSize    int
	claimTimeout time.Duration
}

func newTxPool(rt *chainRuntime, batchSize int, claimTimeout time.Duration) *txPool {
	return newTxPoolWithSubmitter(rt, rt.chainID, batchSize, claimTimeout)
}

func newTxPoolWithSubmitter(sub roomBatchSubmitter, chainID int64, batchSize int, claimTimeout time.Duration) *txPool {
	if batchSize <= 0 {
		batchSize = defaultPoolBatchSize
	}
	if claimTimeout <= 0 {
		claimTimeout = db.DefaultChainTxPoolClaimTimeout
	}
	return &txPool{
		chainID:      chainID,
		sub:          sub,
		batchSize:    batchSize,
		claimTimeout: claimTimeout,
	}
}

func (p *txPool) addCommitment(evt *proto.SubmitPlayerCommitmentRequest) error {
	if evt == nil || evt.Address == nil {
		return fmt.Errorf("invalid commitment request")
	}
	addr := PlayerAddressFromProto(evt.Address)
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
	addr := PlayerAddressFromProto(evt.Address)
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

func (p *txPool) addCreateRoom(evt *RequireGameCreationEvent) {
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

func (p *txPool) addSetTurnReady(evt *RequireSetupNewTurnEvent) {
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

func (p *txPool) runDrainLoop() error {
	for {
		rows, err := claimChainTxPoolBatchForChain(p.chainID, p.batchSize, p.claimTimeout)
		if err != nil {
			return err
		}
		if len(rows) == 0 {
			return nil
		}

		flatTasks, rowIDs, dropIDs, err := p.loadPendingTasksFromRows(rows)
		if err != nil {
			log.Errorw("loadPendingTasksFromRows", "err", err, "chain_id", p.chainID)
			continue
		}
		if len(dropIDs) > 0 {
			if derr := deleteChainTxPoolItemsByIDs(dropIDs); derr != nil {
				log.Errorw("delete dropped pool rows", "err", derr, "chain_id", p.chainID)
			}
		}
		if len(flatTasks) == 0 {
			continue
		}

		if err := p.sub.SubmitTasks(flatTasks); err != nil {
			log.Errorw("failed to submit tasks batch to chain", "error", err, "count", len(flatTasks), "chain_id", p.chainID)
			return nil
		}
		if derr := deleteChainTxPoolItemsByIDs(rowIDs); derr != nil {
			log.Errorw("delete after submit: chain tx pool rows", "err", derr, "chain_id", p.chainID, "ids", rowIDs)
		}
		log.Infow("submitted tasks batch to chain", "count", len(flatTasks), "chain_id", p.chainID)
	}
}

func (p *txPool) loadPendingTasksFromRows(rows []db.ChainTxPoolPendingRow) (tasks []RoomContractTask, rowIDs []uint, dropIDs []uint, err error) {
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

func chainTxPoolKindLabel(kind uint8) string {
	switch kind {
	case dao.ChainTxPoolKindCreateRoom:
		return "create_room"
	case dao.ChainTxPoolKindSetTurnReady:
		return "set_turn_ready"
	case dao.ChainTxPoolKindCommitment:
		return "commitment"
	case dao.ChainTxPoolKindCard:
		return "card"
	default:
		return fmt.Sprintf("unknown(%d)", kind)
	}
}

func (p *txPool) debugLogPendingRowToTask(r db.ChainTxPoolPendingRow, playerIDs []int64) {
	args := []any{
		"pool_row_id", r.ID,
		"chain_id", p.chainID,
		"type", chainTxPoolKindLabel(r.Kind),
		"kind", r.Kind,
		"game_id", r.GameID,
		"round", r.RoundNumber,
		"turn", r.TurnNumber,
	}
	if r.PlayerTemporaryAddr != "" {
		args = append(args, "player_temporary_addr", r.PlayerTemporaryAddr)
	}
	switch len(playerIDs) {
	case 1:
		if playerIDs[0] != 0 {
			args = append(args, "player_id", playerIDs[0])
		}
	case 0:
	default:
		args = append(args, "player_ids", playerIDs)
	}
	log.Debugw("chain tx pool pending row to task", args...)
}

func playerIDsFromCreateRoomEvent(evt *RequireGameCreationEvent) []int64 {
	if evt == nil {
		return nil
	}
	var ids []int64
	for _, pl := range evt.Players {
		if pl.Id != 0 {
			ids = append(ids, pl.Id)
		}
	}
	return ids
}

func playerIDFromProtoAddress(addr *proto.PlayerAddress) int64 {
	if addr == nil {
		return 0
	}
	return addr.GetId()
}

func (p *txPool) pendingRowToTask(r db.ChainTxPoolPendingRow) (t RoomContractTask, ok bool, drop bool) {
	var playerIDs []int64
	defer func() { p.debugLogPendingRowToTask(r, playerIDs) }()

	switch r.Kind {
	case dao.ChainTxPoolKindCreateRoom:
		var evt RequireGameCreationEvent
		if err := json.Unmarshal(r.Payload, &evt); err != nil {
			log.Errorw("create_room pool row: bad json", "id", r.ID, "err", err)
			return t, false, true
		}
		playerIDs = playerIDsFromCreateRoomEvent(&evt)
		t, ok = encodeCreateRoomEventToTask(&evt)
		return t, ok, !ok
	case dao.ChainTxPoolKindSetTurnReady:
		var evt RequireSetupNewTurnEvent
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
		if id := playerIDFromProtoAddress(msg.GetAddress()); id != 0 {
			playerIDs = []int64{id}
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
		if id := playerIDFromProtoAddress(msg.GetAddress()); id != 0 {
			playerIDs = []int64{id}
		}
		t, ok = encodeCardEventToTask(&msg)
		return t, ok, !ok
	default:
		log.Errorw("unknown chain tx pool kind", "id", r.ID, "kind", r.Kind)
		return t, false, true
	}
}

func encodeCreateRoomEventToTask(evt *RequireGameCreationEvent) (RoomContractTask, bool) {
	if evt == nil || len(evt.Players) < 2 {
		gameID := int64(0)
		if evt != nil {
			gameID = evt.GameID
		}
		log.Errorw("failed to encode create room task: need 2 players", "game_id", gameID)
		return RoomContractTask{}, false
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
		return RoomContractTask{}, false
	}
	return RoomContractTask{Index: roomV3TaskIndexCreateRoom, Task: payload}, true
}

func encodeSetTurnReadyEventToTask(evt *RequireSetupNewTurnEvent) (RoomContractTask, bool) {
	if evt == nil {
		return RoomContractTask{}, false
	}
	gameID := new(big.Int).SetInt64(evt.GameID)
	payload, err := EncodeStartNewTurnTask(gameID)
	if err != nil {
		log.Errorw("failed to encode set turn ready task", "error", err, "game_id", evt.GameID)
		return RoomContractTask{}, false
	}
	return RoomContractTask{Index: roomV3TaskIndexStartNewTurn, Task: payload}, true
}

func encodeCommitmentEventToTask(evt *proto.SubmitPlayerCommitmentRequest) (RoomContractTask, bool) {
	if evt == nil {
		return RoomContractTask{}, false
	}
	if len(evt.Commitment) != 32 {
		log.Errorw("commitment must be 32 bytes", "len", len(evt.Commitment), "game_id", evt.GetGameID())
		return RoomContractTask{}, false
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
		return RoomContractTask{}, false
	}
	return RoomContractTask{Index: roomV3TaskIndexSubmitCardHash, Task: payload}, true
}

func encodeCardEventToTask(evt *proto.SubmitPlayerCardRequest) (RoomContractTask, bool) {
	if evt == nil {
		return RoomContractTask{}, false
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
		return RoomContractTask{}, false
	}
	return RoomContractTask{Index: roomV3TaskIndexSubmitCard, Task: payload}, true
}
