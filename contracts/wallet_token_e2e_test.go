package contract

import (
	"context"
	"crypto/ecdsa"
	"fmt"
	"math/big"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/CryptoElementals/common/utils"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/stretchr/testify/require"
)

const (
	testRPCURL         = "https://data-seed-prebsc-1-s1.bnbchain.org:8545"
	testChainID        = 97
	testMockERC20      = "0x59554b201cFc12E6930a3631060C3d9CDF704F67"
	testWalletManager  = "0xFFD251cBd389e482B0609D3B6389a1350827A6C2"
	envPrivateKey      = "E2E_PRIVATE_KEY"
	envPrivateKeyFile  = "E2E_PRIVATE_KEY_FILE"
	envAdminKeyFile    = "E2E_ADMIN_KEY_FILE"
	envPlayerID        = "E2E_PLAYER_ID"
	envDepositAmount   = "E2E_DEPOSIT_AMOUNT"
	envWithdrawAmount  = "E2E_WITHDRAW_AMOUNT"
	envRPCURL          = "E2E_RPC_URL"
	envWalletManager   = "E2E_WALLET_MANAGER"
	envTokenCollector  = "E2E_TOKEN_COLLECTOR"
	envMockERC20       = "E2E_MOCK_ERC20"
	envToAddress       = "E2E_TO_ADDRESS"
	envERC20Address    = "E2E_ERC20_ADDRESS"
	envTransferAmount  = "E2E_TRANSFER_AMOUNT"
	tokenDecimals      = 18
)

// e2eChainAddresses holds shared contract addresses (env override or testnet defaults).
type e2eChainAddresses struct {
	rpcURL        string
	walletManager common.Address
	mockERC20     common.Address
}

// playerTokenCollector is the per-player TokenCollector resolved from WalletManager.
type playerTokenCollector struct {
	Address common.Address
	Index   *big.Int
}

