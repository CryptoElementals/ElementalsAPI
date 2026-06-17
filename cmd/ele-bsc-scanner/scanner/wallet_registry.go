package scanner

import (
	"context"
	"fmt"
	"math/big"
	"strings"
	"sync"
	"time"

	"github.com/CryptoElementals/common/db"
	"github.com/CryptoElementals/common/internal/evmrpc"
	contract "github.com/CryptoElementals/common/contracts"
	"github.com/CryptoElementals/common/log"
	dao "github.com/CryptoElementals/common/models"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
)

type WalletRegistry struct {
	mu       sync.RWMutex
	chainID  uint64
	addrs    map[common.Address]struct{}
	manager  common.Address
	httpRPC  string
	client   *ethclient.Client
	filterer *contract.WalletManagerContractFilterer
	clientMu sync.Mutex
}

func NewWalletRegistry(chainID uint64, httpRPC, walletManagerAddr string) (*WalletRegistry, error) {
	if walletManagerAddr == "" {
		return nil, fmt.Errorf("wallet-manager-address is required")
	}
	return &WalletRegistry{
		chainID: chainID,
		addrs:   make(map[common.Address]struct{}),
		manager: common.HexToAddress(walletManagerAddr),
		httpRPC: httpRPC,
	}, nil
}

func (r *WalletRegistry) ManagerAddress() common.Address {
	return r.manager
}

func (r *WalletRegistry) getClient(ctx context.Context) (*ethclient.Client, error) {
	r.clientMu.Lock()
	defer r.clientMu.Unlock()
	if r.client != nil {
		return r.client, nil
	}
	c, err := ethclient.DialContext(ctx, r.httpRPC)
	if err != nil {
		return nil, err
	}
	filterer, err := contract.NewWalletManagerContractFilterer(r.manager, c)
	if err != nil {
		c.Close()
		return nil, err
	}
	r.client = c
	r.filterer = filterer
	return r.client, nil
}

func (r *WalletRegistry) ensureFilterer(ctx context.Context) error {
	if r.filterer != nil {
		return nil
	}
	_, err := r.getClient(ctx)
	return err
}

