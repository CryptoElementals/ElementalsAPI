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

func TestChainCollectTokenNotConfigured(t *testing.T) {
	h := &Chain{}
	_, err := h.CollectToken(context.Background(), 1, "0x0000000000000000000000000000000000000001", "1000")
	require.ErrorIs(t, err, ErrWalletChainNotConfigured)
}

func TestGasLimitWithBuffer(t *testing.T) {
	require.EqualValues(t, 110_000, gasLimitWithBuffer(100_000))
	require.EqualValues(t, tokenCollectGasLimit, gasLimitWithBuffer(0))
	require.EqualValues(t, tokenCollectGasLimit, gasLimitWithBuffer(280_000))
}

func TestWalletRuntimeCollectTokenValidation(t *testing.T) {
	r := &walletRuntime{}
	_, err := r.CollectToken(context.Background(), 0, "0x0000000000000000000000000000000000000001", "1000")
	require.Error(t, err)
	require.Contains(t, err.Error(), "invalid player_id")

	_, err = r.CollectToken(context.Background(), 1, "not-an-address", "1000")
	require.Error(t, err)
	require.Contains(t, err.Error(), "invalid player_address")

	_, err = r.CollectToken(context.Background(), 1, "0x0000000000000000000000000000000000000001", "0")
	require.Error(t, err)
	require.Contains(t, err.Error(), "invalid token_amount")

	_, err = r.CollectToken(context.Background(), 1, "0x0000000000000000000000000000000000000001", "abc")
	require.Error(t, err)
	require.Contains(t, err.Error(), "invalid token_amount")
}

func TestWalletRuntimeCollectTokenNoSend(t *testing.T) {
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

	playerID := int64(1)
	playerAddr := "0x9156541f2c715810E17c33209767D530978976E5"
	res, err := rt.CollectToken(ctx, playerID, playerAddr, "1000000000000000000")
	require.NoError(t, err)
	require.NotEmpty(t, res.TxHash)
	require.NotZero(t, res.LedgerID)

	var row dao.TokenCollectLedger
	require.NoError(t, db.Get().First(&row, res.LedgerID).Error)
	require.Equal(t, res.TxHash, row.TxHash)
}
