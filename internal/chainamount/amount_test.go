package chainamount

import (
	"math/big"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestWeiToGameToken(t *testing.T) {
	t.Parallel()

	oneTokenWei := new(big.Int).Exp(big.NewInt(10), big.NewInt(18), nil).String()
	delta, err := WeiToGameToken(oneTokenWei)
	require.NoError(t, err)
	require.Equal(t, int32(100), delta)

	_, err = WeiToGameToken("0")
	require.Error(t, err)

	_, err = WeiToGameToken("not-a-number")
	require.Error(t, err)

	base, ok := new(big.Int).SetString(oneTokenWei, 10)
	require.True(t, ok)
	remainderWei := new(big.Int).Add(base, big.NewInt(1)).String()
	delta, err = WeiToGameToken(remainderWei)
	require.NoError(t, err)
	require.Equal(t, int32(100), delta)

	rem, err := WeiToGameTokenRemainder(remainderWei)
	require.NoError(t, err)
	require.Equal(t, int64(1), rem.Int64())
}

func TestWeiToGameTokenTooSmall(t *testing.T) {
	t.Parallel()
	_, err := WeiToGameToken("1")
	require.Error(t, err)
}

func TestGameTokenToWei(t *testing.T) {
	t.Parallel()
	wei, err := GameTokenToWei(1000)
	require.NoError(t, err)
	require.Equal(t, "10000000000000000000", wei)

	delta, err := WeiToGameToken(wei)
	require.NoError(t, err)
	require.Equal(t, int32(1000), delta)
}
