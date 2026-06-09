package worker

import (
	"context"
	"errors"
	"fmt"
	"math/big"
	"sort"
	"strings"

	"github.com/CryptoElementals/common/config"
	contract "github.com/CryptoElementals/common/contracts"
	"github.com/CryptoElementals/common/db"
	"github.com/CryptoElementals/common/log"
	dao "github.com/CryptoElementals/common/models"
	"github.com/CryptoElementals/common/wallet"
	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi/bind/v2"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
)

const (
	batchWithdrawGasLimit         = 1_000_000
	batchWithdrawGasBufferPercent = 110
)

type walletRuntime struct {
	client        *ethclient.Client
	chainID       int64
	walletManager *contract.WalletManagerContract
	optsPool      chan *bind.TransactOpts
}

func newWalletRuntime(
	ctx context.Context,
	cfg *config.WalletChainConfig,
	wallets []*wallet.Wallet,
	isDevelop ...bool,
) (*walletRuntime, error) {
	if cfg == nil {
		return nil, errors.New("wallet-chain config is required")
	}
	if cfg.HttpRpc == "" {
		return nil, errors.New("wallet-chain http-rpc is required")
	}
	if cfg.WalletManagerAddress == "" {
		return nil, errors.New("wallet-manager-address is required")
	}
	if len(wallets) == 0 {
		return nil, errors.New("at least one wallet is required for wallet-chain")
	}

	client, err := ethclient.DialContext(ctx, cfg.HttpRpc)
	if err != nil {
		return nil, err
	}

	chainID := cfg.ChainID
	if chainID == 0 {
		cid, err := client.ChainID(ctx)
		if err != nil {
			return nil, err
		}
		chainID = cid.Int64()
	}

	wm, err := contract.NewWalletManagerContract(common.HexToAddress(cfg.WalletManagerAddress), client)
	if err != nil {
		return nil, fmt.Errorf("new wallet manager contract: %w", err)
	}

	optsPool := make(chan *bind.TransactOpts, len(wallets))
	for _, w := range wallets {
		nonce, err := client.PendingNonceAt(ctx, w.GetAddr())
		if err != nil {
			return nil, fmt.Errorf("pending nonce for wallet %s: %w", w.GetAddr().Hex(), err)
		}
		bindOpts := &bind.TransactOpts{
			Context:  ctx,
			From:     w.GetAddr(),
			Signer:   w.BuildTxSinger(big.NewInt(chainID)),
			GasLimit: batchWithdrawGasLimit,
			Nonce:    new(big.Int).SetUint64(nonce),
		}
		if len(isDevelop) != 0 && isDevelop[0] {
			bindOpts.NoSend = true
		}
		optsPool <- bindOpts
	}

	return &walletRuntime{
		client:        client,
		chainID:       chainID,
		walletManager: wm,
		optsPool:      optsPool,
	}, nil
}

type BatchWithdrawItem struct {
	PlayerID  int64
	Amount    int64
	Signature []byte
}

type BatchWithdrawResult struct {
	TxHash           string
	LedgerID         uint64
	CollectorAddress string
}

type resolvedBatchWithdrawItem struct {
	playerID  int64
	amount    *big.Int
	signature []byte
	collector common.Address
}

func (r *walletRuntime) BatchWithdraw(ctx context.Context, items []BatchWithdrawItem) ([]BatchWithdrawResult, error) {
	if len(items) == 0 {
		return nil, errors.New("items is required")
	}

	resolved := make([]resolvedBatchWithdrawItem, 0, len(items))
	for _, item := range items {
		parsed, err := r.resolveBatchWithdrawItem(ctx, item)
		if err != nil {
			return nil, err
		}
		resolved = append(resolved, parsed)
	}

	groups := groupBatchWithdrawItems(resolved)
	collectorAddrs := make([]string, 0, len(groups))
	for addr := range groups {
		collectorAddrs = append(collectorAddrs, addr)
	}
	sort.Strings(collectorAddrs)

	results := make([]BatchWithdrawResult, 0, len(resolved))
	for _, collectorAddr := range collectorAddrs {
		group := groups[collectorAddr]
		groupResults, err := r.batchWithdrawGroup(ctx, group)
		if err != nil {
			if len(results) > 0 {
				log.Errorw("batch withdraw partial failure",
					"completed", results,
					"failed_collector", collectorAddr,
					"err", err,
				)
			}
			return nil, err
		}
		results = append(results, groupResults...)
	}
	return results, nil
}

