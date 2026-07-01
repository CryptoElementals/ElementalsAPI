package tokenunits

import "testing"

func TestRequiresWithdrawAuditDefaultThreshold(t *testing.T) {
	if RequiresWithdrawAudit(100_000, 0) {
		t.Fatal("100_000 should not require audit")
	}
	if !RequiresWithdrawAudit(100_001, 0) {
		t.Fatal("100_001 should require audit")
	}
}

func TestRequiresWithdrawAuditCustomThreshold(t *testing.T) {
	if RequiresWithdrawAudit(50_000, 50_000) {
		t.Fatal("equal to threshold should not require audit")
	}
	if !RequiresWithdrawAudit(50_001, 50_000) {
		t.Fatal("above custom threshold should require audit")
	}
}