/*
BSC testnet (chain 97) E2E tests: WalletManager + TokenCollector + mock ERC20.

Run from repository root. Use -count=1 to avoid cached results on live chain tests.

Deployed (testnet):
  RPC:            https://data-seed-prebsc-1-s1.bnbchain.org:8545
  MockERC20:      0x59554b201cFc12E6930a3631060C3d9CDF704F67
  WalletManager:  0xFFD251cBd389e482B0609D3B6389a1350827A6C2

Environment variables (no default key file paths):
  E2E_PRIVATE_KEY_FILE  Path to depositor/user key file (hex, with or without 0x)
  E2E_PRIVATE_KEY       Depositor key as hex string (alternative to E2E_PRIVATE_KEY_FILE)
  E2E_ADMIN_KEY_FILE    Path to TokenCollector admin key file (required for withdraw tx sender)
  E2E_RPC_URL           RPC URL (optional; default BSC testnet public node)
  E2E_WALLET_MANAGER    WalletManager address (optional)
  E2E_TOKEN_COLLECTOR   Optional override; skip WalletManager lookup when set (debug only)
  E2E_MOCK_ERC20        Mock ERC20 token address (optional)
  E2E_TO_ADDRESS        ERC20 transfer recipient (required for TestERC20Transfer)
  E2E_ERC20_ADDRESS     ERC20 token contract (optional; falls back to E2E_MOCK_ERC20)
  E2E_TRANSFER_AMOUNT   Transfer size in wei; unset uses 1 token (10^18)
  E2E_PLAYER_ID         Player id; unset defaults to 0 for deposit/withdraw
  E2E_DEPOSIT_AMOUNT    Deposit size in wei; unset uses on-chain minDepositAmount
  E2E_WITHDRAW_AMOUNT   Withdraw size in wei; unset withdraws 10 tokens (or full credited if balance < 10 tokens)

On-chain queries (no hardcoded per-player TokenCollector):
  - TokenCollector for playerId:
      WalletManager.getWalletIndexForPlayerId(playerId) -> walletIndex
      WalletManager.getWalletSlot(walletIndex).currentAddress
    (getWalletAddr(playerId) returns the same wallet address)
  - deposit allowed when: playerId % TokenCollector.totalWallets() == TokenCollector.currentWalletIndex()
    on that player's TokenCollector contract

Prerequisites:
  - User address holds enough mock USDT and has approved the resolved TokenCollector for deposit.
  - Withdraw: Credited(playerId) > 0; E2E user key must match Credited.depositAddr (signer).
  - TokenCollector must be active (setActive(true)); admin sends withdraw, depositAddr signs.

# 1) Deposit (user key only)
E2E_PRIVATE_KEY_FILE=/path/to/user_key.txt \
  go test ./contracts -run TestWalletManagerThenTokenCollectorDeposit -v -count=1

# 2) Withdraw (admin sends tx; user/depositAddr signs — run deposit first if no credited balance)
E2E_ADMIN_KEY_FILE=/path/to/admin_key.txt \
E2E_PRIVATE_KEY_FILE=/path/to/user_key.txt \
  go test ./contracts -run TestTokenCollectorWithdraw -v -count=1

# 3) Deposit + withdraw (both tests in this file)
E2E_ADMIN_KEY_FILE=/path/to/admin_key.txt \
E2E_PRIVATE_KEY_FILE=/path/to/user_key.txt \
  go test ./contracts -run 'TestWalletManagerThenTokenCollectorDeposit|TestTokenCollectorWithdraw' -v -count=1

# 4) ERC20 transfer A -> B (sender key, recipient, token contract)
E2E_PRIVATE_KEY_FILE=/path/to/sender_key.txt \
E2E_TO_ADDRESS=0xRecipientAddress \
E2E_ERC20_ADDRESS=0x59554b201cFc12E6930a3631060C3d9CDF704F67 \
  go test ./contracts -run TestERC20Transfer -v -count=1

# Optional overrides (examples)
# E2E_PLAYER_ID=1 E2E_DEPOSIT_AMOUNT=10000000000000000000 \
#   E2E_PRIVATE_KEY_FILE=/path/to/user_key.txt go test ./contracts -run TestWalletManagerThenTokenCollectorDeposit -v -count=1
# E2E_WITHDRAW_AMOUNT=10000000000000000000 \
#   E2E_ADMIN_KEY_FILE=/path/to/admin_key.txt E2E_PRIVATE_KEY_FILE=/path/to/user_key.txt \
#   go test ./contracts -run TestTokenCollectorWithdraw -v -count=1
*/

