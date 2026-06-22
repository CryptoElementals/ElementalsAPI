package tokenunits

import (
	"fmt"
	"math"
	"math/big"
	"strings"
)

// TokenToWei converts game token units to on-chain wei (smallest USDT unit).
func TokenToWei(tokenAmount int32) (string, error) {
	return DefaultSpec.TokenToWei(tokenAmount)
}

// WeiToToken converts on-chain wei to game token units via floor division.
func WeiToToken(amountWei string) (int32, error) {
	return DefaultSpec.WeiToToken(amountWei)
}

// WeiToTokenRemainder returns the remainder after wei -> token floor division.
func WeiToTokenRemainder(amountWei string) (*big.Int, error) {
	return DefaultSpec.WeiToTokenRemainder(amountWei)
}

// Convert converts amount between units. Remainder is non-zero only for lossy reverse conversions.
func Convert(from, to Unit, amount string) (result string, remainder string, err error) {
	return DefaultSpec.Convert(from, to, amount)
}

func (s Spec) TokenToWei(tokenAmount int32) (string, error) {
	if tokenAmount <= 0 {
		return "", fmt.Errorf("token amount must be positive")
	}
	result, _, err := s.Convert(UnitToken, UnitWei, fmt.Sprintf("%d", tokenAmount))
	return result, err
}

func (s Spec) WeiToToken(amountWei string) (int32, error) {
	result, _, err := s.Convert(UnitWei, UnitToken, amountWei)
	if err != nil {
		return 0, err
	}
	token, ok := new(big.Int).SetString(result, 10)
	if !ok || !token.IsInt64() || token.Int64() > math.MaxInt32 {
		return 0, fmt.Errorf("amount wei overflows int32 game token: %s", result)
	}
	delta := int32(token.Int64())
	if delta <= 0 {
		return 0, fmt.Errorf("amount wei too small after conversion: %s", amountWei)
	}
	return delta, nil
}

func (s Spec) WeiToTokenRemainder(amountWei string) (*big.Int, error) {
	raw := strings.TrimSpace(amountWei)
	wei, ok := new(big.Int).SetString(raw, 10)
	if !ok {
		return nil, fmt.Errorf("invalid amount: %q", amountWei)
	}
	divisor := new(big.Int).Set(s.weiToToken.Div)
	return new(big.Int).Mod(wei, divisor), nil
}

func (s Spec) Convert(from, to Unit, amount string) (result string, remainder string, err error) {
	if from == UnitUnspecified || to == UnitUnspecified {
		return "", "", fmt.Errorf("unit is required")
	}
	raw := strings.TrimSpace(amount)
	if raw == "" {
		return "", "", fmt.Errorf("amount is empty")
	}
	input, err := parseAmount(raw)
	if err != nil {
		return "", "", err
	}
	if input.Sign() <= 0 {
		return "", "", fmt.Errorf("amount must be positive")
	}

	rate, ok := s.rate(from, to)
	if !ok {
		return "", "", fmt.Errorf("unsupported conversion: %s -> %s", from, to)
	}

	quotient, rem := mulDivFloor(input, rate.Mul, rate.Div)
	if from == to {
		rem = big.NewInt(0)
	}
	if quotient.Sign() <= 0 {
		return "", "", fmt.Errorf("amount too small after conversion")
	}
	if to == UnitToken {
		if !quotient.IsInt64() || quotient.Int64() > math.MaxInt32 {
			return "", "", fmt.Errorf("converted amount overflows int32 token: %s", quotient.String())
		}
	}
	return quotient.String(), rem.String(), nil
}

func parseAmount(raw string) (*big.Int, error) {
	value, ok := new(big.Int).SetString(raw, 10)
	if !ok {
		return nil, fmt.Errorf("invalid amount: %q", raw)
	}
	return value, nil
}

func mulDivFloor(amount, mul, div *big.Int) (*big.Int, *big.Int) {
	if div.Sign() == 0 {
		return big.NewInt(0), big.NewInt(0)
	}
	num := new(big.Int).Mul(amount, mul)
	quotient := new(big.Int).Div(num, div)
	remainder := new(big.Int).Mod(num, div)
	return quotient, remainder
}

func (u Unit) String() string {
	switch u {
	case UnitToken:
		return "token"
	case UnitUSDT:
		return "usdt"
	case UnitWei:
		return "wei"
	default:
		return "unspecified"
	}
}

// ParseUnit parses a unit name (case-insensitive).
func ParseUnit(raw string) (Unit, error) {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "token":
		return UnitToken, nil
	case "usdt":
		return UnitUSDT, nil
	case "wei":
		return UnitWei, nil
	default:
		return UnitUnspecified, fmt.Errorf("invalid unit: %q", raw)
	}
}

// ProtoUnit maps protobuf TokenAmountUnit to internal Unit.
func ProtoUnit(v int32) Unit {
	switch v {
	case 1:
		return UnitToken
	case 2:
		return UnitUSDT
	case 3:
		return UnitWei
	default:
		return UnitUnspecified
	}
}

// ToProtoUnit maps internal Unit to protobuf enum value.
func ToProtoUnit(u Unit) int32 {
	switch u {
	case UnitToken:
		return 1
	case UnitUSDT:
		return 2
	case UnitWei:
		return 3
	default:
		return 0
	}
}
