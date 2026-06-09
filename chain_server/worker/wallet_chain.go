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
	dao "github.com/CryptoElementals/common/models"
	"github.com/CryptoElementals/common/wallet"
	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi/bind/v2"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
)

const (
	tokenCollectGasLimit         = 300_000
	tokenCollectGasBufferPercent = 110
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
			GasLimit: tokenCollectGasLimit,
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

type CollectTokenResult struct {
	TxHash   string
	LedgerID uint64
}

func (r *walletRuntime) CollectToken(ctx context.Context, playerID int64, playerAddr string, tokenAmount string) (*CollectTokenResult, error) {
	if r == nil {
		return nil, errors.New("wallet runtime not configured")
	}
	if playerID <= 0 {
		return nil, fmt.Errorf("invalid player_id: %d", playerID)
	}
	playerAddr = strings.TrimSpace(playerAddr)
	if !common.IsHexAddress(playerAddr) {
		return nil, fmt.Errorf("invalid player_address: %q", playerAddr)
	}
	toAddress := common.HexToAddress(playerAddr)

	amount := new(big.Int)
	if _, ok := amount.SetString(strings.TrimSpace(tokenAmount), 10); !ok || amount.Sign() <= 0 {
		return nil, fmt.Errorf("invalid token_amount: %q", tokenAmount)
	}

	callOpts := &bind.CallOpts{Context: ctx}
	playerIDBig := big.NewInt(playerID)

	walletIndex, err := r.walletManager.GetWalletIndexForPlayerId(callOpts, playerIDBig)
	if err != nil {
		return nil, fmt.Errorf("get wallet index for player: %w", err)
	}

	slot, err := r.walletManager.GetWalletSlot(callOpts, walletIndex)
	if err != nil {
		return nil, fmt.Errorf("get wallet slot: %w", err)
	}
	if !slot.Exists {
		return nil, fmt.Errorf("wallet slot %s does not exist", walletIndex.String())
	}
	if !slot.IsActive {
		return nil, fmt.Errorf("wallet slot %s is not active", walletIndex.String())
	}
	if slot.CurrentAddress == (common.Address{}) {
		return nil, fmt.Errorf("wallet slot %s has no current address", walletIndex.String())
	}

	tc, err := contract.NewTokenCollectorContract(slot.CurrentAddress, r.client)
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

	estimatedGas, err := estimateTokenCollectGas(ctx, r.client, bindOpts.From, slot.CurrentAddress, toAddress, amount)
	if err != nil {
		return nil, err
	}
	bindOpts.GasLimit = gasLimitWithBuffer(estimatedGas)

	tx, err := tc.Collect(bindOpts, toAddress, amount)
	sendErr = err
	if err != nil {
		return nil, fmt.Errorf("collect token: %w", err)
	}

	txHash := strings.ToLower(tx.Hash().String())
	ledgerID, err := db.InsertTokenCollectLedger(&dao.TokenCollectLedger{
		PlayerID:         playerID,
		PlayerAddress:    toAddress.Hex(),
		WalletIndex:      walletIndex.Uint64(),
		CollectorAddress: slot.CurrentAddress.Hex(),
		TokenAmount:      amount.String(),
		TxHash:           txHash,
		ChainID:          r.chainID,
	})
	if err != nil {
		return nil, fmt.Errorf("insert token collect ledger: %w", err)
	}

	return &CollectTokenResult{
		TxHash:   txHash,
		LedgerID: uint64(ledgerID),
	}, nil
}

func estimateTokenCollectGas(
	ctx context.Context,
	client *ethclient.Client,
	from common.Address,
	collector common.Address,
	to common.Address,
	amount *big.Int,
) (uint64, error) {
	parsed, err := contract.TokenCollectorContractMetaData.GetAbi()
	if err != nil {
		return 0, fmt.Errorf("load token collector abi: %w", err)
	}
	data, err := parsed.Pack("collect", to, amount)
	if err != nil {
		return 0, fmt.Errorf("pack collect calldata: %w", err)
	}

	collectorAddr := collector
	gas, err := client.EstimateGas(ctx, ethereum.CallMsg{
		From: from,
		To:   &collectorAddr,
		Data: data,
	})
	if err != nil {
		return 0, fmt.Errorf("estimate collect gas: %w", err)
	}
	return gas, nil
}

func gasLimitWithBuffer(estimated uint64) uint64 {
	if estimated == 0 {
		return tokenCollectGasLimit
	}
	buffered := estimated * tokenCollectGasBufferPercent / 100
	if buffered < estimated {
		buffered = estimated + estimated/10
	}
	if buffered > tokenCollectGasLimit {
		return tokenCollectGasLimit
	}
	return buffered
}
