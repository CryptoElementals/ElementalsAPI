package scanner

import (
	"math/big"
	"strings"
	"testing"

	contract "github.com/CryptoElementals/common/contracts"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"
)

func TestDecodeWithdrawCalldata(t *testing.T) {
	parsed, err := abi.JSON(strings.NewReader(contract.TokenCollectorContractABI))
	require.NoError(t, err)

	method := parsed.Methods["withdraw"]
	playerID := big.NewInt(42)
	amount := big.NewInt(1_000_000_000_000_000_000)
	sig := make([]byte, 65)
	for i := range sig {
		sig[i] = byte(i + 1)
	}

	input, err := method.Inputs.Pack(playerID, amount, sig)
	require.NoError(t, err)
	calldata := append(method.ID, input...)

	require.Equal(t, int64(42), decodeWithdrawPlayerID(&parsed, common.Bytes2Hex(calldata)))
}

func TestTokenProcessorEventTopics(t *testing.T) {
	reg, err := NewWalletRegistry(97, "http://localhost:8545", "0xFFD251cBd389e482B0609D3B6389a1350827A6C2")
	require.NoError(t, err)
	p, err := NewTokenProcessor(97, reg)
	require.NoError(t, err)
	require.NotEqual(t, common.Hash{}, p.depositedTopic)
	require.NotEqual(t, common.Hash{}, p.withdrawnTopic)
}
