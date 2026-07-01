package tokenunits

import "fmt"

// MaxWithdrawTokenAmount is the per-request withdraw cap in game token units.
const MaxWithdrawTokenAmount int32 = 100_000_000

// DefaultWithdrawAuditThresholdTokens is the default per-request token amount above which
// withdraw enters manual audit (>100 USDT at 1000 tokens per USDT).
const DefaultWithdrawAuditThresholdTokens int32 = 100_000

// ResolveWithdrawAuditThreshold returns thresholdTokens or the default when unset.
func ResolveWithdrawAuditThreshold(thresholdTokens int32) int32 {
	if thresholdTokens <= 0 {
		return DefaultWithdrawAuditThresholdTokens
	}
	return thresholdTokens
}

// RequiresWithdrawAudit reports whether tokenAmount must enter manual audit.
func RequiresWithdrawAudit(tokenAmount, thresholdTokens int32) bool {
	return tokenAmount > ResolveWithdrawAuditThreshold(thresholdTokens)
}

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