// TestWalletManagerThenTokenCollectorDeposit: approve + Deposit for a playerId.
func TestWalletManagerThenTokenCollectorDeposit(t *testing.T) {
	userPriv, signerAddr := loadUserPrivateKey(t)
	chain := loadE2EChainAddresses(t)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	client, err := ethclient.DialContext(ctx, chain.rpcURL)
	require.NoError(t, err)
	defer client.Close()

	callOpts := &bind.CallOpts{Context: ctx}

	walletManager, err := NewWalletManagerContract(chain.walletManager, client)
	require.NoError(t, err)

	playerID := resolvePlayerID(t)
	playerCollector := resolvePlayerTokenCollector(t, walletManager, callOpts, playerID)

	tokenCollector, err := NewTokenCollectorContract(playerCollector.Address, client)
	require.NoError(t, err)

	requirePlayerIDAllowedForCollector(t, playerID, playerCollector.Address, tokenCollector, callOpts)
	currentIndex, totalWallets := queryDepositWalletWindow(t, tokenCollector, callOpts)
	t.Logf("playerId=%s -> TokenCollector=%s walletIndex=%s currentWalletIndex=%s totalWallets=%s (playerId%%totalWallets=%s)",
		playerID.String(), playerCollector.Address.Hex(), playerCollector.Index.String(),
		currentIndex.String(), totalWallets.String(), playerIDWalletIndex(playerID, totalWallets).String())

	minDeposit, err := tokenCollector.MinDepositAmount(callOpts)
	require.NoError(t, err)
	maxDeposit, err := tokenCollector.MaxDepositAmount(callOpts)
	require.NoError(t, err)
	t.Logf("TokenCollector allowed deposit range: min=%s max=%s", formatWei18(minDeposit), formatWei18(maxDeposit))
	require.True(t, maxDeposit.Cmp(minDeposit) >= 0, "invalid min/max on chain")

	require.True(t, minDeposit.Sign() > 0, "on-chain minDepositAmount is 0; set E2E_DEPOSIT_AMOUNT explicitly")

	depositAmount := resolveDepositAmount(t, minDeposit)

	erc20, err := bindERC20(chain.mockERC20, client)
	require.NoError(t, err)

	balanceBefore, err := erc20BalanceOf(erc20, callOpts, signerAddr)
	require.NoError(t, err)
	t.Logf("[before] mock USDT balance of signer %s: %s", signerAddr.Hex(), formatWei18(balanceBefore))

	creditedBefore, err := tokenCollector.Credited(callOpts, playerID)
	require.NoError(t, err)
	t.Logf("[before] TokenCollector.credited(playerId): depositAddr=%s amount=%s",
		creditedBefore.DepositAddr.Hex(), formatWei18(creditedBefore.Amount))

	require.True(t, depositAmount.Cmp(minDeposit) >= 0 && depositAmount.Cmp(maxDeposit) <= 0,
		"deposit amount must be within on-chain [min,max]: amount=%s min=%s max=%s (set E2E_DEPOSIT_AMOUNT or leave unset to use min)",
		depositAmount.String(), minDeposit.String(), maxDeposit.String())
	t.Logf("using deposit amount: %s", formatWei18(depositAmount))

	require.True(t, balanceBefore.Cmp(depositAmount) >= 0,
		"signer mock USDT balance too low: have %s need %s", balanceBefore.String(), depositAmount.String())

	chainID := big.NewInt(testChainID)

	approveAuth, err := bind.NewKeyedTransactorWithChainID(userPriv, chainID)
	require.NoError(t, err)
	approveAuth.Context = ctx

	approveTx, err := erc20.Transact(approveAuth, "approve", playerCollector.Address, depositAmount)
	require.NoError(t, err)

	approveReceipt, err := bind.WaitMined(ctx, client, approveTx)
	require.NoError(t, err)
	require.NotNil(t, approveReceipt)
	require.EqualValues(t, 1, approveReceipt.Status, "approve tx failed")

	depositAuth, err := bind.NewKeyedTransactorWithChainID(userPriv, chainID)
	require.NoError(t, err)
	depositAuth.Context = ctx

	depositTx, err := tokenCollector.Deposit(depositAuth, depositAmount, playerID)
	require.NoError(t, err)

	depositReceipt, err := bind.WaitMined(ctx, client, depositTx)
	require.NoError(t, err)
	require.NotNil(t, depositReceipt)
	require.EqualValues(t, 1, depositReceipt.Status, "deposit tx failed")

	balanceAfter, err := erc20BalanceOf(erc20, callOpts, signerAddr)
	require.NoError(t, err)
	t.Logf("[after] mock USDT balance of signer %s: %s", signerAddr.Hex(), formatWei18(balanceAfter))

	creditedAfter, err := tokenCollector.Credited(callOpts, playerID)
	require.NoError(t, err)
	t.Logf("[after] TokenCollector.credited(playerId): depositAddr=%s amount=%s",
		creditedAfter.DepositAddr.Hex(), formatWei18(creditedAfter.Amount))

	expectedBal := new(big.Int).Sub(balanceBefore, depositAmount)
	require.Equal(t, 0, balanceAfter.Cmp(expectedBal),
		"signer balance should decrease by depositAmount: before=%s after=%s deposit=%s",
		balanceBefore.String(), balanceAfter.String(), depositAmount.String())

	expectedCredited := new(big.Int).Add(creditedBefore.Amount, depositAmount)
	require.Equal(t, 0, creditedAfter.Amount.Cmp(expectedCredited),
		"Credited.amount should increase by depositAmount: before=%s after=%s deposit=%s",
		creditedBefore.Amount.String(), creditedAfter.Amount.String(), depositAmount.String())

	require.True(t, creditedAfter.Amount.Sign() > 0, "credited amount should be positive")

	t.Logf("TokenCollector=%s walletIndex=%s", playerCollector.Address.Hex(), playerCollector.Index.String())
	t.Logf("Credited.depositAddr=%s signer(msg.sender)=%s", creditedAfter.DepositAddr.Hex(), signerAddr.Hex())
	require.Equal(t, signerAddr, creditedAfter.DepositAddr,
		"this test sends deposit from signer; Credited.depositAddr should equal msg.sender")
}

