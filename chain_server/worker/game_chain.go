package worker

import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"math/big"
	"strings"
	"time"

	"github.com/CryptoElementals/common/config"
	"github.com/CryptoElementals/common/db"
	"github.com/CryptoElementals/common/log"
	"github.com/CryptoElementals/common/rpc/proto"
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
func (r *chainRuntime) SubmitTasks(tasks []RoomContractTask) error {
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

// Chain manages multiple L2 clients, tx pools, and game→chain routing.
type Chain struct {
	ctx context.Context

	runtimes map[int64]*chainRuntime
	pools    map[int64]*txPool

	chainIDs []int64

	poolBatchSize          int
	poolProcessingInterval int
	poolTickerDur          time.Duration
	claimTimeout           time.Duration

	ticker *poolTicker

	walletRuntime *walletRuntime
}

func loadWallets(paths []string) ([]*wallet.Wallet, error) {
	wallets := make([]*wallet.Wallet, 0, len(paths))
	for _, path := range paths {
		w, err := wallet.LoadWallet(path)
		if err != nil {
			return nil, err
		}
		wallets = append(wallets, w)
	}
	return wallets, nil
}

// NewChain builds chain runtimes from chain server config (EffectiveChains).
func NewChain(
	ctx context.Context,
	cfg *config.ChainServerConfig,
	isDevelop ...bool,
) (*Chain, error) {
	entries := cfg.EffectiveChains()
	if len(entries) == 0 {
		return nil, errors.New("no chains configured (set chains[])")
	}
	batch := cfg.PoolBatchSize
	interval := cfg.PoolProcessingInterval
	claimTimeout := time.Duration(cfg.PoolClaimTimeoutSeconds) * time.Second
	if claimTimeout <= 0 {
		claimTimeout = db.DefaultChainTxPoolClaimTimeout
	}
	h := &Chain{
		ctx:                    ctx,
		runtimes:               make(map[int64]*chainRuntime),
		pools:                  make(map[int64]*txPool),
		poolBatchSize:          batch,
		poolProcessingInterval: interval,
		claimTimeout:           claimTimeout,
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
		if len(e.WalletPaths) == 0 {
			return nil, errors.New("wallet-paths required for each chain")
		}
		chainWallets, err := loadWallets(e.WalletPaths)
		if err != nil {
			return nil, err
		}
		rt, err := newChainRuntime(ctx, chainID, client, e.RoomV3ContractAddress, chainWallets, isDevelop...)
		if err != nil {
			return nil, err
		}
		h.runtimes[chainID] = rt
		h.chainIDs = append(h.chainIDs, chainID)
		p := newTxPool(rt, batch, claimTimeout)
		h.pools[chainID] = p
	}
	h.poolTickerDur = time.Duration(interval) * time.Second
	if h.poolTickerDur <= 0 {
		h.poolTickerDur = time.Second
	}
	h.ticker = newPoolTicker(ctx, h, h.poolTickerDur)

	if cfg.WalletChain != nil {
		if len(cfg.WalletChain.WalletPaths) == 0 {
			return nil, errors.New("wallet-paths required for wallet-chain")
		}
		walletChainWallets, err := loadWallets(cfg.WalletChain.WalletPaths)
		if err != nil {
			return nil, err
		}
		wr, err := newWalletRuntime(ctx, cfg.WalletChain, walletChainWallets, isDevelop...)
		if err != nil {
			return nil, fmt.Errorf("init wallet runtime: %w", err)
		}
		h.walletRuntime = wr
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
func (h *Chain) AddCreateRoom(evt *RequireGameCreationEvent) {
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

// Start runs the in-process pool ticker.
func (h *Chain) Start() error {
	h.ticker.start()
	return nil
}

// Stop stops the pool ticker.
func (h *Chain) Stop() {
	if h.ticker != nil {
		h.ticker.stop()
	}
}

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

// AddSetTurnReady enqueues a set-turn-ready task for the game's chain.
func (h *Chain) AddSetTurnReady(evt *RequireSetupNewTurnEvent) {
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

// AddCommitment enqueues a commitment submission for the game's chain.
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

// AddCard enqueues a card submission for the game's chain.
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

var ErrWalletChainNotConfigured = errors.New("wallet-chain not configured")

func (h *Chain) BatchWithdraw(ctx context.Context, items []BatchWithdrawItem) ([]BatchWithdrawResult, error) {
	if h.walletRuntime == nil {
		return nil, ErrWalletChainNotConfigured
	}
	return h.walletRuntime.BatchWithdraw(ctx, items)
}

// ClearGameInfo removes pending tx pool rows for a finished game.
func (h *Chain) ClearGameInfo(gameID int64) {
	if err := db.DeleteChainTxPoolItemsForGame(gameID); err != nil {
		log.Errorw("ClearGameInfo: delete chain tx pool rows", "gameID", gameID, "err", err)
	}
}

func (h *Chain) runAllPoolTicks() {
	cutoff := time.Now().Add(-h.claimTimeout)
	if err := db.ReleaseStaleChainTxPoolClaims(cutoff); err != nil {
		log.Errorw("ReleaseStaleChainTxPoolClaims", "err", err)
	}
	for chainID, p := range h.pools {
		if err := p.runDrainLoop(); err != nil {
			log.Errorw("runDrainLoop", "err", err, "chain_id", chainID)
		}
	}
}
