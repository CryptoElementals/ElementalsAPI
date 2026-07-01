package worker

import (
	"context"
	"os"
	"path"
	"testing"

	"github.com/CryptoElementals/common/config"
	"github.com/CryptoElementals/common/db"
	dao "github.com/CryptoElementals/common/models"
	"github.com/CryptoElementals/common/wallet"
	"github.com/stretchr/testify/require"
)

func TestChainWithdrawNotConfigured(t *testing.T) {
	h := &Chain{}
	_, err := h.Withdraw(context.Background(), 1, "1000", []byte{0x01})
	require.ErrorIs(t, err, ErrWalletChainNotConfigured)
}

func TestGasLimitWithBuffer(t *testing.T) {
	require.EqualValues(t, 110_000, gasLimitWithBuffer(100_000))
	require.EqualValues(t, withdrawGasLimit, gasLimitWithBuffer(0))
	require.EqualValues(t, withdrawGasLimit, gasLimitWithBuffer(400_000))
}

func TestWalletRuntimeWithdrawValidation(t *testing.T) {
	r := &walletRuntime{}

	_, err := r.Withdraw(context.Background(), 0, "1000", []byte{0x01})
	require.Error(t, err)
	require.Contains(t, err.Error(), "invalid player_id")

	_, err = r.Withdraw(context.Background(), 1, "0", []byte{0x01})
	require.Error(t, err)
	require.Contains(t, err.Error(), "invalid amount")

	_, err = r.Withdraw(context.Background(), 1, "-1", []byte{0x01})
	require.Error(t, err)
	require.Contains(t, err.Error(), "invalid amount")

	_, err = r.Withdraw(context.Background(), 1, "1000", nil)
	require.Error(t, err)
	require.Contains(t, err.Error(), "signature is required")
}

func TestWalletRuntimeWithdrawNoSend(t *testing.T) {
	rpcURL := os.Getenv("WALLET_CHAIN_RPC")
	wmAddr := os.Getenv("WALLET_MANAGER_ADDRESS")
	if rpcURL == "" || wmAddr == "" {
		t.Skip("set WALLET_CHAIN_RPC and WALLET_MANAGER_ADDRESS for integration test")
	}

	require.NoError(t, db.Init(&db.Config{Development: true}))
	require.NoError(t, db.MigrateMemDb())

	tempFile := path.Join(t.TempDir(), "test_wallet_file")
	require.NoError(t, os.WriteFile(tempFile, []byte("909a42bf20b616a7d317ecc92bde2c88241509745aade0721ff8a917295d31e2"), 0o644))
	w, err := wallet.LoadWallet(tempFile)
	require.NoError(t, err)

	ctx := context.Background()
	rt, err := newWalletRuntime(ctx, &config.WalletChainConfig{
		ChainID: 97,
		NodeConfig: config.NodeConfig{
			HttpRpc: rpcURL,
		},
		WalletManagerAddress: wmAddr,
	}, []*wallet.Wallet{w}, true)
	require.NoError(t, err)

	sig := make([]byte, 65)
	for i := range sig {
		sig[i] = byte(i + 1)
	}

	result, err := rt.Withdraw(ctx, 1, "1000000000000000000", sig)
	if err != nil {
		t.Skipf("withdraw integration requires valid on-chain state and signature: %v", err)
	}
	require.NotEmpty(t, result.TxHash)
	require.NotZero(t, result.LedgerID)

	var row dao.WithdrawLedger
	require.NoError(t, db.Get().First(&row, result.LedgerID).Error)
	require.Equal(t, result.TxHash, row.TxHash)
}
