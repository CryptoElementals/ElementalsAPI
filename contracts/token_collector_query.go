package contract

import (
	"context"
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
)

var withdrawNonceMethodID = crypto.Keccak256([]byte("withdrawNonce(uint256)"))[:4]

// QueryWithdrawNonce reads TokenCollector.withdrawNonce(playerId) via eth_call.
func QueryWithdrawNonce(ctx context.Context, caller bind.ContractCaller, tokenCollector common.Address, playerID *big.Int) (*big.Int, error) {
	if caller == nil {
		return nil, fmt.Errorf("contract caller is nil")
	}
	if playerID == nil {
		return nil, fmt.Errorf("playerID is nil")
	}
	if tokenCollector == (common.Address{}) {
		return nil, fmt.Errorf("token collector address is zero")
	}

	data := make([]byte, 0, 4+32)
	data = append(data, withdrawNonceMethodID...)
	data = append(data, common.LeftPadBytes(playerID.Bytes(), 32)...)

	out, err := caller.CallContract(ctx, ethereum.CallMsg{To: &tokenCollector, Data: data}, nil)
	if err != nil {
		return nil, fmt.Errorf("withdrawNonce call: %w", err)
	}
	if len(out) == 0 {
		return big.NewInt(0), nil
	}
	return new(big.Int).SetBytes(out), nil
}

// ResolvePlayerTokenCollectorAddress returns the active TokenCollector for playerId via WalletManager.
func ResolvePlayerTokenCollectorAddress(ctx context.Context, caller bind.ContractCaller, walletManager common.Address, playerID *big.Int) (common.Address, error) {
	if caller == nil {
		return common.Address{}, fmt.Errorf("contract caller is nil")
	}
	if playerID == nil {
		return common.Address{}, fmt.Errorf("playerID is nil")
	}
	if walletManager == (common.Address{}) {
		return common.Address{}, fmt.Errorf("wallet manager address is zero")
	}

	wm, err := NewWalletManagerContractCaller(walletManager, caller)
	if err != nil {
		return common.Address{}, fmt.Errorf("create WalletManager client: %w", err)
	}

	opts := &bind.CallOpts{Context: ctx}
	walletIndex, err := wm.GetWalletIndexForPlayerId(opts, playerID)
	if err != nil {
		return common.Address{}, fmt.Errorf("getWalletIndexForPlayerId: %w", err)
	}

	slot, err := wm.GetWalletSlot(opts, walletIndex)
	if err != nil {
		return common.Address{}, fmt.Errorf("getWalletSlot(%s): %w", walletIndex.String(), err)
	}
	if !slot.Exists {
		return common.Address{}, fmt.Errorf("wallet slot %s does not exist for playerId=%s", walletIndex.String(), playerID.String())
	}
	if !slot.IsActive {
		return common.Address{}, fmt.Errorf("wallet slot %s is not active for playerId=%s", walletIndex.String(), playerID.String())
	}
	if slot.CurrentAddress == (common.Address{}) {
		return common.Address{}, fmt.Errorf("wallet slot %s has no current address for playerId=%s", walletIndex.String(), playerID.String())
	}

	return slot.CurrentAddress, nil
}
