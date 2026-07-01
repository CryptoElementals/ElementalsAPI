package chainclient

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestWithdrawValidatesAmountWei(t *testing.T) {
	c := &Client{}

	_, err := c.Withdraw(t.Context(), 1, "", []byte{0x01})
	require.Error(t, err)
	require.Contains(t, err.Error(), "amount_wei is required")

	_, err = c.Withdraw(t.Context(), 1, "0", []byte{0x01})
	require.Error(t, err)
	require.Contains(t, err.Error(), "invalid amount_wei")

	_, err = c.Withdraw(t.Context(), 1, "9999000000000000000", []byte{0x01})
	require.Error(t, err)
	require.NotContains(t, err.Error(), "overflows int64")
}