// TestERC20Transfer: transfer ERC20 from address A (sender private key) to address B.
//
// Required env:
//   - E2E_PRIVATE_KEY_FILE or E2E_PRIVATE_KEY: sender A private key
//   - E2E_TO_ADDRESS: recipient B address
//
// Optional env:
//   - E2E_ERC20_ADDRESS or E2E_MOCK_ERC20: ERC20 contract (default testnet mock USDT)
//   - E2E_TRANSFER_AMOUNT: amount in wei (default 1 token = 10^18)
//   - E2E_RPC_URL: RPC URL (default BSC testnet)
func TestERC20Transfer(t *testing.T) {
	senderPriv, senderAddr := loadUserPrivateKey(t)
	toAddr := loadRecipientAddress(t)
	tokenAddr := resolveERC20Address(t)
	transferAmount := resolveTransferAmount(t)
	rpcURL := getEnvOrDefault(envRPCURL, testRPCURL)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	client, err := ethclient.DialContext(ctx, rpcURL)
	require.NoError(t, err)
	defer client.Close()

	callOpts := &bind.CallOpts{Context: ctx}

	erc20, err := bindERC20(tokenAddr, client)
	require.NoError(t, err)

	t.Logf("sender A=%s recipient B=%s erc20=%s amount=%s",
		senderAddr.Hex(), toAddr.Hex(), tokenAddr.Hex(), formatWei18(transferAmount))

	senderBefore, err := erc20BalanceOf(erc20, callOpts, senderAddr)
	require.NoError(t, err)
	recipientBefore, err := erc20BalanceOf(erc20, callOpts, toAddr)
	require.NoError(t, err)
	t.Logf("[before] sender balance=%s recipient balance=%s",
		formatWei18(senderBefore), formatWei18(recipientBefore))

	require.True(t, senderBefore.Cmp(transferAmount) >= 0,
		"sender balance too low: have %s need %s",
		senderBefore.String(), transferAmount.String())

	chainID := big.NewInt(testChainID)
	auth, err := bind.NewKeyedTransactorWithChainID(senderPriv, chainID)
	require.NoError(t, err)
	auth.Context = ctx

	tx, err := erc20.Transact(auth, "transfer", toAddr, transferAmount)
	require.NoError(t, err)

	receipt, err := bind.WaitMined(ctx, client, tx)
	require.NoError(t, err)
	require.NotNil(t, receipt)
	require.EqualValues(t, 1, receipt.Status, "transfer tx failed")
	t.Logf("transfer tx=%s block=%d", tx.Hash().Hex(), receipt.BlockNumber.Uint64())

	afterOpts := &bind.CallOpts{Context: ctx, BlockNumber: receipt.BlockNumber}
	senderAfter, err := erc20BalanceOf(erc20, afterOpts, senderAddr)
	require.NoError(t, err)
	recipientAfter, err := erc20BalanceOf(erc20, afterOpts, toAddr)
	require.NoError(t, err)
	t.Logf("[after] sender balance=%s recipient balance=%s",
		formatWei18(senderAfter), formatWei18(recipientAfter))

	expectedSender := new(big.Int).Sub(senderBefore, transferAmount)
	require.Equal(t, 0, senderAfter.Cmp(expectedSender),
		"sender balance should decrease by transfer amount: before=%s after=%s transfer=%s",
		senderBefore.String(), senderAfter.String(), transferAmount.String())

	expectedRecipient := new(big.Int).Add(recipientBefore, transferAmount)
	require.Equal(t, 0, recipientAfter.Cmp(expectedRecipient),
		"recipient balance should increase by transfer amount: before=%s after=%s transfer=%s",
		recipientBefore.String(), recipientAfter.String(), transferAmount.String())
}

