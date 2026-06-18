package chainamount

import (
	"fmt"
	"math"
	"math/big"
	"strings"
)

const WeiToGameDivisor int64 = 10_000_000_000_000_000 // 10^16

var weiToGameDivisorBig = big.NewInt(WeiToGameDivisor)

// WeiToGameToken converts on-chain wei to game token units via floor division by 10^16.
func WeiToGameToken(amountWei string) (int32, error) {
	raw := strings.TrimSpace(amountWei)
	if raw == "" {
		return 0, fmt.Errorf("amount wei is empty")
	}
	wei, ok := new(big.Int).SetString(raw, 10)
	if !ok {
		return 0, fmt.Errorf("invalid amount wei: %q", amountWei)
	}
	if wei.Sign() <= 0 {
		return 0, fmt.Errorf("amount wei must be positive")
	}
	quotient := new(big.Int).Div(wei, weiToGameDivisorBig)
	if !quotient.IsInt64() || quotient.Int64() > math.MaxInt32 {
		return 0, fmt.Errorf("amount wei overflows int32 game token: %s", quotient.String())
	}
	delta := int32(quotient.Int64())
	if delta <= 0 {
		return 0, fmt.Errorf("amount wei too small after conversion: %s", amountWei)
	}
	return delta, nil
}

// GameTokenToWei converts game token units to on-chain wei (multiply by 10^16).
func GameTokenToWei(tokenAmount int32) (string, error) {
	if tokenAmount <= 0 {
		return "", fmt.Errorf("token amount must be positive")
	}
	wei := new(big.Int).Mul(big.NewInt(int64(tokenAmount)), weiToGameDivisorBig)
	return wei.String(), nil
}

// WeiToGameTokenRemainder returns the remainder after floor division (for logging).
func WeiToGameTokenRemainder(amountWei string) (*big.Int, error) {
	raw := strings.TrimSpace(amountWei)
	wei, ok := new(big.Int).SetString(raw, 10)
	if !ok {
		return nil, fmt.Errorf("invalid amount wei: %q", amountWei)
	}
	remainder := new(big.Int).Mod(wei, weiToGameDivisorBig)
	return remainder, nil
}
