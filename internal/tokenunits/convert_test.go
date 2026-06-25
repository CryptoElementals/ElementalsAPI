package tokenunits

import (
	"math/big"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestConvertAllDirections(t *testing.T) {
	t.Parallel()
	spec := NewSpec()

	t.Run("token_to_usdt", func(t *testing.T) {
		amount, rem, err := spec.Convert(UnitToken, UnitUSDT, "1000")
		require.NoError(t, err)
		require.Equal(t, "0", rem)
		require.Equal(t, "1", amount)
	})

	t.Run("token_to_wei", func(t *testing.T) {
		amount, rem, err := spec.Convert(UnitToken, UnitWei, "1000")
		require.NoError(t, err)
		require.Equal(t, "0", rem)
		require.Equal(t, "1000000000000000000", amount)
	})

	t.Run("usdt_to_wei", func(t *testing.T) {
		amount, rem, err := spec.Convert(UnitUSDT, UnitWei, "1")
		require.NoError(t, err)
		require.Equal(t, "0", rem)
		require.Equal(t, "1000000000000000000", amount)
	})

	t.Run("wei_to_usdt", func(t *testing.T) {
		amount, rem, err := spec.Convert(UnitWei, UnitUSDT, "1000000000000000000")
		require.NoError(t, err)
		require.Equal(t, "0", rem)
		require.Equal(t, "1", amount)
	})

	t.Run("usdt_to_token", func(t *testing.T) {
		amount, rem, err := spec.Convert(UnitUSDT, UnitToken, "1")
		require.NoError(t, err)
		require.Equal(t, "0", rem)
		require.Equal(t, "1000", amount)
	})

	t.Run("wei_to_token", func(t *testing.T) {
		amount, rem, err := spec.Convert(UnitWei, UnitToken, "1000000000000000000")
		require.NoError(t, err)
		require.Equal(t, "0", rem)
		require.Equal(t, "1000", amount)
	})

	t.Run("token_to_usdt_differs_from_token_to_wei", func(t *testing.T) {
		usdt, _, err := spec.Convert(UnitToken, UnitUSDT, "1000")
		require.NoError(t, err)
		wei, _, err := spec.Convert(UnitToken, UnitWei, "1000")
		require.NoError(t, err)
		require.NotEqual(t, usdt, wei)
	})
}

func TestWeiToTokenRemainder(t *testing.T) {
	t.Parallel()
	base := new(big.Int).Exp(big.NewInt(10), big.NewInt(18), nil)
	remainderWei := new(big.Int).Add(base, big.NewInt(1)).String()

	delta, err := WeiToToken(remainderWei)
	require.NoError(t, err)
	require.Equal(t, int32(1000), delta)

	rem, err := WeiToTokenRemainder(remainderWei)
	require.NoError(t, err)
	require.Equal(t, int64(1), rem.Int64())
}

func TestTokenToWeiRoundTrip(t *testing.T) {
	t.Parallel()
	wei, err := TokenToWei(1000)
	require.NoError(t, err)
	require.Equal(t, "1000000000000000000", wei)

	delta, err := WeiToToken(wei)
	require.NoError(t, err)
	require.Equal(t, int32(1000), delta)
}

func TestRatesMatchDerived(t *testing.T) {
	t.Parallel()
	rates := DefaultSpec.Rates()
	require.Equal(t, "1", rates.TokenToUsdt.Mul.String())
	require.Equal(t, "1000", rates.TokenToUsdt.Div.String())
	require.Equal(t, "1000000000000000000", rates.UsdtToWei.Mul.String())
	require.Equal(t, "1", rates.UsdtToWei.Div.String())
	require.Equal(t, "1000000000000000", rates.TokenToWei.Mul.String())
	require.Equal(t, "1", rates.TokenToWei.Div.String())
	require.Equal(t, "1000", rates.UsdtToToken.Mul.String())
	require.Equal(t, "1", rates.UsdtToToken.Div.String())
	require.Equal(t, "1", rates.WeiToUsdt.Mul.String())
	require.Equal(t, "1000000000000000000", rates.WeiToUsdt.Div.String())
}
