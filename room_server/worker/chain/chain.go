package chain

import (
	"context"
	"crypto/rand"
	"errors"
	"math/big"
	"strings"
	"time"

	"github.com/CryptoElementals/common/config"
	"github.com/CryptoElementals/common/db"
	"github.com/CryptoElementals/common/log"
	"github.com/CryptoElementals/common/room_server/worker/types"
	"github.com/CryptoElementals/common/wallet"
	"github.com/ethereum/go-ethereum/ethclient"
)

// chainRuntime is one L2 + RoomV3 client used for submissions.
type chainRuntime struct {
	chainID               int64
	roomV3Client          *concurrentRoomV3Client
	roomV2ContractAddress string
}

func newChainRuntime(
	ctx context.Context,
	chainID int64,
	client *ethclient.Client,
	roomV3ContractAddressHex string,
	wallets []*wallet.Wallet,
	isDevelop ...bool,
) (*chainRuntime, error) {
	if roomV3ContractAddressHex == "" {
		return nil, errors.New("room contract address is required")
	}
	roomV3Cli, err := newConcurrentRoomV3Client(ctx, client, roomV3ContractAddressHex, wallets, chainID, isDevelop...)
	if err != nil {
		log.Errorf("newConcurrentRoomV3Client: create room v3 client failed: %s", err.Error())
		return nil, err
	}
	return &chainRuntime{
		chainID:               chainID,
		roomV3Client:          roomV3Cli,
		roomV2ContractAddress: strings.ToLower(roomV3ContractAddressHex),
	}, nil
}

// SubmitTasks submits a batch of tasks to this chain's RoomV3 contract.
func (r *chainRuntime) SubmitTasks(tasks []types.RoomContractTask) error {
	if r.roomV3Client == nil {
		return errors.New("room v3 client not initialized")
	}
	if len(tasks) == 0 {
		return nil
	}
	indexes := make([]uint8, 0, len(tasks))
	payloads := make([][]byte, 0, len(tasks))
	for _, t := range tasks {
		indexes = append(indexes, t.Index)
		payloads = append(payloads, t.Task)
	}
	_, err := r.roomV3Client.submitTasks(indexes, payloads)
	return err
}

// Chain is the room server chain facade: multiple L2 clients, tx pools, and game→chain routing.
type Chain struct {
	ctx context.Context

	runtimes map[int64]*chainRuntime
	pools    map[int64]*txPool

	chainIDs []int64 // configured chain ids (for random pick)

	poolBatchSize          int
	poolProcessingInterval int
	poolTickerDur          time.Duration
}

// NewChain builds chain runtimes from room server config (EffectiveChains).
func NewChain(
	ctx context.Context,
	cfg *config.RoomServerConfig,
	wallets []*wallet.Wallet,
	isDevelop ...bool,
) (*Chain, error) {
	entries := cfg.EffectiveChains()
	if len(entries) == 0 {
		return nil, errors.New("no chains configured (set chains[] or legacy chain)")
	}
	batch := cfg.PoolBatchSize
	interval := cfg.PoolProcessingInterval
	h := &Chain{
		ctx:                    ctx,
		runtimes:               make(map[int64]*chainRuntime),
		pools:                  make(map[int64]*txPool),
		poolBatchSize:          batch,
		poolProcessingInterval: interval,
	}
	for _, e := range entries {
		if e.HttpRpc == "" {
			return nil, errors.New("chain http-rpc is required for each chain")
		}
		client, err := ethclient.DialContext(ctx, e.HttpRpc)
		if err != nil {
			return nil, err
		}
		chainID := e.ChainID
		if chainID == 0 {
			cid, err := client.ChainID(ctx)
			if err != nil {
				return nil, err
			}
			chainID = cid.Int64()
		}
		if _, dup := h.runtimes[chainID]; dup {
			return nil, errors.New("duplicate chain id in configuration")
		}
		rt, err := newChainRuntime(ctx, chainID, client, e.RoomV3ContractAddress, wallets, isDevelop...)
		if err != nil {
			return nil, err
		}
		h.runtimes[chainID] = rt
		h.chainIDs = append(h.chainIDs, chainID)
		p := newTxPool(rt, batch)
		h.pools[chainID] = p
	}
	h.poolTickerDur = time.Duration(interval) * time.Second
	if h.poolTickerDur <= 0 {
		h.poolTickerDur = time.Second
	}
	return h, nil
}

// PickChainIDForNewGame returns a random configured chain id.
func (h *Chain) PickChainIDForNewGame() int64 {
	if len(h.chainIDs) == 1 {
		return h.chainIDs[0]
	}
	n, err := rand.Int(rand.Reader, big.NewInt(int64(len(h.chainIDs))))
	if err != nil {
		return h.chainIDs[0]
	}
	return h.chainIDs[int(n.Int64())]
}

// AddCreateRoom picks a random configured chain, persists game_chain_ids, then enqueues for that chain's pool.
func (h *Chain) AddCreateRoom(evt *types.RequireGameCreationEvent) {
	if evt == nil {
		return
	}
	chainID := h.PickChainIDForNewGame()
	if err := db.SaveGameChainIDForGame(evt.GameID, chainID); err != nil {
		log.Errorw("AddCreateRoom: save game_chain_ids", "gameID", evt.GameID, "chain_id", chainID, "err", err)
		return
	}
	p, ok := h.pools[chainID]
	if !ok {
		log.Errorw("AddCreateRoom: no tx pool for chain", "chain_id", chainID)
		return
	}
	p.addCreateRoom(evt)
}

// Start registers the room timer handler and a periodic Asynq cron that enqueues the tx pool tick.
func (h *Chain) Start() error {
	if err := h.registerTxPoolTimerHandler(); err != nil {
		return err
	}
	return h.registerChainTxPoolPeriodic(&chainTxPoolTickEvent{})
}

func (h *Chain) runAllPoolTicks() {
	for chainID, p := range h.pools {
		if err := p.runDrainLoop(); err != nil {
			log.Errorw("runDrainLoop", "err", err, "chain_id", chainID)
		}
	}
}
