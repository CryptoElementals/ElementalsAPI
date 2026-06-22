package tokenunits

import "math/big"

const (
	// token -> usdt (smallest): amount * TokenToUsdtMul * UsdtToWeiMul / (TokenToUsdtDenom * UsdtToWeiDenom)
	// 1 Token = TokenToUsdtMul / TokenToUsdtDenom USDT.
	TokenToUsdtMul   int64 = 1
	TokenToUsdtDenom int64 = 100

	// usdt -> wei: amount * UsdtToWeiMul / UsdtToWeiDenom
	// 1 USDT = UsdtToWeiMul / UsdtToWeiDenom wei (smallest on-chain unit).
	UsdtToWeiMul   int64 = 1_000_000_000_000_000_000 // 10^18
	UsdtToWeiDenom int64 = 1

	// token -> wei derived: TokenToUsdtMul * UsdtToWeiMul / (TokenToUsdtDenom * UsdtToWeiDenom)
	TokenToWeiMul int64 = TokenToUsdtMul * UsdtToWeiMul / UsdtToWeiDenom / TokenToUsdtDenom // 10^16
)

// Unit identifies token amount denomination.
type Unit int

const (
	UnitUnspecified Unit = iota
	UnitToken
	UnitUSDT
	UnitWei
)

// MulDiv is a rational conversion factor: result = floor(amount * Mul / Div).
type MulDiv struct {
	Mul *big.Int
	Div *big.Int
}

// Rates holds derived mul/div pairs for all supported conversion directions.
type Rates struct {
	TokenToUsdt MulDiv
	UsdtToWei   MulDiv
	TokenToWei  MulDiv
	UsdtToToken MulDiv
	WeiToUsdt   MulDiv
	WeiToToken  MulDiv
}

// Spec is the hard-coded token/USDT/wei conversion specification.
type Spec struct {
	tokenToUsdt MulDiv
	usdtToWei   MulDiv
	tokenToWei  MulDiv
	usdtToToken MulDiv
	weiToUsdt   MulDiv
	weiToToken  MulDiv
}

// DefaultSpec is the process-wide conversion spec.
var DefaultSpec = NewSpec()

func NewSpec() Spec {
	tokenToUsdt := MulDiv{
		Mul: big.NewInt(TokenToUsdtMul * UsdtToWeiMul / UsdtToWeiDenom),
		Div: big.NewInt(TokenToUsdtDenom),
	}
	tokenToWei := MulDiv{
		Mul: big.NewInt(TokenToWeiMul),
		Div: big.NewInt(1),
	}
	usdtToWei := MulDiv{Mul: big.NewInt(1), Div: big.NewInt(1)}
	usdtToToken := MulDiv{
		Mul: big.NewInt(TokenToUsdtDenom * UsdtToWeiDenom),
		Div: big.NewInt(TokenToUsdtMul * UsdtToWeiMul),
	}
	weiToUsdt := usdtToWei
	weiToToken := MulDiv{
		Mul: big.NewInt(1),
		Div: big.NewInt(TokenToWeiMul),
	}

	return Spec{
		tokenToUsdt: tokenToUsdt,
		usdtToWei:   usdtToWei,
		tokenToWei:  tokenToWei,
		usdtToToken: usdtToToken,
		weiToUsdt:   weiToUsdt,
		weiToToken:  weiToToken,
	}
}


func (s Spec) Rates() Rates {
	return Rates{
		TokenToUsdt: cloneMulDiv(s.tokenToUsdt),
		UsdtToWei:   cloneMulDiv(s.usdtToWei),
		TokenToWei:  cloneMulDiv(s.tokenToWei),
		UsdtToToken: cloneMulDiv(s.usdtToToken),
		WeiToUsdt:   cloneMulDiv(s.weiToUsdt),
		WeiToToken:  cloneMulDiv(s.weiToToken),
	}
}

func (s Spec) rate(from, to Unit) (MulDiv, bool) {
	if from == to {
		return MulDiv{Mul: big.NewInt(1), Div: big.NewInt(1)}, true
	}
	switch {
	case from == UnitToken && to == UnitUSDT:
		return s.tokenToUsdt, true
	case from == UnitToken && to == UnitWei:
		return s.tokenToWei, true
	case from == UnitUSDT && to == UnitToken:
		return s.usdtToToken, true
	case from == UnitUSDT && to == UnitWei:
		return s.usdtToWei, true
	case from == UnitWei && to == UnitUSDT:
		return s.weiToUsdt, true
	case from == UnitWei && to == UnitToken:
		return s.weiToToken, true
	default:
		return MulDiv{}, false
	}
}

func cloneMulDiv(rate MulDiv) MulDiv {
	return MulDiv{Mul: new(big.Int).Set(rate.Mul), Div: new(big.Int).Set(rate.Div)}
}

// MulDivStrings returns mul/div as decimal strings for API responses.
func MulDivStrings(rate MulDiv) (mul, div string) {
	return rate.Mul.String(), rate.Div.String()
}
