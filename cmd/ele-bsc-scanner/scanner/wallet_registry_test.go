package scanner

import (
	"math/big"
	"strings"
	"testing"

	"github.com/CryptoElementals/common/internal/evmrpc"
	contract "github.com/CryptoElementals/common/contracts"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/stretchr/testify/require"
)

func TestWalletRegistryAddAndContains(t *testing.T) {
	reg, err := NewWalletRegistry(97, "http://localhost:8545", "0xFFD251cBd389e482B0609D3B6389a1350827A6C2")
	require.NoError(t, err)
	addr := common.HexToAddress("0xcc49255a2639560171fc28b09DCd6CdC3b25597C2")
	require.True(t, reg.addToMemory(addr))
	require.True(t, reg.Contains(addr))
	require.False(t, reg.addToMemory(addr))
	require.Equal(t, 1, reg.Count())
}

func TestReceiptLogParsesWalletAddedViaContractBinding(t *testing.T) {
	parsed, err := abi.JSON(strings.NewReader(contract.WalletManagerContractABI))
	require.NoError(t, err)

	walletIndex := big.NewInt(3)
	wallet := common.HexToAddress("0xcc49255a2639560171fc28b09DCd6CdC3b25597C2")
	manager := common.HexToAddress("0xFFD251cBd389e482B0609D3B6389a1350827A6C2")

	event := parsed.Events["WalletAdded"]
	data, err := event.Inputs.NonIndexed().Pack(wallet)
	require.NoError(t, err)

	raw := types.Log{
		Address: manager,
		Topics: []common.Hash{
			event.ID,
			common.BigToHash(walletIndex),
		},
		Data: data,
	}

	filterer, err := contract.NewWalletManagerContractFilterer(manager, nil)
	require.NoError(t, err)

	lg := evmrpc.ReceiptLog{
		Address: raw.Address.Hex(),
		Topics:  []string{raw.Topics[0].Hex(), raw.Topics[1].Hex()},
		Data:    "0x" + common.Bytes2Hex(raw.Data),
	}
	ev, err := filterer.ParseWalletAdded(receiptLogToTypesLog(lg))
	require.NoError(t, err)
	require.Equal(t, wallet, ev.Wallet)
	require.Equal(t, walletIndex, ev.WalletIndex)
}
