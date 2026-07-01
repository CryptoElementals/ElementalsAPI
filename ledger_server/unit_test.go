package ledgerserver

import (
	"testing"

	"github.com/CryptoElementals/common/rpc/proto"
	"github.com/stretchr/testify/require"
)

func TestGetTokenUnitRates(t *testing.T) {
	svc := NewService(nil, nil, 0, 0)
	resp := svc.GetTokenUnitRates()
	require.NotNil(t, resp.GetTokenToUsdt())
	require.Equal(t, "1", resp.GetTokenToUsdt().GetMul())
	require.Equal(t, "1000", resp.GetTokenToUsdt().GetDiv())
	require.Equal(t, "1000000000000000000", resp.GetUsdtToWei().GetMul())
	require.Equal(t, "1", resp.GetUsdtToWei().GetDiv())
	require.Equal(t, "1000000000000000", resp.GetTokenToWei().GetMul())
}

func TestConvertTokenAmount(t *testing.T) {
	svc := NewService(nil, nil, 0, 0)
	resp, err := svc.ConvertTokenAmount(&proto.ConvertTokenAmountRequest{
		FromUnit: proto.TokenAmountUnit_TOKEN_AMOUNT_UNIT_TOKEN,
		ToUnit:   proto.TokenAmountUnit_TOKEN_AMOUNT_UNIT_WEI,
		Amount:   "1000",
	})
	require.NoError(t, err)
	require.Equal(t, "1000000000000000000", resp.GetAmount())
	require.Equal(t, "0", resp.GetRemainder())
}

func TestConvertTokenAmountInvalidUnit(t *testing.T) {
	svc := NewService(nil, nil, 0, 0)
	_, err := svc.ConvertTokenAmount(&proto.ConvertTokenAmountRequest{
		FromUnit: proto.TokenAmountUnit_TOKEN_AMOUNT_UNIT_UNSPECIFIED,
		ToUnit:   proto.TokenAmountUnit_TOKEN_AMOUNT_UNIT_WEI,
		Amount:   "1",
	})
	require.Error(t, err)
}
