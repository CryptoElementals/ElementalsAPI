package tokenunits

import "fmt"

// MaxWithdrawTokenAmount is the per-request withdraw cap in game token units.
const MaxWithdrawTokenAmount int32 = 100_000_000

// ValidateWithdrawTokenAmount checks a withdraw request token amount.
func ValidateWithdrawTokenAmount(tokenAmount int32) error {
	if tokenAmount <= 0 {
		return fmt.Errorf("token_amount is required")
	}
	if tokenAmount > MaxWithdrawTokenAmount {
		return fmt.Errorf("token_amount exceeds max withdraw limit %d", MaxWithdrawTokenAmount)
	}
	return nil
}