// TestTokenCollectorWithdraw: admin sends withdraw; Credited.depositAddr must sign (TokenCollector._withdraw).
//
// Signature (see TokenCollector.sol):
//
//	payloadHash = keccak256(abi.encodePacked(depositAddr, amount, playerId))
//	ethSignedHash = keccak256(abi.encodePacked("\x19Ethereum Signed Message:\n32", payloadHash))
func TestTokenCollectorWithdraw(t *testing.T) {
	adminPriv, adminAddr := loadAdminPrivateKey(t)
	userPriv, userAddr := loadUserPrivateKey(t)
	chain := loadE2EChainAddresses(t)
	t.Logf("admin (tx sender): %s", adminAddr.Hex())
	t.Logf("user (E2E key): %s", userAddr.Hex())

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	client, err := ethclient.DialContext(ctx, chain.rpcURL)
	require.NoError(t, err)
	defer client.Close()

	callOpts := &bind.CallOpts{Context: ctx}

	walletManager, err := NewWalletManagerContract(chain.walletManager, client)
	require.NoError(t, err)

	playerID := resolvePlayerID(t)
	playerCollector := resolvePlayerTokenCollector(t, walletManager, callOpts, playerID)

	tokenCollector, err := NewTokenCollectorContract(playerCollector.Address, client)
	require.NoError(t, err)

	requirePlayerIDAllowedForCollector(t, playerID, playerCollector.Address, tokenCollector, callOpts)
	currentIndex, totalWallets := queryDepositWalletWindow(t, tokenCollector, callOpts)
	t.Logf("playerId=%s -> TokenCollector=%s walletIndex=%s currentWalletIndex=%s totalWallets=%s",
		playerID.String(), playerCollector.Address.Hex(), playerCollector.Index.String(),
		currentIndex.String(), totalWallets.String())

	erc20, err := bindERC20(chain.mockERC20, client)
	require.NoError(t, err)

	creditedBefore, err := tokenCollector.Credited(callOpts, playerID)
	require.NoError(t, err)
	creditedBeforeAmt := new(big.Int).Set(creditedBefore.Amount)
	t.Logf("[before] Credited(playerId=%s): depositAddr=%s amount=%s",
		playerID.String(), creditedBefore.DepositAddr.Hex(), formatWei18(creditedBeforeAmt))
	require.True(t, creditedBeforeAmt.Sign() > 0, "no credited balance to withdraw")

	withdrawAmount := resolveWithdrawAmount(t, creditedBeforeAmt)
	require.True(t, creditedBeforeAmt.Cmp(withdrawAmount) >= 0,
		"withdraw amount exceeds credited: credited=%s withdraw=%s",
		formatWei18(creditedBeforeAmt), formatWei18(withdrawAmount))

	depositAddr := creditedBefore.DepositAddr
	require.NotEqual(t, common.Address{}, depositAddr, "Credited.depositAddr is zero")
	require.Equal(t, userAddr, depositAddr,
		"E2E user key must match Credited.depositAddr (signer); have key=%s credited=%s",
		userAddr.Hex(), depositAddr.Hex())

	recipientBefore, err := erc20BalanceOf(erc20, callOpts, depositAddr)
	require.NoError(t, err)
	t.Logf("[before] recipient mock USDT balance (%s): %s", depositAddr.Hex(), formatWei18(recipientBefore))

	sig, err := utils.SignTokenCollectorWithdraw(depositAddr, withdrawAmount, playerID, userPriv)
	require.NoError(t, err)
	t.Logf("withdraw signature: depositAddr signs encodePacked(to,amount,playerId) + EIP-191")

	chainID := big.NewInt(testChainID)
	auth, err := bind.NewKeyedTransactorWithChainID(adminPriv, chainID)
	require.NoError(t, err)
	auth.Context = ctx

	tx, err := tokenCollector.Withdraw(auth, playerID, withdrawAmount, sig)
	require.NoError(t, err)

	receipt, err := bind.WaitMined(ctx, client, tx)
	require.NoError(t, err)
	require.EqualValues(t, 1, receipt.Status, "withdraw tx failed")
	t.Logf("withdraw tx=%s block=%d", tx.Hash().Hex(), receipt.BlockNumber.Uint64())

	afterOpts := &bind.CallOpts{Context: ctx, BlockNumber: receipt.BlockNumber}
	creditedAfter, err := tokenCollector.Credited(afterOpts, playerID)
	require.NoError(t, err)
	t.Logf("[after] Credited amount=%s (at block %d)", formatWei18(creditedAfter.Amount), receipt.BlockNumber.Uint64())

	expectedCredited := new(big.Int).Sub(creditedBeforeAmt, withdrawAmount)
	require.Equal(t, 0, creditedAfter.Amount.Cmp(expectedCredited),
		"Credited.amount should decrease by withdrawAmount: before=%s after=%s withdraw=%s",
		creditedBeforeAmt.String(), creditedAfter.Amount.String(), withdrawAmount.String())

	recipientAfter, err := erc20BalanceOf(erc20, afterOpts, depositAddr)
	require.NoError(t, err)
	t.Logf("[after] recipient mock USDT balance (%s): %s", depositAddr.Hex(), formatWei18(recipientAfter))

	expectedRecipient := new(big.Int).Add(recipientBefore, withdrawAmount)
	require.Equal(t, 0, recipientAfter.Cmp(expectedRecipient),
		"recipient balance should increase by withdrawAmount: before=%s after=%s withdraw=%s",
		recipientBefore.String(), recipientAfter.String(), withdrawAmount.String())
}

