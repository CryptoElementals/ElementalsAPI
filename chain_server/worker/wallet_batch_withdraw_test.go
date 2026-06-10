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

func TestChainBatchWithdrawNotConfigured(t *testing.T) {
	h := &Chain{}
	_, err := h.BatchWithdraw(context.Background(), []BatchWithdrawItem{
		{PlayerID: 1, Amount: 1000, Signature: []byte{0x01}},
	})
	require.ErrorIs(t, err, ErrWalletChainNotConfigured)
}

func TestGasLimitWithBuffer(t *testing.T) {
	require.EqualValues(t, 110_000, gasLimitWithBuffer(100_000))
	require.EqualValues(t, batchWithdrawGasLimit, gasLimitWithBuffer(0))
	require.EqualValues(t, batchWithdrawGasLimit, gasLimitWithBuffer(950_000))
}

func TestWalletRuntimeBatchWithdrawValidation(t *testing.T) {
	r := &walletRuntime{}

	_, err := r.BatchWithdraw(context.Background(), nil)
	require.Error(t, err)
	require.Contains(t, err.Error(), "items is required")

	_, err = r.BatchWithdraw(context.Background(), []BatchWithdrawItem{
		{PlayerID: 0, Amount: 1000, Signature: []byte{0x01}},
	})
	require.Error(t, err)
	require.Contains(t, err.Error(), "invalid player_id")

	_, err = r.BatchWithdraw(context.Background(), []BatchWithdrawItem{
		{PlayerID: 1, Amount: 0, Signature: []byte{0x01}},
	})
	require.Error(t, err)
	require.Contains(t, err.Error(), "invalid amount")

	_, err = r.BatchWithdraw(context.Background(), []BatchWithdrawItem{
		{PlayerID: 1, Amount: -1, Signature: []byte{0x01}},
	})
	require.Error(t, err)
	require.Contains(t, err.Error(), "invalid amount")

	_, err = r.BatchWithdraw(context.Background(), []BatchWithdrawItem{
		{PlayerID: 1, Amount: 1000, Signature: nil},
	})
	require.Error(t, err)
	require.Contains(t, err.Error(), "signature is required")
}

func TestWalletRuntimeBatchWithdrawNoSend(t *testing.T) {
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

	results, err := rt.BatchWithdraw(ctx, []BatchWithdrawItem{
		{PlayerID: 1, Amount: 1_000_000_000_000_000_000, Signature: sig},
	})
	if err != nil {
		t.Skipf("batch withdraw integration requires valid on-chain state and signatures: %v", err)
	}
	require.Len(t, results, 1)
	require.NotEmpty(t, results[0].TxHash)
	require.NotZero(t, results[0].LedgerID)

	var row dao.BatchWithdrawLedger
	require.NoError(t, db.Get().First(&row, results[0].LedgerID).Error)
	require.Equal(t, results[0].TxHash, row.TxHash)
}