func (r *WalletRegistry) LoadFromDB() error {
	rows, err := db.ListTokenCollectorAddresses(r.chainID)
	if err != nil {
		return err
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, row := range rows {
		r.addrs[common.HexToAddress(row.Address)] = struct{}{}
	}
	log.Infof("WalletRegistry loaded %d addresses from DB for chainID=%d", len(rows), r.chainID)
	return nil
}

func (r *WalletRegistry) Contains(addr common.Address) bool {
	r.mu.RLock()
	defer r.mu.RUnlock()
	_, ok := r.addrs[addr]
	return ok
}

func (r *WalletRegistry) Count() int {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return len(r.addrs)
}

func (r *WalletRegistry) addToMemory(addr common.Address) bool {
	if addr == (common.Address{}) {
		return false
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.addrs[addr]; ok {
		return false
	}
	r.addrs[addr] = struct{}{}
	return true
}

func (r *WalletRegistry) upsert(addr common.Address, walletIndex *uint64, source string, blockNumber *uint64) (bool, error) {
	if addr == (common.Address{}) {
		return false, nil
	}
	inserted, err := db.UpsertTokenCollectorAddress(dao.TokenCollectorAddress{
		ChainID:     r.chainID,
		Address:     strings.ToLower(addr.Hex()),
		WalletIndex: walletIndex,
		Source:      source,
		BlockNumber: blockNumber,
	})
	if err != nil {
		return false, err
	}
	if r.addToMemory(addr) && inserted {
		log.Infof("WalletRegistry new address chainID=%d addr=%s source=%s", r.chainID, addr.Hex(), source)
	}
	return inserted, nil
}

func (r *WalletRegistry) FullRefresh(ctx context.Context) error {
	client, err := r.getClient(ctx)
	if err != nil {
		return err
	}
	caller, err := contract.NewWalletManagerContractCaller(r.manager, client)
	if err != nil {
		return err
	}
	opts := &bind.CallOpts{Context: ctx}
	total, err := caller.TotalWallets(opts)
	if err != nil {
		return fmt.Errorf("totalWallets: %w", err)
	}

	added := 0
	for i := int64(0); i < total.Int64(); i++ {
		idx := big.NewInt(i)
		walletIndex := uint64(i)
		slot, err := caller.GetWalletSlot(opts, idx)
		if err != nil {
			return fmt.Errorf("getWalletSlot(%d): %w", i, err)
		}
		if !slot.Exists {
			continue
		}
		if inserted, err := r.upsert(slot.CurrentAddress, &walletIndex, dao.TokenCollectorSourceOnChainRefresh, nil); err != nil {
			return err
		} else if inserted {
			added++
		}
		histLen := int64(0)
		if slot.HistoryLength != nil {
			histLen = slot.HistoryLength.Int64()
		}
		for j := int64(0); j < histLen; j++ {
			addr, err := caller.GetWalletHistory(opts, idx, big.NewInt(j))
			if err != nil {
				return fmt.Errorf("getWalletHistory(%d,%d): %w", i, j, err)
			}
			if inserted, err := r.upsert(addr, &walletIndex, dao.TokenCollectorSourceHistory, nil); err != nil {
				return err
			} else if inserted {
				added++
			}
		}
	}
	log.Infof("WalletRegistry full refresh chainID=%d totalWallets=%d tracked=%d new=%d",
		r.chainID, total.Int64(), r.Count(), added)
	return nil
}

func bigintToUint64Ptr(v *big.Int) *uint64 {
	if v == nil || !v.IsUint64() {
		return nil
	}
	n := v.Uint64()
	return &n
}

func receiptLogToTypesLog(lg evmrpc.ReceiptLog) types.Log {
	topics := make([]common.Hash, len(lg.Topics))
	for i, t := range lg.Topics {
		topics[i] = common.HexToHash(t)
	}
	return types.Log{
		Address: common.HexToAddress(lg.Address),
		Topics:  topics,
		Data:    common.FromHex(lg.Data),
		TxHash:  common.HexToHash(lg.TxHash),
		Index:   uint(lg.LogIndex),
	}
}

func (r *WalletRegistry) ApplyReceiptLog(ctx context.Context, lg evmrpc.ReceiptLog, blockNumber uint64) error {
	if !strings.EqualFold(lg.Address, r.manager.Hex()) || len(lg.Topics) == 0 {
		return nil
	}
	if err := r.ensureFilterer(ctx); err != nil {
		return err
	}
	blockNum := blockNumber
	rawLog := receiptLogToTypesLog(lg)

	if ev, err := r.filterer.ParseWalletAdded(rawLog); err == nil {
		_, err := r.upsert(ev.Wallet, bigintToUint64Ptr(ev.WalletIndex), dao.TokenCollectorSourceWalletAdded, &blockNum)
		return err
	}
	if ev, err := r.filterer.ParseWalletAddressUpdated(rawLog); err == nil {
		walletIndex := bigintToUint64Ptr(ev.WalletIndex)
		if _, err := r.upsert(ev.OldAddress, walletIndex, dao.TokenCollectorSourceAddressUpdated, &blockNum); err != nil {
			return err
		}
		_, err := r.upsert(ev.NewAddress, walletIndex, dao.TokenCollectorSourceAddressUpdated, &blockNum)
		return err
	}
	return nil
}

func (r *WalletRegistry) RunPeriodicRefresh(ctx context.Context, interval time.Duration) {
	if interval <= 0 {
		return
	}
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			refreshCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
			if err := r.FullRefresh(refreshCtx); err != nil {
				log.Errorf("WalletRegistry periodic refresh failed: %v", err)
			}
			cancel()
		}
	}
}