// --- helpers ---

func loadE2EChainAddresses(t *testing.T) e2eChainAddresses {
	t.Helper()
	return e2eChainAddresses{
		rpcURL:        getEnvOrDefault(envRPCURL, testRPCURL),
		walletManager: common.HexToAddress(getEnvOrDefault(envWalletManager, testWalletManager)),
		mockERC20:     common.HexToAddress(getEnvOrDefault(envMockERC20, testMockERC20)),
	}
}

func queryPlayerTokenCollector(t *testing.T, wm *WalletManagerContract, opts *bind.CallOpts, playerID *big.Int) playerTokenCollector {
	t.Helper()
	walletIndex, err := wm.GetWalletIndexForPlayerId(opts, playerID)
	require.NoError(t, err, "getWalletIndexForPlayerId(playerId=%s)", playerID.String())

	slot, err := wm.GetWalletSlot(opts, walletIndex)
	require.NoError(t, err, "getWalletSlot(walletIndex=%s)", walletIndex.String())
	require.True(t, slot.Exists, "wallet slot %s does not exist for playerId=%s", walletIndex.String(), playerID.String())
	require.True(t, slot.IsActive, "wallet slot %s is not active for playerId=%s", walletIndex.String(), playerID.String())
	require.NotEqual(t, common.Address{}, slot.CurrentAddress,
		"wallet slot %s has no current address for playerId=%s", walletIndex.String(), playerID.String())

	return playerTokenCollector{Address: slot.CurrentAddress, Index: walletIndex}
}

func resolvePlayerTokenCollector(t *testing.T, wm *WalletManagerContract, opts *bind.CallOpts, playerID *big.Int) playerTokenCollector {
	t.Helper()
	if addr, ok := tokenCollectorOverride(t); ok {
		t.Logf("%s override: using %s (skipping WalletManager lookup)", envTokenCollector, addr.Hex())
		return playerTokenCollector{Address: addr}
	}
	return queryPlayerTokenCollector(t, wm, opts, playerID)
}

func tokenCollectorOverride(t *testing.T) (common.Address, bool) {
	t.Helper()
	raw := strings.TrimSpace(os.Getenv(envTokenCollector))
	if raw == "" {
		return common.Address{}, false
	}
	require.True(t, common.IsHexAddress(raw), "invalid %s=%q", envTokenCollector, raw)
	return common.HexToAddress(raw), true
}

func queryDepositWalletWindow(t *testing.T, tc *TokenCollectorContract, opts *bind.CallOpts) (current, total *big.Int) {
	t.Helper()
	current, err := tc.CurrentWalletIndex(opts)
	require.NoError(t, err)
	total, err = tc.TotalWallets(opts)
	require.NoError(t, err)
	require.True(t, total.Sign() > 0, "TokenCollector.totalWallets is 0")
	return current, total
}