func (r *walletRuntime) resolveBatchWithdrawItem(ctx context.Context, item BatchWithdrawItem) (resolvedBatchWithdrawItem, error) {
	if item.PlayerID <= 0 {
		return resolvedBatchWithdrawItem{}, fmt.Errorf("invalid player_id: %d", item.PlayerID)
	}
	if item.Amount <= 0 {
		return resolvedBatchWithdrawItem{}, fmt.Errorf("invalid amount for player %d: %d", item.PlayerID, item.Amount)
	}
	amount := big.NewInt(item.Amount)
	if len(item.Signature) == 0 {
		return resolvedBatchWithdrawItem{}, fmt.Errorf("signature is required for player %d", item.PlayerID)
	}

	callOpts := &bind.CallOpts{Context: ctx}
	playerIDBig := big.NewInt(item.PlayerID)

	walletIndex, err := r.walletManager.GetWalletIndexForPlayerId(callOpts, playerIDBig)
	if err != nil {
		return resolvedBatchWithdrawItem{}, fmt.Errorf("get wallet index for player %d: %w", item.PlayerID, err)
	}

	slot, err := r.walletManager.GetWalletSlot(callOpts, walletIndex)
	if err != nil {
		return resolvedBatchWithdrawItem{}, fmt.Errorf("get wallet slot for player %d: %w", item.PlayerID, err)
	}
	if !slot.Exists {
		return resolvedBatchWithdrawItem{}, fmt.Errorf("wallet slot %s does not exist for player %d", walletIndex.String(), item.PlayerID)
	}
	if !slot.IsActive {
		return resolvedBatchWithdrawItem{}, fmt.Errorf("wallet slot %s is not active for player %d", walletIndex.String(), item.PlayerID)
	}
	if slot.CurrentAddress == (common.Address{}) {
		return resolvedBatchWithdrawItem{}, fmt.Errorf("wallet slot %s has no current address for player %d", walletIndex.String(), item.PlayerID)
	}

	return resolvedBatchWithdrawItem{
		playerID:  item.PlayerID,
		amount:    amount,
		signature: item.Signature,
		collector: slot.CurrentAddress,
	}, nil
}

func groupBatchWithdrawItems(items []resolvedBatchWithdrawItem) map[string][]resolvedBatchWithdrawItem {
	groups := make(map[string][]resolvedBatchWithdrawItem)
	for _, item := range items {
		key := strings.ToLower(item.collector.Hex())
		groups[key] = append(groups[key], item)
	}
	return groups
}

func (r *walletRuntime) batchWithdrawGroup(ctx context.Context, group []resolvedBatchWithdrawItem) ([]BatchWithdrawResult, error) {
	collector := group[0].collector
	playerIDs := make([]*big.Int, len(group))
	amounts := make([]*big.Int, len(group))
	signatures := make([][]byte, len(group))

	for i, item := range group {
		playerIDs[i] = big.NewInt(item.playerID)
		amounts[i] = item.amount
		signatures[i] = item.signature
	}

	tc, err := contract.NewTokenCollectorContract(collector, r.client)
	if err != nil {
		return nil, fmt.Errorf("new token collector contract: %w", err)
	}

	bindOpts := <-r.optsPool
	var sendErr error
	defer func() {
		if sendErr == nil && !bindOpts.NoSend && bindOpts.Nonce != nil {
			bindOpts.Nonce = new(big.Int).Add(bindOpts.Nonce, big.NewInt(1))
		}
		r.optsPool <- bindOpts
	}()

	estimatedGas, err := estimateBatchWithdrawGas(ctx, r.client, bindOpts.From, collector, playerIDs, amounts, signatures)
	if err != nil {
		log.Errorw("estimate batch withdraw gas", "collector", collector.Hex(), "err", err)
		return nil, fmt.Errorf("estimate batch withdraw gas: %w", err)
	}
	bindOpts.GasLimit = gasLimitWithBuffer(estimatedGas)

	tx, err := tc.BatchWithdraw(bindOpts, playerIDs, amounts, signatures)
	sendErr = err
	if err != nil {
		log.Errorw("batch withdraw tx", "collector", collector.Hex(), "err", err)
		return nil, fmt.Errorf("batch withdraw: %w", err)
	}

	txHash := strings.ToLower(tx.Hash().String())
	collectorHex := strings.ToLower(collector.Hex())
	results := make([]BatchWithdrawResult, 0, len(group))
	for _, item := range group {
		ledgerID, err := db.InsertBatchWithdrawLedger(&dao.BatchWithdrawLedger{
			PlayerID:         item.playerID,
			Amount:           item.amount.Int64(),
			Signature:        db.FormatWithdrawSignatureHex(item.signature),
			CollectorAddress: collectorHex,
			ChainID:          r.chainID,
			TxHash:           txHash,
		})
		if err != nil {
			log.Errorw("insert batch withdraw ledger",
				"collector", collectorHex,
				"player_id", item.playerID,
				"tx_hash", txHash,
				"err", err,
			)
			return nil, fmt.Errorf("insert batch withdraw ledger: %w", err)
		}
		results = append(results, BatchWithdrawResult{
			TxHash:           txHash,
			LedgerID:         uint64(ledgerID),
			CollectorAddress: collectorHex,
		})
	}
	return results, nil
}

func estimateBatchWithdrawGas(
	ctx context.Context,
	client *ethclient.Client,
	from common.Address,
	collector common.Address,
	playerIds []*big.Int,
	amounts []*big.Int,
	signatures [][]byte,
) (uint64, error) {
	parsed, err := contract.TokenCollectorContractMetaData.GetAbi()
	if err != nil {
		return 0, fmt.Errorf("load token collector abi: %w", err)
	}
	data, err := parsed.Pack("batchWithdraw", playerIds, amounts, signatures)
	if err != nil {
		return 0, fmt.Errorf("pack batchWithdraw calldata: %w", err)
	}

	gas, err := client.EstimateGas(ctx, ethereum.CallMsg{
		From: from,
		To:   &collector,
		Data: data,
	})
	if err != nil {
		return 0, fmt.Errorf("estimate batchWithdraw gas: %w", err)
	}
	return gas, nil
}

func gasLimitWithBuffer(estimated uint64) uint64 {
	if estimated == 0 {
		return batchWithdrawGasLimit
	}
	buffered := estimated * batchWithdrawGasBufferPercent / 100
	if buffered > batchWithdrawGasLimit {
		return batchWithdrawGasLimit
	}
	return buffered
}
