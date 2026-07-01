package worker

import (
	"context"
	"errors"
	"fmt"
	"math/big"
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
	withdrawGasLimit         = 300_000
	withdrawGasBufferPercent = 110
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
			GasLimit: withdrawGasLimit,
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

type WithdrawResult struct {
	TxHash           string
	LedgerID         uint64
	CollectorAddress string
}

type resolvedWithdrawItem struct {
	playerID  int64
	amount    *big.Int
	signature []byte
	collector common.Address
}

func (r *walletRuntime) Withdraw(ctx context.Context, playerID int64, amountWei string, signature []byte) (*WithdrawResult, error) {
	item, err := r.resolveWithdrawItem(ctx, playerID, amountWei, signature)
	if err != nil {
		return nil, err
	}

	tc, err := contract.NewTokenCollectorContract(item.collector, r.client)
	if err != nil {
		return nil, fmt.Errorf("new token collector contract: %w", err)
	}

	bindOpts := <-r.optsPool
	var txSent bool
	defer func() {
		if txSent && !bindOpts.NoSend && bindOpts.Nonce != nil {
			bindOpts.Nonce = new(big.Int).Add(bindOpts.Nonce, big.NewInt(1))
		}
		r.optsPool <- bindOpts
	}()

	estimatedGas, err := estimateWithdrawGas(ctx, r.client, bindOpts.From, item.collector, big.NewInt(item.playerID), item.amount, item.signature)
	if err != nil {
		log.Errorw("estimate withdraw gas",
			"collector", item.collector.Hex(),
			"player_id", item.playerID,
			"signature", db.FormatWithdrawSignatureHex(item.signature),
			"err", err)
		return nil, fmt.Errorf("estimate withdraw gas: %w", err)
	}
	bindOpts.GasLimit = gasLimitWithBuffer(estimatedGas)

	tx, err := tc.Withdraw(bindOpts, big.NewInt(item.playerID), item.amount, item.signature)
	if err != nil {
		log.Errorw("withdraw tx", "collector", item.collector.Hex(), "player_id", item.playerID, "err", err)
		return nil, fmt.Errorf("withdraw: %w", err)
	}
	txSent = true

	txHash := strings.ToLower(tx.Hash().String())
	collectorHex := strings.ToLower(item.collector.Hex())
	ledgerID, err := db.InsertWithdrawLedger(&dao.WithdrawLedger{
		PlayerID:         item.playerID,
		Amount:           item.amount.String(),
		Signature:        db.FormatWithdrawSignatureHex(item.signature),
		CollectorAddress: collectorHex,
		ChainID:          r.chainID,
		TxHash:           txHash,
	})
	if err != nil {
		log.Errorw("insert withdraw ledger",
			"collector", collectorHex,
			"player_id", item.playerID,
			"tx_hash", txHash,
			"err", err,
		)
		return nil, fmt.Errorf("insert withdraw ledger: %w", err)
	}
	return &WithdrawResult{
		TxHash:           txHash,
		LedgerID:         uint64(ledgerID),
		CollectorAddress: collectorHex,
	}, nil
}

func (r *walletRuntime) resolveWithdrawItem(ctx context.Context, playerID int64, amountWei string, signature []byte) (resolvedWithdrawItem, error) {
	if playerID <= 0 {
		return resolvedWithdrawItem{}, fmt.Errorf("invalid player_id: %d", playerID)
	}
	amountWei = strings.TrimSpace(amountWei)
	if amountWei == "" {
		return resolvedWithdrawItem{}, fmt.Errorf("invalid amount for player %d: amount_wei is required", playerID)
	}
	amountBigInt, ok := new(big.Int).SetString(amountWei, 10)
	if !ok || amountBigInt.Sign() <= 0 {
		return resolvedWithdrawItem{}, fmt.Errorf("invalid amount for player %d: %q", playerID, amountWei)
	}
	if len(signature) == 0 {
		return resolvedWithdrawItem{}, fmt.Errorf("signature is required for player %d", playerID)
	}

	callOpts := &bind.CallOpts{Context: ctx}
	playerIDBig := big.NewInt(playerID)

	walletIndex, err := r.walletManager.GetWalletIndexForPlayerId(callOpts, playerIDBig)
	if err != nil {
		return resolvedWithdrawItem{}, fmt.Errorf("get wallet index for player %d: %w", playerID, err)
	}

	slot, err := r.walletManager.GetWalletSlot(callOpts, walletIndex)
	if err != nil {
		return resolvedWithdrawItem{}, fmt.Errorf("get wallet slot for player %d: %w", playerID, err)
	}
	if !slot.Exists {
		return resolvedWithdrawItem{}, fmt.Errorf("wallet slot %s does not exist for player %d", walletIndex.String(), playerID)
	}
	if !slot.IsActive {
		return resolvedWithdrawItem{}, fmt.Errorf("wallet slot %s is not active for player %d", walletIndex.String(), playerID)
	}
	if slot.CurrentAddress == (common.Address{}) {
		return resolvedWithdrawItem{}, fmt.Errorf("wallet slot %s has no current address for player %d", walletIndex.String(), playerID)
	}

	return resolvedWithdrawItem{
		playerID:  playerID,
		amount:    amountBigInt,
		signature: signature,
		collector: slot.CurrentAddress,
	}, nil
}

func estimateWithdrawGas(
	ctx context.Context,
	client *ethclient.Client,
	from common.Address,
	collector common.Address,
	playerID *big.Int,
	amount *big.Int,
	signature []byte,
) (uint64, error) {
	parsed, err := contract.TokenCollectorContractMetaData.GetAbi()
	if err != nil {
		return 0, fmt.Errorf("load token collector abi: %w", err)
	}
	data, err := parsed.Pack("withdraw", playerID, amount, signature)
	if err != nil {
		return 0, fmt.Errorf("pack withdraw calldata: %w", err)
	}

	gas, err := client.EstimateGas(ctx, ethereum.CallMsg{
		From: from,
		To:   &collector,
		Data: data,
	})
	if err != nil {
		return 0, fmt.Errorf("estimate withdraw gas: %w", err)
	}
	return gas, nil
}

func gasLimitWithBuffer(estimated uint64) uint64 {
	if estimated == 0 {
		return withdrawGasLimit
	}
	buffered := estimated * withdrawGasBufferPercent / 100
	if buffered > withdrawGasLimit {
		return withdrawGasLimit
	}
	return buffered
}
