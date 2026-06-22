package ledgerserver

import (
	"github.com/CryptoElementals/common/internal/tokenunits"
	"github.com/CryptoElementals/common/rpc/proto"
)

func mulDivRateToProto(rate tokenunits.MulDiv) *proto.MulDivRate {
	mul, div := tokenunits.MulDivStrings(rate)
	return &proto.MulDivRate{Mul: mul, Div: div}
}

func tokenUnitRatesToProto(rates tokenunits.Rates) *proto.GetTokenUnitRatesResponse {
	return &proto.GetTokenUnitRatesResponse{
		TokenToUsdt: mulDivRateToProto(rates.TokenToUsdt),
		UsdtToWei:   mulDivRateToProto(rates.UsdtToWei),
		TokenToWei:  mulDivRateToProto(rates.TokenToWei),
		UsdtToToken: mulDivRateToProto(rates.UsdtToToken),
		WeiToUsdt:   mulDivRateToProto(rates.WeiToUsdt),
		WeiToToken:  mulDivRateToProto(rates.WeiToToken),
	}
}