func playerIDWalletIndex(playerID, totalWallets *big.Int) *big.Int {
	return new(big.Int).Mod(new(big.Int).Set(playerID), totalWallets)
}

func playerIDAllowedForWalletWindow(playerID, currentIndex, totalWallets *big.Int) bool {
	return playerIDWalletIndex(playerID, totalWallets).Cmp(currentIndex) == 0
}

func resolvePlayerIDFromEnv(t *testing.T) *big.Int {
	t.Helper()
	raw := strings.TrimSpace(os.Getenv(envPlayerID))
	if raw == "" {
		return nil
	}
	playerID, ok := new(big.Int).SetString(raw, 10)
	require.True(t, ok, "invalid %s", envPlayerID)
	return playerID
}

func resolvePlayerID(t *testing.T) *big.Int {
	t.Helper()
	if playerID := resolvePlayerIDFromEnv(t); playerID != nil {
		return playerID
	}
	t.Logf("%s unset; using playerId=0", envPlayerID)
	return big.NewInt(0)
}

func requirePlayerIDAllowedForCollector(t *testing.T, playerID *big.Int, collectorAddr common.Address, tc *TokenCollectorContract, opts *bind.CallOpts) {
	t.Helper()
	current, total := queryDepositWalletWindow(t, tc, opts)
	require.True(t, playerIDAllowedForWalletWindow(playerID, current, total),
		"E2E_PLAYER_ID=%s not allowed on TokenCollector %s: require playerId %% totalWallets (%s) == currentWalletIndex (%s)",
		playerID.String(), collectorAddr.Hex(), total.String(), current.String())
}

func formatWei18(wei *big.Int) string {
	if wei == nil || wei.Sign() == 0 {
		return "0 × 10^18"
	}
	divisor := new(big.Int).Exp(big.NewInt(10), big.NewInt(tokenDecimals), nil)
	tokens := new(big.Float).Quo(new(big.Float).SetInt(wei), new(big.Float).SetInt(divisor))
	f, _ := tokens.Float64()
	return fmt.Sprintf("%+.4e × 10^18", f)
}

func resolveDepositAmount(t *testing.T, minD *big.Int) *big.Int {
	t.Helper()
	raw := strings.TrimSpace(os.Getenv(envDepositAmount))
	if raw == "" {
		t.Logf("%s unset; using chain minDepositAmount", envDepositAmount)
		return new(big.Int).Set(minD)
	}
	amt, ok := new(big.Int).SetString(raw, 10)
	require.True(t, ok, "invalid %s", envDepositAmount)
	require.True(t, amt.Sign() > 0, "deposit amount must be positive")
	return amt
}

func resolveWithdrawAmount(t *testing.T, credited *big.Int) *big.Int {
	t.Helper()
	raw := strings.TrimSpace(os.Getenv(envWithdrawAmount))
	if raw != "" {
		amt, ok := new(big.Int).SetString(raw, 10)
		require.True(t, ok, "invalid %s", envWithdrawAmount)
		require.True(t, amt.Sign() > 0)
		return amt
	}
	ten := new(big.Int).Mul(big.NewInt(10), new(big.Int).Exp(big.NewInt(10), big.NewInt(18), nil))
	if credited.Cmp(ten) < 0 {
		t.Logf("%s unset; withdrawing full credited balance", envWithdrawAmount)
		return new(big.Int).Set(credited)
	}
	t.Logf("%s unset; withdrawing 10 tokens", envWithdrawAmount)
	return ten
}

func loadRecipientAddress(t *testing.T) common.Address {
	t.Helper()
	raw := strings.TrimSpace(os.Getenv(envToAddress))
	if raw == "" {
		t.Fatalf("missing recipient address: set %s", envToAddress)
	}
	if !common.IsHexAddress(raw) {
		t.Fatalf("invalid %s=%q", envToAddress, raw)
	}
	return common.HexToAddress(raw)
}

func resolveERC20Address(t *testing.T) common.Address {
	t.Helper()
	raw := strings.TrimSpace(os.Getenv(envERC20Address))
	if raw == "" {
		raw = strings.TrimSpace(os.Getenv(envMockERC20))
	}
	if raw == "" {
		raw = testMockERC20
	}
	if !common.IsHexAddress(raw) {
		t.Fatalf("invalid ERC20 address: %s=%q or %s", envERC20Address, raw, envMockERC20)
	}
	return common.HexToAddress(raw)
}

