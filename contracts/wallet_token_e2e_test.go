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
	testTokenCollector = "0xcc49255a2639560171fc28b09DCd6CdC3b25597C"
	defaultPlayerID    = "1"
	envPrivateKey      = "E2E_PRIVATE_KEY"
	envPrivateKeyFile  = "E2E_PRIVATE_KEY_FILE"
	envAdminKeyFile    = "E2E_ADMIN_KEY_FILE"
	envPlayerID        = "E2E_PLAYER_ID"
	envDepositAmount   = "E2E_DEPOSIT_AMOUNT"
	envWithdrawAmount  = "E2E_WITHDRAW_AMOUNT"
	tokenDecimals      = 18
)

/*
BSC testnet (chain 97) E2E tests: WalletManager + TokenCollector + mock ERC20.

Run from repository root. Use -count=1 to avoid cached results on live chain tests.

Deployed (testnet):
  RPC:             https://data-seed-prebsc-1-s1.bnbchain.org:8545
  MockERC20:       0x59554b201cFc12E6930a3631060C3d9CDF704F67
  WalletManager:   0xFFD251cBd389e482B0609D3B6389a1350827A6C2
  TokenCollector:  0xcc49255a2639560171fc28b09DCd6CdC3b25597C

Environment variables (no default key file paths):
  E2E_PRIVATE_KEY_FILE  Path to depositor/user key file (hex, with or without 0x)
  E2E_PRIVATE_KEY       Depositor key as hex string (alternative to E2E_PRIVATE_KEY_FILE)
  E2E_ADMIN_KEY_FILE    Path to TokenCollector admin key file (required for withdraw tx sender)
  E2E_PLAYER_ID         Player id (default "1"); must satisfy playerId % totalWallets == currentWalletIndex
  E2E_DEPOSIT_AMOUNT    Deposit size in wei; unset uses on-chain minDepositAmount
  E2E_WITHDRAW_AMOUNT   Withdraw size in wei; unset withdraws 10 tokens (or full credited if balance < 10 tokens)

Prerequisites:
  - User address holds enough mock USDT and has approved TokenCollector for deposit.
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

	playerID, ok := new(big.Int).SetString(getEnvOrDefault(envPlayerID, defaultPlayerID), 10)
	require.True(t, ok, "invalid E2E_PLAYER_ID")

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	client, err := ethclient.DialContext(ctx, testRPCURL)
	require.NoError(t, err)
	defer client.Close()

	callOpts := &bind.CallOpts{Context: ctx}

	walletManager, err := NewWalletManagerContract(common.HexToAddress(testWalletManager), client)
	require.NoError(t, err)

	tokenCollector, err := NewTokenCollectorContract(common.HexToAddress(testTokenCollector), client)
	require.NoError(t, err)

	minDeposit, err := tokenCollector.MinDepositAmount(callOpts)
	require.NoError(t, err)
	maxDeposit, err := tokenCollector.MaxDepositAmount(callOpts)
	require.NoError(t, err)
	t.Logf("TokenCollector allowed deposit range: min=%s max=%s", formatWei18(minDeposit), formatWei18(maxDeposit))
	require.True(t, maxDeposit.Cmp(minDeposit) >= 0, "invalid min/max on chain")

	require.True(t, minDeposit.Sign() > 0, "on-chain minDepositAmount is 0; set E2E_DEPOSIT_AMOUNT explicitly")

	depositAmount := resolveDepositAmount(t, minDeposit)

	erc20, err := bindERC20(common.HexToAddress(testMockERC20), client)
	require.NoError(t, err)

	balanceBefore, err := erc20BalanceOf(erc20, callOpts, signerAddr)
	require.NoError(t, err)
	t.Logf("[before] mock USDT balance of signer %s: %s", signerAddr.Hex(), formatWei18(balanceBefore))

	walletInfo, err := walletManager.GetWalletAddr(callOpts, playerID)
	require.NoError(t, err)
	require.NotEqual(t, common.Address{}, walletInfo.Wallet, "wallet address is zero")
	t.Logf("[before] playerId=%s -> stake wallet (WalletManager)=%s walletIndex=%s",
		playerID.String(), walletInfo.Wallet.Hex(), walletInfo.WalletIndex.String())

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

	approveTx, err := erc20.Transact(approveAuth, "approve", common.HexToAddress(testTokenCollector), depositAmount)
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

	if walletInfo.Wallet == common.HexToAddress(testTokenCollector) {
		t.Logf("WalletManager wallet equals TokenCollector contract (walletIndex=%s)",
			walletInfo.WalletIndex.String())
	} else {
		t.Logf("WalletManager wallet=%s (index=%s)", walletInfo.Wallet.Hex(), walletInfo.WalletIndex.String())
	}
	t.Logf("Credited.depositAddr=%s signer(msg.sender)=%s", creditedAfter.DepositAddr.Hex(), signerAddr.Hex())
	require.Equal(t, signerAddr, creditedAfter.DepositAddr,
		"this test sends deposit from signer; Credited.depositAddr should equal msg.sender")
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
	t.Logf("admin (tx sender): %s", adminAddr.Hex())
	t.Logf("user (E2E key): %s", userAddr.Hex())

	playerID, ok := new(big.Int).SetString(getEnvOrDefault(envPlayerID, defaultPlayerID), 10)
	require.True(t, ok, "invalid E2E_PLAYER_ID")

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	client, err := ethclient.DialContext(ctx, testRPCURL)
	require.NoError(t, err)
	defer client.Close()

	callOpts := &bind.CallOpts{Context: ctx}
	tcAddr := common.HexToAddress(testTokenCollector)

	tokenCollector, err := NewTokenCollectorContract(tcAddr, client)
	require.NoError(t, err)

	erc20, err := bindERC20(common.HexToAddress(testMockERC20), client)
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

func bindERC20(token common.Address, backend bind.ContractBackend) (*bind.BoundContract, error) {
	const erc20ABI = `[
		{"constant":true,"inputs":[{"name":"account","type":"address"}],"name":"balanceOf","outputs":[{"name":"","type":"uint256"}],"stateMutability":"view","type":"function"},
		{"constant":false,"inputs":[{"name":"spender","type":"address"},{"name":"amount","type":"uint256"}],"name":"approve","outputs":[{"name":"","type":"bool"}],"stateMutability":"nonpayable","type":"function"}
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
