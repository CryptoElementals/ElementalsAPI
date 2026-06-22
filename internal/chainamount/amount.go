package chainamount

import (
	"math/big"

	"github.com/CryptoElementals/common/internal/tokenunits"
)

// WeiToGameDivisor is 10^16: 1 game token = 0.01 USDT = 10^16 wei.
const WeiToGameDivisor int64 = tokenunits.TokenToWeiMul

// WeiToGameToken converts on-chain wei to game token units.
func WeiToGameToken(amountWei string) (int32, error) {
	return tokenunits.WeiToToken(amountWei)
}

// GameTokenToWei converts game token units to on-chain wei.
func GameTokenToWei(tokenAmount int32) (string, error) {
	return tokenunits.TokenToWei(tokenAmount)
}

// WeiToGameTokenRemainder returns the remainder after floor division (for logging).
func WeiToGameTokenRemainder(amountWei string) (*big.Int, error) {
	return tokenunits.WeiToTokenRemainder(amountWei)
}