func resolveTransferAmount(t *testing.T) *big.Int {
	t.Helper()
	raw := strings.TrimSpace(os.Getenv(envTransferAmount))
	if raw != "" {
		amt, ok := new(big.Int).SetString(raw, 10)
		require.True(t, ok, "invalid %s", envTransferAmount)
		require.True(t, amt.Sign() > 0, "transfer amount must be positive")
		return amt
	}
	oneToken := new(big.Int).Exp(big.NewInt(10), big.NewInt(tokenDecimals), nil)
	t.Logf("%s unset; transferring 1 token (%s wei)", envTransferAmount, oneToken.String())
	return oneToken
}

func bindERC20(token common.Address, backend bind.ContractBackend) (*bind.BoundContract, error) {
	const erc20ABI = `[
		{"constant":true,"inputs":[{"name":"account","type":"address"}],"name":"balanceOf","outputs":[{"name":"","type":"uint256"}],"stateMutability":"view","type":"function"},
		{"constant":false,"inputs":[{"name":"spender","type":"address"},{"name":"amount","type":"uint256"}],"name":"approve","outputs":[{"name":"","type":"bool"}],"stateMutability":"nonpayable","type":"function"},
		{"constant":false,"inputs":[{"name":"to","type":"address"},{"name":"amount","type":"uint256"}],"name":"transfer","outputs":[{"name":"","type":"bool"}],"stateMutability":"nonpayable","type":"function"}
	]`

	parsed, err := abi.JSON(strings.NewReader(erc20ABI))
	if err != nil {
		return nil, err
	}
	return bind.NewBoundContract(token, parsed, backend, backend, backend), nil
}

func erc20BalanceOf(c *bind.BoundContract, opts *bind.CallOpts, account common.Address) (*big.Int, error) {
	var out []interface{}
	if err := c.Call(opts, &out, "balanceOf", account); err != nil {
		return nil, err
	}
	v := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)
	return v, nil
}

func getEnvOrDefault(key, fallback string) string {
	v := strings.TrimSpace(os.Getenv(key))
	if v == "" {
		return fallback
	}
	return v
}

func loadPrivateKeyHexFromEnv(t *testing.T) string {
	t.Helper()
	keyFile := strings.TrimSpace(os.Getenv(envPrivateKeyFile))
	if keyFile != "" {
		b, err := os.ReadFile(keyFile)
		if err != nil {
			t.Fatalf("failed to read %s=%q: %v", envPrivateKeyFile, keyFile, err)
		}
		return strings.TrimSpace(string(b))
	}
	if hex := strings.TrimSpace(os.Getenv(envPrivateKey)); hex != "" {
		return hex
	}
	t.Fatalf("missing user private key: set %s (path to key file) or %s (hex string)", envPrivateKeyFile, envPrivateKey)
	return ""
}

func loadUserPrivateKey(t *testing.T) (*ecdsa.PrivateKey, common.Address) {
	t.Helper()
	keyHex := loadPrivateKeyHexFromEnv(t)
	priv, err := crypto.HexToECDSA(strings.TrimPrefix(keyHex, "0x"))
	require.NoError(t, err)
	return priv, crypto.PubkeyToAddress(priv.PublicKey)
}

func loadAdminPrivateKey(t *testing.T) (*ecdsa.PrivateKey, common.Address) {
	t.Helper()
	keyPath := strings.TrimSpace(os.Getenv(envAdminKeyFile))
	if keyPath == "" {
		t.Fatalf("missing admin private key: set %s to the path of the admin key file", envAdminKeyFile)
	}
	b, err := os.ReadFile(keyPath)
	if err != nil {
		t.Fatalf("failed to read %s=%q: %v", envAdminKeyFile, keyPath, err)
	}
	priv, err := crypto.HexToECDSA(strings.TrimPrefix(strings.TrimSpace(string(b)), "0x"))
	require.NoError(t, err)
	return priv, crypto.PubkeyToAddress(priv.PublicKey)
}
