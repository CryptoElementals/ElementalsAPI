package config

import (
	"testing"

	"github.com/CryptoElementals/common/internal/tokenunits"
)

func TestResolvedWithdrawAuditThresholdDefault(t *testing.T) {
	cfg := &LedgerServerConfig{}
	setLedgerServerDefaults(cfg)
	if got := cfg.ResolvedWithdrawAuditThreshold(); got != tokenunits.DefaultWithdrawAuditThresholdTokens {
		t.Fatalf("expected default %d, got %d", tokenunits.DefaultWithdrawAuditThresholdTokens, got)
	}
}

func TestResolvedWithdrawAuditThresholdCustom(t *testing.T) {
	cfg := &LedgerServerConfig{WithdrawAuditThresholdTokens: 50_000}
	if got := cfg.ResolvedWithdrawAuditThreshold(); got != 50_000 {
		t.Fatalf("expected 50000, got %d", got)
	}
}
